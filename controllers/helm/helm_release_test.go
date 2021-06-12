package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

var releaseKind *helmv1alpha1.Release
var release *helmv1alpha1.Release
var releaseChart *helmv1alpha1.Chart
var releaseRepo *helmv1alpha1.Repo

var _ = Context("Install a release", func() {

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
			releaseRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-123",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					Url:  "https://jfelten.github.io/helm-charts/charts",
				},
			}

			err = k8sClient.Create(ctx, releaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			deployment = &helmv1alpha1.Repo{}
			repoChart = &helmv1alpha1.Chart{}
			//configmap := &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(ctx, client.ObjectKey{Name: "testresource-123", Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&deployment.ObjectMeta.Name).To(Equal("testresource-123"))

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "busybox", Namespace: namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&repoChart.ObjectMeta.Name).To(Equal("busybox"))

			/*By("should create a new Release resource with specified")

			releaseKind = &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "deployment-name",
					Chart:   "busybox",
					Repo:    "testresource-123",
					Version: "0.1.0",
				},
			}

			err = k8sClient.Create(ctx, releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(5 * time.Second)

			release = &helmv1alpha1.Release{}
			releaseChart = &helmv1alpha1.Chart{}
			//configmap := &v1.ConfigMap{}

			Eventually(
				GetReleaseFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&release.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "busybox", Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&releaseChart.ObjectMeta.Name).To(Equal("busybox"))

			By("should remove this Release resource with the specified configmaps after deletion")

			err = k8sClient.Delete(ctx, releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				GetReleaseFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "busybox", Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())
			*/
			By("should remove this Repository resource with the specified name and specified url")

			err = k8sClient.Delete(ctx, releaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				GetResourceFunc(ctx, client.ObjectKey{Name: "testresource-123", Namespace: releaseRepo.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "busybox", Namespace: releaseRepo.Namespace}, repoChart),
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

func GetReleaseFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}
