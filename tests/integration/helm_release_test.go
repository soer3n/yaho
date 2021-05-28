package tests

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

var releaseKind *helmv1alpha1.Release
var release *helmv1alpha1.Release
var releaseChart *helmv1alpha1.Chart
var releaseRepo *helmv1alpha1.Repo

var _ = Context("Install a release", func() {
	ctx := context.TODO()
	//repoNeeded = true
	ns := SetupTest(ctx)
	namespace = ns.ObjectMeta.Name

	Describe("when no existing resources exist", func() {

		It("should create a new namespace", func() {

			By("should create a new Repository resource with the specified name and specified url")
			releaseRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource-123",
					Namespace: ns.Name,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					Url:  "https://submariner-io.github.io/submariner-charts/charts",
				},
			}

			err := k8sClient.Create(ctx, releaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			deployment = &helmv1alpha1.Repo{}
			repoChart = &helmv1alpha1.Chart{}
			//configmap := &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(ctx, client.ObjectKey{Name: "testresource-123", Namespace: ns.Name}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&deployment.ObjectMeta.Name).To(Equal("testresource-123"))

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: ns.Name}, repoChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&repoChart.ObjectMeta.Name).To(Equal("submariner"))

			By("should create a new Release resource with specified")

			releaseKind = &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: ns.Name,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "deployment-name",
					Chart:   "submariner",
					Repo:    "testresource-123",
					Version: "0.6.0",
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
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&releaseChart.ObjectMeta.Name).To(Equal("submariner"))

			By("should remove this Release resource with the specified configmaps after deletion")

			err = k8sClient.Delete(ctx, releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetReleaseFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("should remove this Repository resource with the specified name and specified url")

			err = k8sClient.Delete(ctx, repoKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				GetResourceFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: repoKind.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: repoKind.Namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())
		})
	})
})

func GetReleaseFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}
