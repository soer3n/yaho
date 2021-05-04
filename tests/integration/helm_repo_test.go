package tests

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

var _ = Context("Inside of a new namespace", func() {
	ctx := context.TODO()
	ns := SetupTest(ctx)

	Describe("when no existing resources exist", func() {

		It("should create a new Deployment resource with the specified name and one replica if none is provided", func() {
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
				time.Second*5, time.Millisecond*500).Should(BeNil())

			Expect(*&deployment.ObjectMeta.Name).To(Equal("testresource"))
		})

		It("should create a new Deployment resource with the specified name and two replicas if two is specified", func() {
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
				time.Second*5, time.Millisecond*500).Should(BeNil())

			Expect(*&deployment.ObjectMeta.Name).To(Equal("testresource"))
		})

		It("should allow updating the replicas count after creating a MyKind resource", func() {
			deploymentObjectKey := client.ObjectKey{
				Name:      "deployment-name",
				Namespace: ns.Name,
			}
			myKindObjectKey := client.ObjectKey{
				Name:      "testresource",
				Namespace: ns.Name,
			}
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
				getResourceFunc(ctx, deploymentObjectKey, deployment),
				time.Second*5, time.Millisecond*500).Should(BeNil(), "deployment resource should exist")

			Expect(*&deployment.ObjectMeta.Name).To(Equal("deployment-name"))

			err = k8sClient.Get(ctx, myKindObjectKey, myKind)
			Expect(err).NotTo(HaveOccurred(), "failed to retrieve MyKind resource")

			// myKind.Spec.Replicas = pointer.Int32Ptr(2)
			err = k8sClient.Update(ctx, myKind)
			Expect(err).NotTo(HaveOccurred(), "failed to Update MyKind resource")

			Eventually(getDeploymentReplicasFunc(ctx, deploymentObjectKey)).
				Should(Equal(int32(2)), "expected Deployment resource to be scale to 2 replicas")
		})

		It("should clean up an old Deployment resource if the deploymentName is changed", func() {
			deploymentObjectKey := client.ObjectKey{
				Name:      "deployment-name",
				Namespace: ns.Name,
			}
			newDeploymentObjectKey := client.ObjectKey{
				Name:      "new-deployment",
				Namespace: ns.Name,
			}
			myKindObjectKey := client.ObjectKey{
				Name:      "testresource",
				Namespace: ns.Name,
			}
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
				getResourceFunc(ctx, deploymentObjectKey, deployment),
				time.Second*5, time.Millisecond*500).Should(BeNil(), "deployment resource should exist")

			err = k8sClient.Get(ctx, myKindObjectKey, myKind)
			Expect(err).NotTo(HaveOccurred(), "failed to retrieve MyKind resource")

			myKind.Spec.Name = newDeploymentObjectKey.Name
			err = k8sClient.Update(ctx, myKind)
			Expect(err).NotTo(HaveOccurred(), "failed to Update MyKind resource")

			Eventually(
				getResourceFunc(ctx, deploymentObjectKey, deployment),
				time.Second*5, time.Millisecond*500).ShouldNot(BeNil(), "old deployment resource should be deleted")

			Eventually(
				getResourceFunc(ctx, newDeploymentObjectKey, deployment),
				time.Second*5, time.Millisecond*500).Should(BeNil(), "new deployment resource should be created")
		})
	})
})

func getResourceFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repo) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}

func getDeploymentReplicasFunc(ctx context.Context, key client.ObjectKey) func() int32 {
	return func() int32 {
		depl := &apps.Deployment{}
		err := k8sClient.Get(ctx, key, depl)
		Expect(err).NotTo(HaveOccurred(), "failed to get Deployment resource")

		return *depl.Spec.Replicas
	}
}
