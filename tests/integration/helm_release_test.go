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

var _ = Context("Install a release", func() {
	ctx := context.TODO()
	ns = SetupTest(ctx)

	Describe("when no existing resources exist", func() {
		It("should create a new Release resource with specified", func() {
			PrepareReleaseTest(ctx)

			releaseKind = &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: ns.Name,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "deployment-name",
					Chart:   "submariner",
					Repo:    "submariner",
					Version: "0.6.0",
				},
			}

			err := k8sClient.Create(ctx, releaseKind)
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

		})

		It("should remove this Release resource with the specified configmaps after deletion", func() {

			err = k8sClient.Delete(ctx, releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(5 * time.Second)

			CleanUpReleaseTest(ctx)

			Eventually(
				GetReleaseFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())
		})

	})
})

func GetReleaseFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}
