package helm

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	valuesReleaseKind                          *helmv1alpha1.Release
	valuesRelease                              *helmv1alpha1.Release
	valuesReleaseChart                         *helmv1alpha1.Chart
	valuesReleaseRepo, valuesReleaseRepoSecond *helmv1alpha1.Repository
	values                                     *helmv1alpha1.Values
)

var _ = Context("Install a release with values", func() {
	Describe("when no existing resources exist", func() {

		obj := setupNamespace()
		namespace := obj.ObjectMeta.Name

		It("should create a new Repository resource with the specified name and specified url", func() {
			ctx := context.Background()
			// namespace = "test-" + randStringRunes(7)

			By("should create a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Repository resources with the specified name and specified url")
			valuesReleaseRepo = &helmv1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: testRepoName,
				},
				Spec: helmv1alpha1.RepositorySpec{
					Name: testRepoName,
					URL:  testRepoURL,
					Charts: []helmv1alpha1.Entry{
						{
							Name:     "testing",
							Versions: []string{"0.1.0"},
						},
						{
							Name:     "testing-nested",
							Versions: []string{"0.1.0"},
						},
					},
				},
			}

			err = testClient.Create(context.Background(), valuesReleaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			valuesReleaseRepoSecond = &helmv1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: testRepoNameSecond,
				},
				Spec: helmv1alpha1.RepositorySpec{
					Name: testRepoNameSecond,
					URL:  testRepoURLSecond,
					Charts: []helmv1alpha1.Entry{
						{
							Name:     "testing-dep",
							Versions: []string{"0.1.0"},
						},
					},
				},
			}

			err = testClient.Create(context.Background(), valuesReleaseRepoSecond)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new values resource with specified")

			nestedMap := map[string]string{
				"baz": "faz",
			}

			valuesSpec := map[string]interface{}{
				"foo": "bar",
				"boo": nestedMap,
			}

			valuesSpecRaw, err := json.Marshal(valuesSpec)
			Expect(err).NotTo(HaveOccurred(), "failed to convert values")

			values = &helmv1alpha1.Values{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ValuesSpec{
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

			nestedMap = map[string]string{
				"baz": "faz",
			}
			valuesSpec = map[string]interface{}{
				"foo": "bar",
				"boo": nestedMap,
			}

			valuesSpecRaw, err = json.Marshal(valuesSpec)
			Expect(err).NotTo(HaveOccurred(), "failed to convert values")

			values = &helmv1alpha1.Values{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-nested",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ValuesSpec{
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

			valuesSpecRaw, err = json.Marshal(valuesSpec)
			Expect(err).NotTo(HaveOccurred(), "failed to convert values")

			values = &helmv1alpha1.Values{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-embedded",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ValuesSpec{
					ValuesMap: &runtime.RawExtension{
						Raw: []byte(valuesSpecRaw),
					},
				},
			}

			err = testClient.Create(context.Background(), values)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Release resource with specified")

			deployment := &helmv1alpha1.Repository{}
			valuesReleaseChart = &helmv1alpha1.Chart{}
			valuesRelease = &helmv1alpha1.Release{}
			configmap := &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName}, valuesReleaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			valuesReleaseKind = &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testReleaseName,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    testReleaseName,
					Chart:   testReleaseChartName,
					Repo:    testRepoName,
					Version: testReleaseChartVersion,
					Values: []string{
						"testresource",
						"notpresent",
					},
				},
			}

			err = testClient.Create(context.Background(), valuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: testReleaseName, Namespace: valuesReleaseKind.Namespace}, valuesRelease),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(valuesRelease.ObjectMeta.Name).To(Equal(testReleaseName))

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName}, valuesReleaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion))

			By("should update release after changing value resource")

			nestedMap = map[string]string{
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

			secondValuesReleaseKind := &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testReleaseNameSecond,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    testReleaseNameSecond,
					Chart:   testReleaseChartNameSecond,
					Repo:    testRepoNameSecond,
					Version: testReleaseChartVersionSecond,
					Values: []string{
						"testresource",
					},
				},
			}

			err = testClient.Create(context.Background(), secondValuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), valuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			err = testClient.Delete(context.Background(), secondValuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: testReleaseName, Namespace: valuesReleaseKind.Namespace}, valuesRelease),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("should remove this Repository resources with the specified name and specified url")

			err = testClient.Delete(context.Background(), valuesReleaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			err = testClient.Delete(context.Background(), valuesReleaseRepoSecond)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName}, valuesReleaseChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

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
