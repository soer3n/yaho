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

var _ = Context("Install a repository", func() {
	ctx := context.TODO()
	ns := SetupTest(ctx)

	Describe("when no existing resources exist", func() {

		It("should create a new Repository resource with the specified name and specified url", func() {
			myKind := &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: ns.Name,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					Url:  "https://submariner-io.github.io/submariner-charts/charts",
				},
			}

			err := k8sClient.Create(ctx, myKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			deployment := &helmv1alpha1.Repo{}
			Eventually(
				getResourceFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: myKind.Namespace}, deployment),
				time.Second*5, time.Millisecond*1500).Should(BeNil())

			Expect(*&deployment.ObjectMeta.Name).To(Equal("testresource"))
		})

	})
})

func getResourceFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repo) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}
