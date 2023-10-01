package cmd

import (
	"flag"
	"os"

	helmcontrollers "github.com/soer3n/yaho/controllers/agent"
	"github.com/soer3n/yaho/internal/utils"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func NewAgentCmd(scheme *runtime.Scheme) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "agent subcommands",
		Long:  `agent subcommands`,
	}

	cmd.AddCommand(newAgentRunCmd(scheme))
	cmd.AddCommand(newAgentKubeconfigCmd(scheme))
	return cmd
}

func newAgentKubeconfigCmd(scheme *runtime.Scheme) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "parse and store agent kubeconfig",
		Long:  `parse and store agent kubeconfig`,
		Run: func(cmd *cobra.Command, args []string) {
		},
	}
	return cmd
}

func newAgentRunCmd(scheme *runtime.Scheme) *cobra.Command {

	var metricsAddr string
	var isLocal bool
	var enableLeaderElection bool
	var probeAddr string
	var configFile string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "runs the agent",
		Long:  `runs the agent`,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, _ = cmd.Flags().GetString("config")
			metricsAddr, _ = cmd.Flags().GetString("metrics-bind-address")
			probeAddr, _ = cmd.Flags().GetString("health-probe-bind-address")
			enableLeaderElection, _ = cmd.Flags().GetBool("leader-elect")
			isLocal, _ = cmd.Flags().GetBool("is-local")
			runAgent(scheme, configFile, isLocal, metricsAddr, probeAddr, enableLeaderElection)
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

func runAgent(scheme *runtime.Scheme, configFile string, isLocal bool, metricsAddr, probeAddr string, enableLeaderElection bool) {

	var err error

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// set default options
	options, err := utils.ManagerOptions(configFile)
	options.Scheme = scheme

	if err != nil {
		setupLog.Error(err, "unable to load the config file")
		os.Exit(1)
	}

	ns := getWatchNamespace()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), *options)
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
