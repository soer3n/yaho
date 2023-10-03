package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Context("Install and configure a repository", func() {

	obj := setupNamespace()
	namespace := obj.ObjectMeta.Name

	Describe("when no existing resource exist", func() {

		It("should start with creating dependencies", func() {
			ctx := context.Background()

			// wait on readiness of controllers
			time.Sleep(2 * time.Second)

			By("install a new namespace")
			repoNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, repoNamespace)
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

			repoOneAssert.Obj = &yahov1alpha2.Repository{ObjectMeta: metav1.ObjectMeta{Name: testRepoName}}
			repoTwoAssert.Obj = &yahov1alpha2.Repository{ObjectMeta: metav1.ObjectMeta{Name: testRepoNameSecond}}

			chartOneAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartOneAssert.Name}}
			chartTwoAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartTwoAssert.Name}}
			chartThreeAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartThreeAssert.Name}}

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating a new repository resource with wrong url")

			repoOneAssert.Obj.Spec = yahov1alpha2.RepositorySpec{
				Source: yahov1alpha2.RepositorySource{
					URL:  testRepoURL + "foo",
					Type: "helm",
				},
				Charts: yahov1alpha2.RepositoryCharts{
					Sync: yahov1alpha2.Sync{
						Enabled:  true,
						Interval: "10s",
					},
					Items: []yahov1alpha2.Entry{},
				},
			}

			err = testClient.Create(context.Background(), repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoOneAssert.IsPresent = true

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("updating the repository resource with correct url")

			repoOneAssert.Obj.Spec.Source.URL = testRepoURL

			err = testClient.Update(context.Background(), repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoOneAssert.Status = BeTrue()
			repoOneAssert.Synced = BeNil()
			chartOneAssert.IndicesInstalled = BeNil()

			chartThreeAssert.IndicesInstalled = BeNil()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("updating the repository resource with charts should create index configmaps")

			err = testClient.Get(context.Background(), types.NamespacedName{Name: repoOneAssert.Obj.Name}, repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

			repoOneAssert.Obj.Spec.Charts = yahov1alpha2.RepositoryCharts{
				Sync: yahov1alpha2.Sync{
					Enabled:  true,
					Interval: "10s",
				},
				Items: []yahov1alpha2.Entry{
					{
						Name:     testRepoChartNameAssert,
						Versions: []string{},
					},
					{
						Name:     testRepoChartThirdNameAssert,
						Versions: []string{},
					},
				},
			}

			err = testClient.Update(context.Background(), repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoOneAssert.InstalledCharts = int64(2)

			chartOneAssert.IsPresent = BeNil()
			chartOneAssert.Synced = BeTrue()
			chartOneAssert.Deps = BeTrue()

			chartThreeAssert.IsPresent = BeNil()
			chartThreeAssert.Synced = BeTrue()
			chartThreeAssert.Deps = BeTrue()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("updating the repository resource with chart versions belonging to it should create configmaps for chart version")

			err = testClient.Get(context.Background(), types.NamespacedName{Name: repoOneAssert.Obj.Name}, repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

			repoOneAssert.Obj.Spec.Charts = yahov1alpha2.RepositoryCharts{
				Sync: yahov1alpha2.Sync{
					Enabled:  true,
					Interval: "10s",
				},
				Items: []yahov1alpha2.Entry{
					{
						Name:     testRepoChartNameAssert,
						Versions: []string{testRepoChartNameAssertqVersion},
					},
					{
						Name:     testRepoChartThirdNameAssert,
						Versions: []string{testRepoChartThirdNameAssertVersion},
					},
				},
			}

			err = testClient.Update(context.Background(), repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert.ResourcesInstalled = BeNil()
			chartOneAssert.Deps = BeTrue()

			chartThreeAssert.ResourcesInstalled = BeNil()
			chartThreeAssert.Deps = BeTrue()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("deleting the first chart of the repository resource")

			err = testClient.Get(context.Background(), types.NamespacedName{Name: repoOneAssert.Obj.Name}, repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

			repoOneAssert.Obj.Spec.Charts = yahov1alpha2.RepositoryCharts{
				Sync: yahov1alpha2.Sync{
					Enabled:  true,
					Interval: "10s",
				},
				Items: []yahov1alpha2.Entry{
					{
						Name:     testRepoChartThirdNameAssert,
						Versions: []string{testRepoChartThirdNameAssertVersion},
					},
				},
			}

			err = testClient.Update(context.Background(), repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert.IsPresent = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, testRepoChartNameAssert+"-"+testRepoName))
			chartOneAssert.ResourcesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present"))
			chartOneAssert.Synced = BeFalse()
			chartOneAssert.Deps = BeFalse()

			repoOneAssert.InstalledCharts = int64(2)

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("deleting the repository resource should also cleanup related resources")

			err = testClient.Get(context.Background(), types.NamespacedName{Name: repoOneAssert.Obj.Name}, repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

			err = testClient.Delete(context.Background(), repoOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")

			chartOneAssert.IndicesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-testing-index"))

			chartThreeAssert.IsPresent = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, testRepoChartThirdNameAssert+"-"+testRepoName))
			chartThreeAssert.IndicesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-testing-nested-index"))
			chartThreeAssert.Deps = BeFalse()
			chartThreeAssert.Synced = BeFalse()
			chartThreeAssert.ResourcesInstalled = BeFalse()

			repoOneAssert.IsPresent = false
			repoOneAssert.InstalledCharts = int64(0)
			repoOneAssert.Status = BeFalse()
			repoOneAssert.Synced = BeFalse()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("by deletion of namespace test should finish successfully")

			repoNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Delete(context.Background(), repoNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to delete testresource")
		})
	})
})
