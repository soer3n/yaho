package helm

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

var (
	values *yahov1alpha2.Values
)

var _ = Context("Install a release with values", func() {
	Describe("when no existing resources exist", func() {

		obj := setupNamespace()
		namespace := obj.ObjectMeta.Name

		It("should create a new Release resource with the specified name and values", func() {
			ctx := context.Background()

			// wait on readiness of controllers
			time.Sleep(2 * time.Second)

			By("install a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert := &ChartAssert{
				Name:    testRepoChartNameAssert,
				Version: testRepoChartNameAssertqVersion,
			}

			chartOneAssert.setDefault(testRepoName)

			chartTwoAssert := &ChartAssert{
				Name:    testRepoChartSecondNameAssert,
				Version: testRepoChartSecondNameAssertVersion,
			}

			chartTwoAssert.setDefault(testRepoNameSecond)

			chartThreeAssert := &ChartAssert{
				Name:    testRepoChartThirdNameAssert,
				Version: testRepoChartThirdNameAssertVersion,
			}

			chartThreeAssert.setDefault(testRepoName)

			repoOneAssert := RepositoryAssert{
				Name:          testRepoName,
				ManagedCharts: []*ChartAssert{chartOneAssert, chartThreeAssert},
			}

			repoOneAssert.setDefault()

			repoTwoAssert := RepositoryAssert{
				Name:          testRepoNameSecond,
				ManagedCharts: []*ChartAssert{chartTwoAssert},
			}

			repoTwoAssert.setDefault()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			SetupRBAC(namespace)
			SetupConfig(namespace)

			By("creating needed repository group resource")

			releaseRepoOne = &yahov1alpha2.Repository{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoName},
				Spec: yahov1alpha2.RepositorySpec{
					Source: yahov1alpha2.RepositorySource{
						URL:  testRepoURL,
						Type: "helm",
					},
					Charts: yahov1alpha2.RepositoryCharts{
						Sync: yahov1alpha2.Sync{
							Enabled:  true,
							Interval: "10s",
						},
						Items: []yahov1alpha2.Entry{
							{
								Name:     testReleaseChartName,
								Versions: []string{testReleaseChartVersion},
							},
							{
								Name:     testReleaseChartThirdNameAssert,
								Versions: []string{testReleaseChartThirdNameAssertVersion},
							},
						},
					},
				},
			}

			err = testClient.Create(ctx, releaseRepoOne)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseRepoTwo = &yahov1alpha2.Repository{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoNameSecond},
				Spec: yahov1alpha2.RepositorySpec{
					Source: yahov1alpha2.RepositorySource{
						URL:  testRepoURLSecond,
						Type: "helm",
					},
					Charts: yahov1alpha2.RepositoryCharts{
						Sync: yahov1alpha2.Sync{
							Enabled:  true,
							Interval: "10s",
						},
						Items: []yahov1alpha2.Entry{
							{
								Name:     testReleaseChartNameSecond,
								Versions: []string{testReleaseChartVersionSecond},
							},
						},
					},
				},
			}

			err = testClient.Create(ctx, releaseRepoTwo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert.setEverythingInstalled()
			chartTwoAssert.setEverythingInstalled()
			chartThreeAssert.setEverythingInstalled()

			repoOneAssert.setEverythingInstalled()
			repoTwoAssert.setEverythingInstalled()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating a new release resource with a valid repository and version")

			s := "config"

			releaseAssert := &ReleaseAssert{
				Name: testReleaseName,
				Obj: &yahov1alpha2.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testReleaseName,
						Namespace: namespace,
					},
				},
				IsPresent: true,
			}

			releaseAssert.Obj = &yahov1alpha2.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testReleaseName,
					Namespace: namespace,
				},
				Spec: yahov1alpha2.ReleaseSpec{
					Name:      testReleaseName,
					Namespace: &namespace,
					Chart:     testReleaseChartName,
					Repo:      testRepoName,
					Version:   testReleaseChartVersion,
					Config:    &s,
				},
			}

			err = testClient.Create(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseAssert.IsPresent = true
			releaseAssert.Status = "success"
			releaseAssert.Synced = BeTrue()
			releaseAssert.Revision = 1

			releaseAssert.Do(namespace)

			var expectedValues map[string]interface{}

			err = CompareValues(releaseAssert.Obj.ObjectMeta.Name, namespace, expectedValues)
			Expect(err).NotTo(HaveOccurred())

			By("should create a new values resource with specified")

			nestedMap := map[string]interface{}{
				"baz": "faz",
			}

			valuesSpec := map[string]interface{}{
				"foo": "bar",
				"boo": nestedMap,
			}

			valuesSpecRaw, err := json.Marshal(valuesSpec)
			Expect(err).NotTo(HaveOccurred(), "failed to convert values")

			values = &yahov1alpha2.Values{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: namespace,
				},
				Spec: yahov1alpha2.ValuesSpec{
					ValuesMap: &runtime.RawExtension{
						Raw: []byte(valuesSpecRaw),
					},
					Refs: map[string]string{
						"ref": "testresource-nested",
					},
				},
			}

			err = testClient.Create(context.Background(), values)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new values resource with specified")

			valuesSpecRaw, err = json.Marshal(valuesSpec)
			Expect(err).NotTo(HaveOccurred(), "failed to convert values")

			values = &yahov1alpha2.Values{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-nested",
					Namespace: namespace,
				},
				Spec: yahov1alpha2.ValuesSpec{
					ValuesMap: &runtime.RawExtension{
						Raw: []byte(valuesSpecRaw),
					},

					Refs: map[string]string{
						"ref": "testresource-embedded",
					},
				},
			}

			err = testClient.Create(context.Background(), values)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new values resource with specified")

			refMap := map[string]interface{}{
				"baz": "faz",
			}
			refSpec := map[string]interface{}{
				"foo": "bar",
				"boo": refMap,
			}

			refSpecRaw, err := json.Marshal(refSpec)
			Expect(err).NotTo(HaveOccurred(), "failed to convert values")

			values = &yahov1alpha2.Values{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-embedded",
					Namespace: namespace,
				},
				Spec: yahov1alpha2.ValuesSpec{
					ValuesMap: &runtime.RawExtension{
						Raw: []byte(refSpecRaw),
					},
				},
			}

			err = testClient.Create(context.Background(), values)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(time.Second)

			By("should create a new Release resource with specified")

			releaseAssert.Obj.Spec = yahov1alpha2.ReleaseSpec{
				Name:    testReleaseName,
				Chart:   testReleaseChartName,
				Repo:    testRepoName,
				Version: testReleaseChartVersion,
				Config:  &s,
				Values: []string{
					"testresource",
				},
			}

			err = testClient.Update(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseAssert.Revision = 2

			releaseAssert.Do(namespace)

			embeddedValues := map[string]interface{}{
				"ref": refSpec,
				"foo": "bar",
				"boo": nestedMap,
			}

			expectedValues = map[string]interface{}{
				"foo": "bar",
				"boo": nestedMap,
				"ref": embeddedValues,
			}

			err = CompareValues(releaseAssert.Obj.ObjectMeta.Name, namespace, expectedValues)
			Expect(err).NotTo(HaveOccurred())

			By("should update release after changing value resource")

			nestedMap = map[string]interface{}{
				"baz": "foo",
			}
			valuesSpec = map[string]interface{}{
				"foo": "bar",
				"boo": nestedMap,
			}

			valuesSpecRaw, err = json.Marshal(valuesSpec)
			Expect(err).NotTo(HaveOccurred(), "failed to convert values")

			err = testClient.Get(context.Background(), types.NamespacedName{
				Name:      "testresource-nested",
				Namespace: namespace,
			}, values)
			Expect(err).NotTo(HaveOccurred(), "failed to get values resource")

			values.Spec.ValuesMap.Raw = []byte(valuesSpecRaw)

			err = testClient.Update(context.Background(), values)
			Expect(err).NotTo(HaveOccurred(), "failed to update values resource")

			releaseAssert.Revision = 3

			releaseAssert.Do(namespace)

			helperMap := expectedValues["ref"].(map[string]interface{})
			helperMap["boo"] = nestedMap
			expectedValues["ref"] = helperMap

			err = CompareValues(releaseAssert.Obj.ObjectMeta.Name, namespace, expectedValues)
			Expect(err).NotTo(HaveOccurred())

			releaseAssert.Obj.Spec.Values = []string{
				"testresource",
				"notpresent",
			}

			err = testClient.Update(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to udpate test resource")

			releaseAssert.Do(namespace)

			err = CompareValues(releaseAssert.Obj.ObjectMeta.Name, namespace, expectedValues)
			Expect(err).NotTo(HaveOccurred())

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseAssert.IsPresent = false

			releaseAssert.Do(namespace)

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseRepoOne)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			err = testClient.Delete(context.Background(), releaseRepoTwo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert.setDefault(testRepoName)
			chartTwoAssert.setDefault(testRepoNameSecond)
			chartThreeAssert.setDefault(testRepoName)

			repoOneAssert.setDefault()
			repoTwoAssert.setDefault()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			RemoveConfig(namespace)
			RemoveRBAC(namespace)

			By("by deletion of namespace")
			releaseNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Delete(context.Background(), releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")
		})
	})
})
