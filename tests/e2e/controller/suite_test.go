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
	"crypto/rand"
	"math/big"
	mr "math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	controllers "github.com/soer3n/yaho/controllers/helm"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg                   *rest.Config
	k8sClient, testClient client.Client
)

var (
	err     error
	testEnv *envtest.Environment
)

const (
	testRepoName                  = "testresource"
	testRepoURL                   = "https://soer3n.github.io/charts/testing_a"
	testRepoNameSecond            = "testresource-2"
	testRepoURLSecond             = "https://soer3n.github.io/charts/testing_b"
	testRepoChartNameAssert       = "testing"
	testRepoChartSecondNameAssert = "testing-dep"
)

const (
	testReleaseName               = "testresource"
	testReleaseChartName          = "testing"
	testReleaseChartVersion       = "0.1.0"
	testReleaseNameSecond         = "testresource-2"
	testReleaseChartNameSecond    = "testing-dep"
	testReleaseChartVersionSecond = "0.1.0"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t,
		"Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	// Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "/opt/kubebuilder/testbin/bin/kube-apiserver")).To(Succeed())
	// Expect(os.Setenv("TEST_ASSET_ETCD", "/opt/kubebuilder/testbin/bin/etcd")).To(Succeed())
	// Expect(os.Setenv("TEST_ASSET_KUBECTL", "/opt/kubebuilder/testbin/bin/kubectl")).To(Succeed())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
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

	logf.Log.Info("namespace:", "namespace", "default")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred(), "failed to create manager")

	err = (&controllers.RepoReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Recorder: mgr.GetEventRecorderFor("repo-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup controller")

	err = (&controllers.RepoGroupReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Recorder: mgr.GetEventRecorderFor("repogroup-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup repogroup controller")

	err = (&controllers.ChartReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("charts-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup repogroup controller")

	err = (&controllers.ReleaseGroupReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("releasegroup-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup release group controller")

	err = (&controllers.ReleaseReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("release-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup release controller")

	err = (&controllers.ValuesReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("values-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup values controller")

	go func() {
		defer GinkgoRecover()
		err := mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).NotTo(HaveOccurred(), "failed to start helm manager")
	}()

	testClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
}, 60)

var _ = AfterSuite(func() {
	// close(stopCh)
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func init() {
	mr.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		n, _ := rand.Int(rand.Reader, (big.NewInt(30)))
		b[i] = letterRunes[n.Uint64()]
	}
	return string(b)
}
