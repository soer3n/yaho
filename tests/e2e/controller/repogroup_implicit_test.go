package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	repoGroupImplicitKind *helmv1alpha1.RepoGroup
)

var _ = Context("Install and configure a repository group", func() {

	obj := setupNamespace()
	namespace := obj.ObjectMeta.Name

	Describe("when no existing resource exist", func() {

		FIt("should start with creating dependencies", func() {
			ctx := context.Background()

			// wait on readiness of controllers
			time.Sleep(2 * time.Second)

			By("install a new namespace")
			repoGroupNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			chartOneAssert := &ChartAssert{
				Name:               testRepoChartNameAssert,
				Version:            testRepoChartNameAssertqVersion,
				IsPresent:          BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-testresource")),
				IndicesInstalled:   BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-testing-index")),
				ResourcesInstalled: BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present")),
				Synced:             BeFalse(),
				Deps:               BeFalse(),
			}

			chartTwoAssert := &ChartAssert{
				Name:               testRepoChartSecondNameAssert,
				Version:            testRepoChartSecondNameAssertVersion,
				IsPresent:          BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-dep-testresource-2")),
				IndicesInstalled:   BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-2-testing-dep-index")),
				ResourcesInstalled: BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present")),
				Synced:             BeFalse(),
				Deps:               BeFalse(),
			}

			chartThreeAssert := &ChartAssert{
				Name:               testRepoChartThirdNameAssert,
				Version:            testRepoChartThirdNameAssertVersion,
				IsPresent:          BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-nested-testresource")),
				IndicesInstalled:   BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-testing-nested-index")),
				ResourcesInstalled: BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present")),
				Synced:             BeFalse(),
				Deps:               BeFalse(),
			}

			repoOneAssert := RepositoryAssert{
				Name:            testRepoName,
				IsPresent:       false,
				Synced:          BeFalse(),
				Status:          BeFalse(),
				InstalledCharts: int64(0),
				ManagedCharts:   []*ChartAssert{chartOneAssert, chartThreeAssert},
			}

			repoTwoAssert := RepositoryAssert{
				Name:            testRepoNameSecond,
				IsPresent:       false,
				Synced:          BeFalse(),
				Status:          BeFalse(),
				InstalledCharts: int64(0),
				ManagedCharts:   []*ChartAssert{chartTwoAssert},
			}

			repoOneAssert.Obj = &helmv1alpha1.Repository{ObjectMeta: metav1.ObjectMeta{Name: testRepoName}}
			repoTwoAssert.Obj = &helmv1alpha1.Repository{ObjectMeta: metav1.ObjectMeta{Name: testRepoNameSecond}}

			chartOneAssert.Obj = &helmv1alpha1.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartOneAssert.Name}}
			chartTwoAssert.Obj = &helmv1alpha1.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartTwoAssert.Name}}
			chartThreeAssert.Obj = &helmv1alpha1.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartThreeAssert.Name}}

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating a new empty repository group resource")
			repoGroupImplicitKind = &helmv1alpha1.RepoGroup{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoName},
				Spec: helmv1alpha1.RepoGroupSpec{
					LabelSelector: "foo",
					Repos:         []helmv1alpha1.RepositorySpec{},
				},
			}

			err = testClient.Create(context.Background(), repoGroupImplicitKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("adding first repository without charts")

			repoGroupImplicitKind.Spec.Repos = []helmv1alpha1.RepositorySpec{
				{
					Name:   testRepoNameSecond,
					URL:    testRepoURLSecond,
					Charts: []helmv1alpha1.Entry{},
				},
			}

			err = testClient.Update(context.Background(), repoGroupImplicitKind)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoTwoAssert.IsPresent = true
			repoTwoAssert.Status = BeTrue()
			repoTwoAssert.Synced = BeTrue()

			chartTwoAssert.IndicesInstalled = BeNil()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("...")

			repoGroupImplicitKind.Spec.Repos = []helmv1alpha1.RepositorySpec{
				{
					Name: testRepoNameSecond,
					URL:  testRepoURLSecond,
					Charts: []helmv1alpha1.Entry{
						{
							Name:     "testing-dep",
							Versions: []string{"0.1.1"},
						},
					},
				},
			}

			err = testClient.Update(context.Background(), repoGroupImplicitKind)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			// time.Sleep(300 * time.Second)

			chartTwoAssert.IsPresent = BeNil()
			chartTwoAssert.ResourcesInstalled = BeNil()
			chartTwoAssert.Synced = BeFalse()
			chartTwoAssert.Deps = BeFalse()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("...")

			repoGroupImplicitKind.Spec.Repos = []helmv1alpha1.RepositorySpec{
				{
					Name: testRepoNameSecond,
					URL:  testRepoURLSecond,
					Charts: []helmv1alpha1.Entry{
						{
							Name:     "testing-dep",
							Versions: []string{"0.1.1"},
						},
					},
				},
				{
					Name: testRepoName,
					URL:  testRepoURL,
				},
			}

			err = testClient.Update(context.Background(), repoGroupImplicitKind)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			// time.Sleep(500 * time.Second)

			chartOneAssert.IndicesInstalled = BeNil()
			chartOneAssert.IsPresent = BeNil()
			chartOneAssert.Deps = BeTrue()
			chartOneAssert.Synced = BeTrue()

			chartTwoAssert.IsPresent = BeNil()
			chartTwoAssert.ResourcesInstalled = BeNil()
			chartTwoAssert.Synced = BeTrue()
			chartTwoAssert.Deps = BeTrue()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("remove every repository left when group is deleted")

			err = testClient.Delete(context.Background(), repoGroupImplicitKind)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")

			chartTwoAssert.IsPresent = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-dep-testresource-2"))
			chartTwoAssert.IndicesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-2-testing-dep-index"))
			chartTwoAssert.ResourcesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present"))
			chartTwoAssert.Synced = BeFalse()
			chartTwoAssert.Deps = BeFalse()

			repoTwoAssert.IsPresent = false
			repoTwoAssert.Synced = BeFalse()
			repoTwoAssert.Status = BeFalse()
			repoTwoAssert.InstalledCharts = int64(1)

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("by deletion of namespace test should finish successfully")
			repoGroupNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Delete(context.Background(), repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")
		})
	})
})
