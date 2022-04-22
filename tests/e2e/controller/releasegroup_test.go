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
	releaseGroupChart                        *helmv1alpha1.Chart
	releaseGroup                             *helmv1alpha1.ReleaseGroup
	releaseGroupRepo, releaseGroupRepoSecond *helmv1alpha1.Repository
)

var _ = Context("Install a releasegroup", func() {
	Describe("when no existing resources exist", func() {

		obj := setupNamespace()
		namespace := obj.ObjectMeta.Name

		It("should create a new Repository resource with the specified name and specified url", func() {
			ctx := context.Background()
			// namespace = "test-" + randStringRunes(7)

			By("should create a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Repository resource with the specified name and specified url")
			releaseGroupRepo = &helmv1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: testRepoName,
					// Namespace: namespace,
					Labels: map[string]string{"repoGroup": "foo"},
				},
				Spec: helmv1alpha1.RepositorySpec{
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
			}

			err = testClient.Create(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Repository resource with the specified name and specified url")
			releaseGroupRepoSecond = &helmv1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name: testRepoNameSecond,
					// Namespace: namespace,
					Labels: map[string]string{"repoGroup": "foo"},
				},
				Spec: helmv1alpha1.RepositorySpec{
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

			err = testClient.Create(context.Background(), releaseGroupRepoSecond)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			deployment = &helmv1alpha1.Repository{}
			releaseGroupChart = &helmv1alpha1.Chart{}
			configmap := &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName, Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName, Namespace: namespace}, releaseGroupChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			By("should create a new Release resource with specified")

			releaseGroupKind := &helmv1alpha1.ReleaseGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseGroupSpec{
					Name: "ReleaseGroup",
					// LabelSelector: "select",
					Releases: []helmv1alpha1.ReleaseSpec{
						{
							Name:    testReleaseName,
							Chart:   testReleaseChartName,
							Repo:    testRepoName,
							Version: testReleaseChartVersion,
						},
						{
							Name:    testReleaseNameSecond,
							Chart:   testReleaseChartNameSecond,
							Repo:    testRepoNameSecond,
							Version: testReleaseChartVersionSecond,
							Values: []string{
								"notpresent",
							},
						},
					},
				},
			}

			err = testClient.Create(context.Background(), releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseGroup = &helmv1alpha1.ReleaseGroup{}
			releaseGroupChart = &helmv1alpha1.Chart{}

			Eventually(
				GetReleaseGroupFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseGroupKind.Namespace}, releaseGroup),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(releaseGroup.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartNameSecond}, releaseGroupChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetReleaseGroupFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseGroupKind.Namespace}, releaseGroup),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			rel := &helmv1alpha1.Release{}
			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseGroupKind.Namespace}, rel),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: "testresource-2", Namespace: releaseGroupKind.Namespace}, rel),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseGroupRepoSecond)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoNameSecond}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName}, releaseGroupChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond, Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartNameSecond + "-" + testReleaseChartVersionSecond, Namespace: releaseGroupKind.Namespace}, configmap),
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

func GetReleaseGroupFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.ReleaseGroup) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}
