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

package tests

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	helmrepo "github.com/soer3n/apps-operator/controllers/helm"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var namespace string
var ns *v1.Namespace
var err error
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	logf.Log.Info("namespace:", "namespace", namespace)
	Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())
	Expect(os.Setenv("WATCH_NAMESPACE", namespace)).To(Succeed())
	//Expect(os.Setenv("TEST_ASSET_KUBE_APISERVER", "/opt/kubebuilder/testbin/bin/kube-apiserver")).To(Succeed())
	//Expect(os.Setenv("TEST_ASSET_ETCD", "/opt/kubebuilder/testbin/bin/etcd")).To(Succeed())
	//Expect(os.Setenv("TEST_ASSET_KUBECTL", "/opt/kubebuilder/testbin/bin/kubectl")).To(Succeed())
	// Expect(os.Setenv("CGO_ENABLED", "0")).To(Succeed())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Namespace: ns.Name, MetricsBindAddress: "0"})
	Expect(err).NotTo(HaveOccurred(), "failed to create manager")

	controller := &helmrepo.RepoReconciler{
		Client:   mgr.GetClient(),
		Log:      logf.Log,
		Recorder: mgr.GetEventRecorderFor("repo-controller"),
		Scheme:   mgr.GetScheme(),
	}
	err = controller.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup controller")

	//repoGroupMgr, err := ctrl.NewManager(cfg, ctrl.Options{Namespace: ns.Name, MetricsBindAddress: "0"})
	//Expect(err).NotTo(HaveOccurred(), "failed to create manager")

	repoGroupController := &helmrepo.RepoGroupReconciler{
		Client: mgr.GetClient(),
		Log:    logf.Log,
		// Recorder: mgr.GetEventRecorderFor("repo-controller"),
		Scheme: mgr.GetScheme(),
	}
	err = repoGroupController.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup repogroup controller")

	releaseGroupController := &helmrepo.ReleaseGroupReconciler{
		Client: mgr.GetClient(),
		Log:    logf.Log,
		Scheme: mgr.GetScheme(),
	}
	err = releaseGroupController.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup release group controller")

	releaseController := &helmrepo.ReleaseReconciler{
		Client: mgr.GetClient(),
		Log:    logf.Log,
		Scheme: mgr.GetScheme(),
	}
	err = releaseController.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred(), "failed to setup release controller")

	go func() {
		err := mgr.Start(context.TODO())
		Expect(err).NotTo(HaveOccurred(), "failed to start repogroup manager")
	}()

	ns = &core.Namespace{}
	*ns = core.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-" + randStringRunes(5)},
	}

	err = k8sClient.Create(context.TODO(), ns)
	Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

}, 60)

var _ = AfterSuite(func() {
	err = k8sClient.Delete(context.TODO(), ns)
	Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func SetupTest(ctx context.Context) *core.Namespace {
	var stopCh chan struct{}
	ns = &core.Namespace{}
	repo := &helmv1alpha1.Repo{}
	repo.ObjectMeta.Name = "testresource"

	BeforeEach(func() {
		stopCh = make(chan struct{})
	})

	AfterEach(func() {
		close(stopCh)
	})

	return ns
}

func PrepareReleaseTest(ctx context.Context) {
	testRepo := &helmv1alpha1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testresource",
			Namespace: ns.Name,
		},
		Spec: helmv1alpha1.RepoSpec{
			Name: "deployment-name",
			Url:  "https://submariner-io.github.io/submariner-charts/charts",
		},
	}

	namespace = ns.ObjectMeta.Name
	err = k8sClient.Create(context.TODO(), testRepo)
	Expect(err).NotTo(HaveOccurred(), "failed to delete test repo")
	namespace = ns.ObjectMeta.Name
}

func CleanUpReleaseTest(ctx context.Context) {
	testRepo := &helmv1alpha1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testresource",
			Namespace: ns.Name,
		},
		Spec: helmv1alpha1.RepoSpec{
			Name: "deployment-name",
			Url:  "https://submariner-io.github.io/submariner-charts/charts",
		},
	}

	namespace = ns.ObjectMeta.Name
	testRepo.ObjectMeta.Namespace = ns.ObjectMeta.Name
	err = k8sClient.Delete(context.TODO(), testRepo)
	Expect(err).NotTo(HaveOccurred(), "failed to delete test repo")
}

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
