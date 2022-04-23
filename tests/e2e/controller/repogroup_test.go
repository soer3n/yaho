package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	repoGroupKind *helmv1alpha1.RepoGroup
	repoGroup     *helmv1alpha1.RepoGroup
	chart         *helmv1alpha1.Chart
)

var _ = Context("Install a repository group", func() {
	Describe("when no existing resource exist", func() {

		obj := setupNamespace()
		namespace := obj.ObjectMeta.Name

		It("should start with creating dependencies", func() {
			ctx := context.Background()

			By("install a new namespace")
			repoGroupNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("creating a new repository group resource with the specified names and specified urls")
			repoGroupKind = &helmv1alpha1.RepoGroup{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoName},
				Spec: helmv1alpha1.RepoGroupSpec{
					LabelSelector: "foo",
					Repos: []helmv1alpha1.RepositorySpec{
						{
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
					},
				},
			}

			err = testClient.Create(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoGroup = &helmv1alpha1.RepoGroup{}
			chart = &helmv1alpha1.Chart{}

			By("update group by adding another repository resource with the specified name and specified url")

			repoGroupKind.Spec.Repos = append(repoGroupKind.Spec.Repos, helmv1alpha1.RepositorySpec{
				Name: testRepoNameSecond,
				URL:  testRepoURLSecond,
				Charts: []helmv1alpha1.Entry{
					{
						Name:     "testing-dep",
						Versions: []string{"0.1.0"},
					},
				},
			})

			err = testClient.Update(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			deployment = &helmv1alpha1.Repository{}
			deployment2 := &helmv1alpha1.Repository{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName}, deployment2),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testRepoChartNameAssert + "-" + testRepoName}, chart),
				time.Second*40, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testRepoChartSecondNameAssert + "-" + testRepoNameSecond}, chart),
				time.Second*40, time.Millisecond*1500).Should(BeNil())

			By("should remove the first repository resource from the group")

			repoGroupKind.Spec.Repos = []helmv1alpha1.RepositorySpec{
				{
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

			err = testClient.Update(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoNameSecond}, deployment2),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testRepoChartNameAssert + "-" + testRepoName}, chart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testRepoChartSecondNameAssert + "-" + testRepoNameSecond}, chart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			By("remove every repository left when group is deleted")

			err = testClient.Delete(context.Background(), repoGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")

			Eventually(
				getRepoGroupFunc(context.Background(), client.ObjectKey{Name: testRepoNameSecond}, repoGroup),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testRepoChartSecondNameAssert + "-" + testRepoNameSecond}, chart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("by deletion of namespace test should finish successfully")
			repoGroupNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Delete(context.Background(), repoGroupNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")
		})
	})
})

func getRepoGroupFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.RepoGroup) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}
