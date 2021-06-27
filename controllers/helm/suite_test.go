/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helm

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var stopCh context.Context

//var namespace string

// var ns *v1.Namespace
var err error
var testEnv *envtest.Environment

//var repoNeeded bool

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	// Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "/opt/kubebuilder/testbin/bin/kube-apiserver")).To(Succeed())
	// Expect(os.Setenv("TEST_ASSET_ETCD", "/opt/kubebuilder/testbin/bin/etcd")).To(Succeed())
	// Expect(os.Setenv("TEST_ASSET_KUBECTL", "/opt/kubebuilder/testbin/bin/kubectl")).To(Succeed())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	// stopCh = ctrl.SetupSignalHandler()
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	//Expect(os.Setenv("WATCH_NAMESPACE", "")).To(Succeed())
	logf.Log.Info("namespace:", "namespace", "default")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred(), "failed to create manager")

	err = (&RepoReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Recorder: mgr.GetEventRecorderFor("repo-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup controller")

	//repoGroupMgr, err := ctrl.NewManager(cfg, ctrl.Options{Namespace: ns.Name, MetricsBindAddress: "0"})
	//Expect(err).NotTo(HaveOccurred(), "failed to create manager")

	err = (&RepoGroupReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Recorder: mgr.GetEventRecorderFor("repogroup-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup repogroup controller")

	err = (&ChartReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("charts-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup repogroup controller")

	err = (&ReleaseGroupReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("releasegroup-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup release group controller")

	err = (&ReleaseReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("release-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup release controller")

	err = (&ValuesReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("values-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup values controller")

	go func() {
		err := mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).NotTo(HaveOccurred(), "failed to start helm manager")
	}()

	close(done)
}, 60)

var _ = AfterSuite(func() {
	//close(stopCh)
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
