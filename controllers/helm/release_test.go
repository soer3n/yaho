package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

var releaseKind *helmv1alpha1.Release
var release *helmv1alpha1.Release
var releaseChart *helmv1alpha1.Chart
var releaseRepo *helmv1alpha1.Repo

var _ = Context("Install a release", func() {

	Describe("when no existing resources exist", func() {

		FIt("should start with creating dependencies", func() {
			ctx := context.Background()
			namespace := "test-" + randStringRunes(7)

			By("install a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			By("creating a new repository resource with the specified name and specified url")
			releaseRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-123",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					URL:  "https://submariner-io.github.io/submariner-charts/charts",
				},
			}

			err = testClient.Create(context.Background(), releaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			time.Sleep(3 * time.Second)

			deployment = &helmv1alpha1.Repo{}
			repoChart = &helmv1alpha1.Chart{}

			By("creating a new release resource with specified data")

			releaseKind = &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name: "deployment-name",
					Namespace: helmv1alpha1.Namespace{
						Name:    namespace,
						Install: false,
					},
					Chart:   "submariner-operator",
					Repo:    "testresource-123",
					Version: "0.7.x",
				},
			}

			err = testClient.Create(context.Background(), releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			time.Sleep(5 * time.Second)

			release = &helmv1alpha1.Release{}
			releaseChart = &helmv1alpha1.Chart{}
			configmap := &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: "testresource-123", Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "submariner-operator", Namespace: namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&release.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "submariner-operator", Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-default-submariner-operator-0.7.0"))

			time.Sleep(3 * time.Second)

			By("should update this Release resource with values reference")

			existigRelease := &helmv1alpha1.Release{}
			err = testClient.Get(context.Background(), types.NamespacedName{
				Name:      "testresource",
				Namespace: namespace,
			}, existigRelease)
			existigRelease.Spec.ValuesTemplate = &helmv1alpha1.ValueTemplate{
				ValueRefs: []string{"notpresent"},
			}
			Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

			err = testClient.Update(context.Background(), existigRelease)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			time.Sleep(5 * time.Second)

			release = &helmv1alpha1.Release{}
			releaseChart = &helmv1alpha1.Chart{}
			configmap = &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: "testresource-123", Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "submariner-operator", Namespace: namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&release.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "submariner-operator", Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-default-submariner-operator-0.7.0"))

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: "testresource-123", Namespace: releaseRepo.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "submariner-operator", Namespace: releaseRepo.Namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeTrue())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-submariner-operator-0.7.0", Namespace: releaseKind.Namespace}, configmap),
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

func GetReleaseFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}

func GetConfigMapFunc(ctx context.Context, key client.ObjectKey, obj *v1.ConfigMap) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}
