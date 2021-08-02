package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

var repoKind *helmv1alpha1.Repo
var deployment *helmv1alpha1.Repo
var repoChart *helmv1alpha1.Chart

var _ = Context("Install a repository", func() {

	Describe("when no existing resource exist", func() {
		It("should start with creating dependencies", func() {
			ctx := context.Background()
			namespace := "test-" + randStringRunes(7)

			By("install a new namespace")
			repoNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, repoNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("install a new namespace")
			repoSecret := &v1.Secret{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoAuth, Namespace: namespace},
				Data: map[string][]byte{
					"password": []byte("SnU/M2Foc2kK"),
					"user":     []byte("c29lcjNuCg=="),
				},
			}

			err = testClient.Create(ctx, repoSecret)
			Expect(err).NotTo(HaveOccurred(), "failed to create test secret resource")

			By("creating a new repository resource with the specified name and specified url")
			repoKind = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRepoName,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name:       "deployment-name",
					URL:        testRepoURL,
					AuthSecret: testRepoAuth,
				},
			}

			err = testClient.Create(context.Background(), repoKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			deployment = &helmv1alpha1.Repo{}
			repoChart = &helmv1alpha1.Chart{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName, Namespace: repoKind.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testRepoChartNameAssert, Namespace: repoKind.Namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			By("should remove this repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), repoKind)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test resource")

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName, Namespace: repoKind.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testRepoChartNameAssert, Namespace: repoKind.Namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("by deletion of namespace test should finish successfully")
			repoNamespace = &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Delete(context.Background(), repoNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to delete testresource")
		})
	})
})

func GetResourceFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repo) func() error {
	return func() error {
		if err := testClient.Get(ctx, key, obj); err != nil {
			return err
		}

		if len(obj.Status.Conditions) > 0 {
			return nil
		}

		return &errors.StatusError{}
	}
}

func GetChartFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Chart) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}
