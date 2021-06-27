package helm

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

var valuesReleaseKind *helmv1alpha1.Release
var valuesRelease *helmv1alpha1.Release
var valuesReleaseChart *helmv1alpha1.Chart
var valuesReleaseRepo *helmv1alpha1.Repo
var values *helmv1alpha1.Values

var _ = Context("Install a release with values", func() {

	Describe("when no existing resources exist", func() {

		It("should create a new Repository resource with the specified name and specified url", func() {
			ctx := context.Background()
			namespace := "test-" + randStringRunes(7)

			By("should create a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = k8sClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Repository resource with the specified name and specified url")
			valuesReleaseRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-123",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					Url:  "https://submariner-io.github.io/submariner-charts/charts",
				},
			}

			err = k8sClient.Create(ctx, valuesReleaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(3 * time.Second)

			deployment := &helmv1alpha1.Repo{}
			valuesReleaseChart = &helmv1alpha1.Chart{}

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

			err = k8sClient.Create(ctx, values)
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

			err = k8sClient.Create(ctx, values)
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

			err = k8sClient.Create(ctx, values)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetResourceFunc(ctx, client.ObjectKey{Name: "testresource-123", Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&deployment.ObjectMeta.Name).To(Equal("testresource-123"))

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner-operator", Namespace: namespace}, valuesReleaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&valuesReleaseChart.ObjectMeta.Name).To(Equal("submariner-operator"))

			By("should create a new Release resource with specified")

			valuesReleaseKind = &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "deployment-values",
					Chart:   "submariner-operator",
					Repo:    "testresource-123",
					Version: "0.7.0",
					ValuesTemplate: &helmv1alpha1.ValueTemplate{
						ValueRefs: []string{
							"testresource",
							"notpresent",
						},
					},
				},
			}

			err = k8sClient.Create(ctx, valuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(5 * time.Second)

			valuesRelease = &helmv1alpha1.Release{}
			valuesReleaseChart = &helmv1alpha1.Chart{}
			configmap := &v1.ConfigMap{}

			Eventually(
				GetReleaseFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: valuesReleaseKind.Namespace}, valuesRelease),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&valuesRelease.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner-operator", Namespace: valuesReleaseKind.Namespace}, valuesReleaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&valuesReleaseChart.ObjectMeta.Name).To(Equal("submariner-operator"))

			Eventually(
				GetConfigMapFunc(ctx, client.ObjectKey{Name: "helm-tmpl-submariner-operator-0.7.0", Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&configmap.ObjectMeta.Name).To(Equal("helm-tmpl-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(ctx, client.ObjectKey{Name: "helm-crds-submariner-operator-0.7.0", Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&configmap.ObjectMeta.Name).To(Equal("helm-crds-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(ctx, client.ObjectKey{Name: "helm-default-submariner-operator-0.7.0", Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&configmap.ObjectMeta.Name).To(Equal("helm-default-submariner-operator-0.7.0"))

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

			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "testresource-nested",
				Namespace: namespace,
			}, values)
			Expect(err).NotTo(HaveOccurred(), "failed to get values resource")

			values.Spec.ValuesMap.Raw = []byte(valuesSpecRaw)

			err = k8sClient.Update(ctx, values)
			Expect(err).NotTo(HaveOccurred(), "failed to update values resource")

			time.Sleep(5 * time.Second)

			secondValuesReleaseKind := &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-2",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "deployment-values-2",
					Chart:   "submariner-operator",
					Repo:    "testresource-123",
					Version: "0.7.0",
					ValuesTemplate: &helmv1alpha1.ValueTemplate{
						ValueRefs: []string{
							"testresource",
						},
					},
				},
			}

			err = k8sClient.Create(ctx, secondValuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(5 * time.Second)

			By("should remove this Release resource with the specified configmaps after deletion")

			err = k8sClient.Delete(ctx, valuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			err = k8sClient.Delete(ctx, secondValuesReleaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(5 * time.Second)

			Eventually(
				GetReleaseFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: valuesReleaseKind.Namespace}, valuesRelease),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("should remove this Repository resource with the specified name and specified url")

			err = k8sClient.Delete(ctx, valuesReleaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				GetResourceFunc(ctx, client.ObjectKey{Name: "testresource-123", Namespace: valuesReleaseRepo.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner-operator", Namespace: valuesReleaseRepo.Namespace}, valuesReleaseChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(ctx, client.ObjectKey{Name: "helm-tmpl-submariner-operator-0.7.0", Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(ctx, client.ObjectKey{Name: "helm-crds-submariner-operator-0.7.0", Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(ctx, client.ObjectKey{Name: "helm-default-submariner-operator-0.7.0", Namespace: valuesReleaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("by deletion of namespace")
			releaseNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = k8sClient.Delete(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

		})
	})
})
