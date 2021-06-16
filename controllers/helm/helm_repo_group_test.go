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

var repoGroupKind *helmv1alpha1.RepoGroup
var repoGroup *helmv1alpha1.RepoGroup
var chart *helmv1alpha1.Chart

var _ = Context("Install a repository group", func() {

	Describe("when no existing resources exist", func() {

		It("should create a new Repository resource with the specified name and specified url", func() {
			ctx := context.Background()
			namespace := "test-" + randStringRunes(7)

			By("should create a new namespace")
			repoGroupNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = k8sClient.Create(ctx, repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Repository resource with the specified name and specified url")
			repoGroupKind = &helmv1alpha1.RepoGroup{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "testresource", Namespace: namespace},
				Spec: helmv1alpha1.RepoGroupSpec{
					LabelSelector: "foo",
					Repos: []helmv1alpha1.RepoSpec{
						{
							Name: "deployment-name-2",
							Url:  "https://submariner-io.github.io/submariner-charts/charts",
							Auth: &helmv1alpha1.Auth{},
						},
					},
				},
			}

			err = k8sClient.Create(ctx, repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			repoGroup = &helmv1alpha1.RepoGroup{}
			chart = &helmv1alpha1.Chart{}
			//configmap := &v1.ConfigMap{}

			Eventually(
				getRepoGroupFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: repoGroupKind.Namespace}, repoGroup),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&repoGroup.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				getChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&chart.ObjectMeta.Name).To(Equal("submariner"))

			By("should add a new Repository resource with the specified name and specified url to the group")

			repoGroupKind.Spec.Repos = append(repoGroupKind.Spec.Repos, helmv1alpha1.RepoSpec{
				Name: "deployment-name-3",
				Url:  "https://rocketchat.github.io/helm-charts",
				Auth: &helmv1alpha1.Auth{},
			})

			err = k8sClient.Update(ctx, repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				getRepoGroupFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: repoGroupKind.Namespace}, repoGroup),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&repoGroup.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				getChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&chart.ObjectMeta.Name).To(Equal("submariner"))

			Eventually(
				getChartFunc(ctx, client.ObjectKey{Name: "rocketchat", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&chart.ObjectMeta.Name).To(Equal("rocketchat"))

			By("should remove the first Repository resource from the group")

			repoGroupKind.Spec.Repos = []helmv1alpha1.RepoSpec{
				{
					Name: "deployment-name-3",
					Url:  "https://rocketchat.github.io/helm-charts",
					Auth: &helmv1alpha1.Auth{},
				},
			}

			err = k8sClient.Update(ctx, repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				getChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				getChartFunc(ctx, client.ObjectKey{Name: "rocketchat", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&chart.ObjectMeta.Name).To(Equal("rocketchat"))

			By("should  remove every repository left when group is deleted")

			err = k8sClient.Delete(ctx, repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				getRepoGroupFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: repoGroupKind.Namespace}, repoGroup),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				getChartFunc(ctx, client.ObjectKey{Name: "rocketchat", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("by deletion of namespace")
			repoGroupNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = k8sClient.Delete(ctx, repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

		})
	})
})

func getResourceFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repo) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}

func getChartFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Chart) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}

func getRepoGroupFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.RepoGroup) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}
