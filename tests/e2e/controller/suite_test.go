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
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	mr "math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	controllers "github.com/soer3n/yaho/controllers/helm"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
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
	testRepoName                         = "testresource"
	testRepoURL                          = "https://soer3n.github.io/charts/testing_a"
	testRepoNameSecond                   = "testresource-2"
	testRepoURLSecond                    = "https://soer3n.github.io/charts/testing_b"
	testRepoChartNameAssert              = "testing"
	testRepoChartNameAssertqVersion      = "0.1.1"
	testRepoChartSecondNameAssert        = "testing-dep"
	testRepoChartSecondNameAssertVersion = "0.1.1"
	testRepoChartThirdNameAssert         = "testing-nested"
	testRepoChartThirdNameAssertVersion  = "0.1.0"
	testRepoChartNotValidVersion         = "9.9.9"
)

const (
	testReleaseName                        = "testresource"
	testReleaseChartName                   = "testing"
	testReleaseChartVersion                = "0.1.1"
	testReleaseNameSecond                  = "testresource-2"
	testReleaseChartNameSecond             = "testing-dep"
	testReleaseChartVersionSecond          = "0.1.1"
	testReleaseChartThirdNameAssert        = "testing-nested"
	testReleaseChartThirdNameAssertVersion = "0.1.0"
	testReleaseChartNotValidVersion        = "9.9.9"
)

type RepositoryAssert struct {
	Name            string
	Obj             *helmv1alpha1.Repository
	IsPresent       bool
	InstalledCharts int64
	Status          types.GomegaMatcher
	Synced          types.GomegaMatcher
	ManagedCharts   []*ChartAssert
}

type ChartAssert struct {
	Name               string
	Obj                *helmv1alpha1.Chart
	Version            string
	IsPresent          types.GomegaMatcher
	IndicesInstalled   types.GomegaMatcher
	ResourcesInstalled types.GomegaMatcher
	Synced             types.GomegaMatcher
	Deps               types.GomegaMatcher
}

type ReleaseAssert struct {
	Name      string
	Obj       *helmv1alpha1.Release
	IsPresent bool
	Revision  int
	Synced    types.GomegaMatcher
	Status    string
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t,
		"Controller Suite")
}

func setupNamespace() *v1.Namespace {

	ch := context.TODO()
	stopCh, cancelFn := context.WithCancel(ch)
	ns := "test-" + randStringRunes(7)

	BeforeEach(func() {

		logf.Log.Info("namespace:", "namespace", ns)

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme, MetricsBindAddress: "0", HealthProbeBindAddress: "0"})
		Expect(err).NotTo(HaveOccurred(), "failed to create manager")

		config := mgr.GetConfig()
		rc, err := client.NewWithWatch(config, client.Options{Scheme: mgr.GetScheme(), Mapper: mgr.GetRESTMapper()})

		if err != nil {
			logf.Log.Error(err, "failed to setup rest client")
			os.Exit(1)
		}

		err = (&controllers.RepoReconciler{
			Client:         mgr.GetClient(),
			WatchNamespace: ns,
			Log:            logf.Log,
			Recorder:       mgr.GetEventRecorderFor("repo-controller"),
			Scheme:         mgr.GetScheme(),
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
			WithWatch:      rc,
			WatchNamespace: ns,
			Log:            logf.Log,
			Scheme:         mgr.GetScheme(),
			Recorder:       mgr.GetEventRecorderFor("charts-controller"),
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred(), "failed to setup repogroup controller")

		err = (&controllers.ReleaseGroupReconciler{
			Client:         mgr.GetClient(),
			WatchNamespace: ns,
			Log:            logf.Log,
			Scheme:         mgr.GetScheme(),
			Recorder:       mgr.GetEventRecorderFor("releasegroup-controller"),
		}).SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred(), "failed to setup release group controller")

		err = (&controllers.ReleaseReconciler{
			WithWatch:      rc,
			WatchNamespace: ns,
			IsLocal:        true,
			Log:            logf.Log,
			Scheme:         mgr.GetScheme(),
			Recorder:       mgr.GetEventRecorderFor("release-controller"),
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
			err := mgr.Start(stopCh)
			Expect(err).NotTo(HaveOccurred(), "failed to start helm manager")
		}()

	}, 60)

	AfterEach(func() {
		cancelFn()
	}, 60)

	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
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

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// +kubebuilder:scaffold:scheme

	err = helmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	testClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
}, 60)

var _ = AfterSuite(func() {
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

func GetRepositoryFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repository) func() error {
	return func() error {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return err
		}

		return nil
	}
}

func GetRepositoryStatusFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repository) func() bool {
	return func() bool {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return false
		}

		if obj.Status.Synced == nil {
			return false
		}

		if *obj.Status.Synced {
			return true
		}

		return false
	}
}

func GetRepositoryCountFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repository) func() *int64 {
	return func() *int64 {
		if err := testClient.Get(ctx, key, obj); err != nil {
			l := int64(0)
			return &l
		}

		return obj.Status.Charts
	}
}

func GetChartFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Chart) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}

func GetChartSyncedStatusFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Chart) func() bool {
	return func() bool {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return false
		}

		if obj.Status.Versions == nil {
			return false
		}

		if *obj.Status.Versions == "synced" {
			return true
		}

		return false
	}
}

func GetChartDependencyStatusFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Chart) func() bool {
	return func() bool {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return false
		}

		if obj.Status.Dependencies == nil {
			return false
		}

		if *obj.Status.Dependencies == "synced" {
			return true
		}

		return false
	}
}

/*
func getRepoGroupFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.RepoGroup) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}
*/

func GetReleaseFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release) func() error {
	return func() error {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return err
		}

		return nil
	}
}

func GetReleaseStatusFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release, status string) func() bool {
	return func() bool {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return false
		}

		if obj.Status.Status == nil {
			return false
		}

		if *obj.Status.Status == status {
			return true
		}

		return false
	}
}

func GetReleaseRevisionFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release, revision int) func() bool {
	return func() bool {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return false
		}

		if obj.Status.Revision == nil {
			return false
		}

		if *obj.Status.Revision == revision {
			return true
		}

		return false
	}
}

func GetReleaseSyncedFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release) func() bool {
	return func() bool {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return false
		}

		if obj.Status.Synced == nil {
			return false
		}

		return *obj.Status.Synced
	}
}

func GetConfigMapFunc(ctx context.Context, key client.ObjectKey, obj *v1.ConfigMap) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}

func (r *RepositoryAssert) Do(namespace string) {

	configmap := &v1.ConfigMap{}
	chartReturnError := BeNil()

	if !r.IsPresent {
		chartReturnError = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "repositories", Group: "yaho.soer3n.dev"}, r.Name))
	}

	Eventually(
		GetRepositoryFunc(context.Background(), client.ObjectKey{Name: r.Name}, r.Obj),
		time.Second*20, time.Millisecond*1500).Should(chartReturnError)

	Eventually(
		GetRepositoryStatusFunc(context.Background(), client.ObjectKey{Name: r.Name}, r.Obj),
		time.Second*20, time.Millisecond*1500).Should(r.Status)

	Eventually(
		GetRepositoryCountFunc(context.Background(), client.ObjectKey{Name: r.Name}, r.Obj),
		time.Second*20, time.Millisecond*1500).Should(BeEquivalentTo(&r.InstalledCharts))

	for _, chartAssert := range r.ManagedCharts {

		Eventually(
			GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-" + r.Name + "-" + chartAssert.Name + "-index", Namespace: namespace}, configmap),
			time.Second*20, time.Millisecond*1500).Should(chartAssert.IndicesInstalled)

		chartAssert.Do(namespace, r.Name)
	}
}

func (r *RepositoryAssert) setDefault() {

	r.IsPresent = false
	r.Synced = BeFalse()
	r.Status = BeFalse()
	r.InstalledCharts = int64(0)

	if r.Obj == nil {
		r.Obj = &helmv1alpha1.Repository{ObjectMeta: metav1.ObjectMeta{Name: r.Name}}
	}
}

func (r *RepositoryAssert) setEverythingInstalled() {

	r.IsPresent = true
	r.Synced = BeTrue()
	r.Status = BeTrue()
	r.InstalledCharts = int64(len(r.ManagedCharts))

	if r.Obj == nil {
		r.Obj = &helmv1alpha1.Repository{ObjectMeta: metav1.ObjectMeta{Name: r.Name}}
	}

}

func (c *ChartAssert) Do(namespace, repo string) {

	configmap := &v1.ConfigMap{}

	Eventually(
		GetChartFunc(context.Background(), client.ObjectKey{Name: c.Name + "-" + repo}, c.Obj),
		time.Second*20, time.Millisecond*1500).Should(c.IsPresent)

	Eventually(
		GetChartSyncedStatusFunc(context.Background(), client.ObjectKey{Name: c.Name + "-" + repo}, c.Obj),
		time.Second*20, time.Millisecond*1500).Should(c.Synced)

	Eventually(
		GetChartDependencyStatusFunc(context.Background(), client.ObjectKey{Name: c.Name + "-" + repo}, c.Obj),
		time.Second*20, time.Millisecond*1500).Should(c.Deps)

	tmplMatcher := BeNil()
	crdMatcher := BeNil()
	defaultValueMatcher := BeNil()

	if !reflect.DeepEqual(c.ResourcesInstalled, BeNil()) {
		tmplMatcher = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-tmpl-"+repo+"-"+c.Name+"-"+c.Version))
		crdMatcher = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-crds-"+repo+"-"+c.Name+"-"+c.Version))
		defaultValueMatcher = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-default-"+repo+"-"+c.Name+"-"+c.Version))
	}

	Eventually(
		GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + repo + "-" + c.Name + "-" + c.Version, Namespace: namespace}, configmap),
		time.Second*20, time.Millisecond*1500).Should(tmplMatcher)

	Eventually(
		GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + repo + "-" + c.Name + "-" + c.Version, Namespace: namespace}, configmap),
		time.Second*20, time.Millisecond*1500).Should(crdMatcher)

	Eventually(
		GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + repo + "-" + c.Name + "-" + c.Version, Namespace: namespace}, configmap),
		time.Second*20, time.Millisecond*1500).Should(defaultValueMatcher)

}

