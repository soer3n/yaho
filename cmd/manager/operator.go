/*
Copyright 2021.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"flag"
	"log"
	"os"

	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	helmcontrollers "github.com/soer3n/yaho/controllers/helm"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	command := newRootCmd()

	if err := command.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manager",
		Short: "manager app",
		Long:  `manager app`,
	}

	cmd.AddCommand(newOperatorCmd())
	return cmd
}

func newOperatorCmd() *cobra.Command {

	var metricsAddr string
	var isLocal bool
	var enableLeaderElection bool
	var probeAddr string
	var configFile string

	cmd := &cobra.Command{
		Use:   "operator",
		Short: "runs the operator",
		Long:  `apps operator`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ = cmd.Flags().GetString("config")
			metricsAddr, _ = cmd.Flags().GetString("metrics-bind-address")
			probeAddr, _ = cmd.Flags().GetString("health-probe-bind-address")
			enableLeaderElection, _ = cmd.Flags().GetBool("leader-elect")
			isLocal, _ = cmd.Flags().GetBool("is-local")
			run(configFile, isLocal, metricsAddr, probeAddr, enableLeaderElection)
		},
	}

	cmd.PersistentFlags().String("config", "", "The controller will load its initial configuration from this file. "+
		"Omit this flag to use the default configuration values. "+
		"Command-line flags override configuration from this file.")
	cmd.PersistentFlags().String("metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	cmd.PersistentFlags().String("health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	cmd.PersistentFlags().Bool("leader-elect", false, "Enable leader election for controller manager. "+
		"Enabling this will ensure there is only one active controller manager.")
	cmd.PersistentFlags().Bool("is-local", false, "if true sets the k8s api server url to 127.0.0.1:6443, else to cluster domain")

	return cmd
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(helmv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func run(configFile string, isLocal bool, metricsAddr, probeAddr string, enableLeaderElection bool) {

	var err error

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// set default options
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "bb07b8f2.soer3n.dev",
	}

	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	ns := getWatchNamespace()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	config := mgr.GetConfig()
	rc, err := client.NewWithWatch(config, client.Options{Scheme: mgr.GetScheme(), Mapper: mgr.GetRESTMapper()})

	if err != nil {
		setupLog.Error(err, "failed to setup rest client")
		os.Exit(1)
	}

	if err = (&helmcontrollers.RepoReconciler{
		Client:         rc,
		WatchNamespace: ns,
		Log:            ctrl.Log.WithName("controllers").WithName("helm").WithName("Repo"),
		Scheme:         mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Repo")
		os.Exit(1)
	}
	if err = (&helmcontrollers.RepoGroupReconciler{
		Client: rc,
		Log:    ctrl.Log.WithName("controllers").WithName("helm").WithName("RepoGroup"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RepoGroup")
		os.Exit(1)
	}
	if err = (&helmcontrollers.ReleaseReconciler{
		WithWatch:      rc,
		WatchNamespace: ns,
		IsLocal:        isLocal,
		Log:            ctrl.Log.WithName("controllers").WithName("helm").WithName("Release"),
		Scheme:         mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Release")
		os.Exit(1)
	}
	if err = (&helmcontrollers.ReleaseGroupReconciler{
		Client:         rc,
		WatchNamespace: ns,
		Log:            ctrl.Log.WithName("controllers").WithName("helm").WithName("ReleaseGroup"),
		Scheme:         mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ReleaseGroup")
		os.Exit(1)
	}
	if err = (&helmcontrollers.ChartReconciler{
		WithWatch:      rc,
		WatchNamespace: ns,
		Log:            ctrl.Log.WithName("controllers").WithName("helm").WithName("Chart"),
		Scheme:         mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Chart")
		os.Exit(1)
	}
	if err = (&helmcontrollers.ValuesReconciler{
		Client: rc,
		Log:    ctrl.Log.WithName("controllers").WithName("helm").WithName("Values"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Values")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func getWatchNamespace() string {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	watchNamespaceEnvVar := "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		ctrl.Log.WithName("setup").Info("watched namespace not set, using default.", "namespace", ns)
		return "default"
	}
	ctrl.Log.WithName("setup").Info("watched namespace for configmaps.", "namespace", ns)
	return ns
}
