package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var (
	repoOne, repoTwo *yahov1alpha2.Repository
)

var _ = Context("Install and configure a chart", func() {

	obj := setupNamespace()
	namespace := obj.ObjectMeta.Name

	Describe("when no existing resource exist", func() {

		It("should start with creating dependencies", func() {
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

			repoOneAssert.Obj = &yahov1alpha2.Repository{ObjectMeta: metav1.ObjectMeta{Name: testRepoName}}
			repoTwoAssert.Obj = &yahov1alpha2.Repository{ObjectMeta: metav1.ObjectMeta{Name: testRepoNameSecond}}

			chartOneAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartOneAssert.Name}}
			chartTwoAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartTwoAssert.Name}}
			chartThreeAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartThreeAssert.Name}}

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating needed repository group resource")
			repoOne = &yahov1alpha2.Repository{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoName},
				Spec: yahov1alpha2.RepositorySpec{

					Name:   testRepoName,
					URL:    testRepoURL,
					Charts: []yahov1alpha2.Entry{},
				},
			}

			err = testClient.Create(context.Background(), repoOne)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			By("creating needed repository group resource")
			repoTwo = &yahov1alpha2.Repository{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoNameSecond},
				Spec: yahov1alpha2.RepositorySpec{
					Name:   testRepoNameSecond,
					URL:    testRepoURLSecond,
					Charts: []yahov1alpha2.Entry{},
				},
			}

			err = testClient.Create(context.Background(), repoTwo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoOneAssert.IsPresent = true
			repoOneAssert.Status = BeTrue()
			repoOneAssert.Synced = BeTrue()

			repoTwoAssert.IsPresent = true
			repoTwoAssert.Status = BeTrue()
			repoTwoAssert.Synced = BeTrue()

			chartOneAssert.IndicesInstalled = BeNil()

			chartTwoAssert.IndicesInstalled = BeNil()

			chartThreeAssert.IndicesInstalled = BeNil()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating a chart to first repository without a dependency")

			chartThreeAssert.Obj = &yahov1alpha2.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name: testRepoChartThirdNameAssert + "-" + testRepoName,
				},
				Spec: yahov1alpha2.ChartSpec{
					Name:       testRepoChartThirdNameAssert,
					Repository: testRepoName,
					Versions:   []string{},
					CreateDeps: true,
				},
			}

			err = testClient.Create(context.Background(), chartThreeAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoOneAssert.InstalledCharts = int64(1)

			chartThreeAssert.IsPresent = BeNil()
			chartThreeAssert.Deps = BeTrue()
			chartThreeAssert.Synced = BeTrue()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("adding a not valid version to the first chart")

			chartThreeAssert.Obj.Spec = yahov1alpha2.ChartSpec{
				Name:       testRepoChartThirdNameAssert,
				Repository: testRepoName,
				Versions:   []string{testRepoChartNotValidVersion},
				CreateDeps: true,
			}

			err = testClient.Update(context.Background(), chartThreeAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoOneAssert.Synced = BeFalse()

			chartThreeAssert.Synced = BeFalse()
			chartThreeAssert.Deps = BeFalse()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("adding a valid version to the first chart")

			err = testClient.Get(context.Background(), types.NamespacedName{Name: testRepoChartThirdNameAssert + "-" + testRepoName}, chartThreeAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			chartThreeAssert.Obj.Spec = yahov1alpha2.ChartSpec{
				Name:       testRepoChartThirdNameAssert,
				Repository: testRepoName,
				Versions:   []string{testRepoChartThirdNameAssertVersion},
				CreateDeps: true,
			}

			err = testClient.Update(context.Background(), chartThreeAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoOneAssert.Synced = BeTrue()

			chartThreeAssert.ResourcesInstalled = BeNil()
			chartThreeAssert.Deps = BeTrue()
			chartThreeAssert.Synced = BeTrue()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating a second chart and disabling dependency creation")

			chartTwoAssert.Obj = &yahov1alpha2.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name: testRepoChartSecondNameAssert + "-" + testRepoNameSecond,
				},
				Spec: yahov1alpha2.ChartSpec{
					Name:       testRepoChartSecondNameAssert,
					Repository: testRepoNameSecond,
					Versions:   []string{},
					CreateDeps: false,
				},
			}

			err = testClient.Create(context.Background(), chartTwoAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoTwoAssert.InstalledCharts = int64(1)

			chartTwoAssert.IsPresent = BeNil()
			chartTwoAssert.IndicesInstalled = BeNil()
			chartTwoAssert.Deps = BeFalse()
			chartTwoAssert.Synced = BeTrue()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("updating a second chart setting a valid version")

			chartTwoAssert.Obj.Spec = yahov1alpha2.ChartSpec{
				Name:       testRepoChartSecondNameAssert,
				Repository: testRepoNameSecond,
				Versions:   []string{testRepoChartSecondNameAssertVersion},
				CreateDeps: false,
			}

			err = testClient.Update(context.Background(), chartTwoAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			chartTwoAssert.ResourcesInstalled = BeNil()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("updating a second chart by enabling dependency creation")

			chartTwoAssert.Obj.Spec = yahov1alpha2.ChartSpec{
				Name:       testRepoChartSecondNameAssert,
				Repository: testRepoNameSecond,
				Versions:   []string{testRepoChartSecondNameAssertVersion},
				CreateDeps: true,
			}

			err = testClient.Update(context.Background(), chartTwoAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoOneAssert.InstalledCharts = int64(2)

			chartOneAssert.IsPresent = BeNil()
			chartOneAssert.ResourcesInstalled = BeNil()
			chartOneAssert.Deps = BeTrue()
			chartOneAssert.Synced = BeTrue()

			chartTwoAssert.Deps = BeTrue()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("deleting the second chart")

			err = testClient.Delete(context.Background(), chartTwoAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoTwoAssert.InstalledCharts = int64(0)

			chartTwoAssert.ResourcesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present"))
			chartTwoAssert.IsPresent = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-dep-testresource-2"))
			chartTwoAssert.Deps = BeFalse()
			chartTwoAssert.Synced = BeFalse()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("deleting the first chart")

			err = testClient.Delete(context.Background(), chartThreeAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoOneAssert.InstalledCharts = int64(1)

			chartThreeAssert.ResourcesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present"))
			chartThreeAssert.IsPresent = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-nested-testresource"))
			chartThreeAssert.Deps = BeFalse()
			chartThreeAssert.Synced = BeFalse()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("deleting dependency chart related to second chart resource")

			err = testClient.Delete(context.Background(), chartOneAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			repoOneAssert.InstalledCharts = int64(0)

			chartOneAssert.ResourcesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present"))
			chartOneAssert.IsPresent = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-testresource"))
			chartOneAssert.Deps = BeFalse()
			chartOneAssert.Synced = BeFalse()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("deleting repogroup resource")

			err = testClient.Delete(context.Background(), repoOne)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			err = testClient.Delete(context.Background(), repoTwo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoOneAssert.IsPresent = false
			repoOneAssert.Status = BeFalse()
			repoOneAssert.Synced = BeFalse()

			repoTwoAssert.IsPresent = false
			repoTwoAssert.Status = BeFalse()
			repoTwoAssert.Synced = BeFalse()

			chartOneAssert.IndicesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-testing-index"))

			chartTwoAssert.IndicesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-2-testing-dep-index"))

			chartThreeAssert.IndicesInstalled = BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "helm-testresource-testing-nested-index"))

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("deleting test namespace")

			repoGroupNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Delete(context.Background(), repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")
		})
	})
})