func (c *ChartAssert) setDefault(repo string) {

	c.IsPresent = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, c.Name+"-"+repo))
	c.IndicesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-"+repo+"-"+c.Name+"-index"))
	c.ResourcesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present"))
	c.Synced = BeFalse()
	c.Deps = BeFalse()

	if c.Obj == nil {
		c.Obj = &helmv1alpha1.Chart{ObjectMeta: metav1.ObjectMeta{Name: c.Name}}
	}

}

func (c *ChartAssert) setEverythingInstalled() {

	c.IsPresent = BeNil()
	c.IndicesInstalled = BeNil()
	c.ResourcesInstalled = BeNil()
	c.Synced = BeTrue()
	c.Deps = BeTrue()

	if c.Obj == nil {
		c.Obj = &helmv1alpha1.Chart{ObjectMeta: metav1.ObjectMeta{Name: c.Name}}
	}
}

func (r ReleaseAssert) Do(namespace string) {

	Eventually(
		GetReleaseFunc(context.Background(), client.ObjectKey{Name: r.Name, Namespace: namespace}, r.Obj),
		time.Second*20, time.Millisecond*1500).Should(BeNil())

	if r.IsPresent {
		Eventually(
			GetReleaseStatusFunc(context.Background(), client.ObjectKey{Name: r.Name, Namespace: namespace}, r.Obj, r.Status),
			time.Second*20, time.Millisecond*1500).Should(BeTrue())

		Eventually(
			GetReleaseRevisionFunc(context.Background(), client.ObjectKey{Name: r.Name, Namespace: namespace}, r.Obj, r.Revision),
			time.Second*20, time.Millisecond*1500).Should(BeTrue())

		Eventually(
			GetReleaseSyncedFunc(context.Background(), client.ObjectKey{Name: r.Name, Namespace: namespace}, r.Obj),
			time.Second*20, time.Millisecond*1500).Should(r.Synced)
	}

}

func SetupRBAC(namespace string) {

	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "account",
			Namespace: namespace,
		},
	}

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "account-role",
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts", "pods", "services", "secrets"},
				Verbs:     []string{"get", "list", "create", "delete", "update"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments"},
				Verbs:     []string{"get", "list", "create", "delete", "update"},
			},
		},
	}

	roleBindung := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "account-rolebinding",
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "account-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      "account",
			},
		},
	}

	err = testClient.Create(context.Background(), serviceAccount)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Create(context.Background(), role)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Create(context.Background(), roleBindung)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

}

func RemoveRBAC(namespace string) {

	serviceAccount := &v1.ServiceAccount{}
	role := &rbacv1.Role{}
	roleBinding := &rbacv1.RoleBinding{}

	err = testClient.Get(context.Background(), apitypes.NamespacedName{Name: "account", Namespace: namespace}, serviceAccount)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Get(context.Background(), apitypes.NamespacedName{Name: "account-role", Namespace: namespace}, role)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Get(context.Background(), apitypes.NamespacedName{Name: "account-rolebinding", Namespace: namespace}, roleBinding)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Delete(context.Background(), serviceAccount)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Delete(context.Background(), role)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Delete(context.Background(), roleBinding)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")
}

func SetupConfig(namespace string) {

	config := &helmv1alpha1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config",
			Namespace: namespace,
		},
		Spec: helmv1alpha1.ConfigSpec{
			ServiceAccountName: "account",
			Namespace: helmv1alpha1.Namespace{
				Allowed: []string{namespace},
			},
		},
	}

	err = testClient.Create(context.Background(), config)
	Expect(err).NotTo(HaveOccurred(), "failed to create helm config")
}

func RemoveConfig(namespace string) {

	config := &helmv1alpha1.Config{}

	err = testClient.Get(context.Background(), apitypes.NamespacedName{Name: "config", Namespace: namespace}, config)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

	err = testClient.Delete(context.Background(), config)
	Expect(err).NotTo(HaveOccurred(), "failed to create test resource")
}

func CompareValues(name, namespace string, expected map[string]interface{}) error {

	config := &helmv1alpha1.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config",
			Namespace: namespace,
		},
		Spec: helmv1alpha1.ConfigSpec{
			ServiceAccountName: "account",
		},
	}

	getter, _ := utils.NewRESTClientGetter(config, namespace, namespace, true, testClient, logf.Log)
	ac, err := utils.InitActionConfig(getter, []byte{}, logf.Log)

	if err != nil {
		return err
	}

	client := action.NewGetValues(ac)
	v, err := client.Run(name)

	if err != nil {
		return err
	}

	if diff := cmp.Diff(expected, v); diff != "" {
		fmt.Printf("CompareValues() mismatch (-want +got):\n%s", diff)
		return errors.New("values are not equal")
	}

	return nil
}
