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

	Describe("when no existing resource exist", func() {

		It("should start with creating dependencies", func() {
			ctx := context.Background()
			namespace := "test-" + randStringRunes(7)

			By("install a new namespace")
			repoGroupNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = k8sClient.Create(ctx, repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("creating a new repository group resource with the specified names and specified urls")
			repoGroupKind = &helmv1alpha1.RepoGroup{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: "testresource", Namespace: namespace},
				Spec: helmv1alpha1.RepoGroupSpec{
					LabelSelector: "foo",
					Repos: []helmv1alpha1.RepoSpec{
						{
							Name: "deployment-name-2",
							URL:  "https://submariner-io.github.io/submariner-charts/charts",
							Auth: &helmv1alpha1.Auth{},
						},
					},
				},
			}

			err = k8sClient.Create(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			time.Sleep(5 * time.Second)

			repoGroup = &helmv1alpha1.RepoGroup{}
			chart = &helmv1alpha1.Chart{}

			Eventually(
				getChartFunc(context.Background(), client.ObjectKey{Name: "submariner", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

			By("update group by adding another repository resource with the specified name and specified url")

			repoGroupKind.Spec.Repos = append(repoGroupKind.Spec.Repos, helmv1alpha1.RepoSpec{
				Name: "deployment-name-3",
				URL:  "https://rocketchat.github.io/helm-charts",
				Auth: &helmv1alpha1.Auth{},
			})

			err = k8sClient.Update(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			Eventually(
				getChartFunc(context.Background(), client.ObjectKey{Name: "submariner", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*40, time.Millisecond*1500).Should(BeTrue())

			Eventually(
				getChartFunc(context.Background(), client.ObjectKey{Name: "rocketchat", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*40, time.Millisecond*1500).Should(BeTrue())

			By("should remove the first repository resource from the group")

			repoGroupKind.Spec.Repos = []helmv1alpha1.RepoSpec{
				{
					Name: "deployment-name-3",
					URL:  "https://rocketchat.github.io/helm-charts",
					Auth: &helmv1alpha1.Auth{},
				},
			}

			err = k8sClient.Update(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			Eventually(
				getChartFunc(context.Background(), client.ObjectKey{Name: "submariner", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeTrue())

			Eventually(
				getChartFunc(context.Background(), client.ObjectKey{Name: "rocketchat", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

			By("remove every repository left when group is deleted")

			err = k8sClient.Delete(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")

			Eventually(
				getRepoGroupFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: repoGroupKind.Namespace}, repoGroup),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				getChartFunc(context.Background(), client.ObjectKey{Name: "rocketchat", Namespace: repoGroupKind.Namespace}, chart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeTrue())

			By("by deletion of namespace test should finish successfully")
			repoGroupNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = k8sClient.Delete(context.Background(), repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")

		})
	})
})

func getResourceFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repo) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}

func getChartFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Chart) func() bool {
	return func() bool {
		l := &helmv1alpha1.ChartList{}
		_ = k8sClient.List(ctx, l)

		for _, v := range l.Items {
			if key.Name == v.ObjectMeta.Name {
				return true
			}
		}
		return false
	}
}

func getRepoGroupFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.RepoGroup) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}
