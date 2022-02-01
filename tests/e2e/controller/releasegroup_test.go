package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
)

var releaseGroupChart *helmv1alpha1.Chart
var releaseGroup *helmv1alpha1.ReleaseGroup
var releaseGroupRepo *helmv1alpha1.Repo

var _ = Context("Install a releasegroup", func() {

	Describe("when no existing resources exist", func() {

		It("should create a new Repository resource with the specified name and specified url", func() {
			ctx := context.Background()
			namespace := "test-" + randStringRunes(7)

			By("should create a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Repository resource with the specified name and specified url")
			releaseGroupRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRepoName,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: testRepoName,
					URL:  testRepoURL,
				},
			}

			err = testClient.Create(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			By("should create a new Repository resource with the specified name and specified url")
			releaseGroupRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRepoNameSecond,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: testRepoNameSecond,
					URL:  testRepoURLSecond,
				},
			}

			err = testClient.Create(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			deployment = &helmv1alpha1.Repo{}
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
							ValuesTemplate: &helmv1alpha1.ValueTemplate{
								ValueRefs: []string{
									"notpresent",
								},
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
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartNameSecond, Namespace: releaseGroupKind.Namespace}, releaseGroupChart),
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

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			By("should remove this Repository resource with the specified name and specified url")

			releaseGroupRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRepoName,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: testRepoName,
					URL:  testRepoURL,
				},
			}

			err = testClient.Delete(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName, Namespace: releaseGroupRepo.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName, Namespace: releaseGroupRepo.Namespace}, releaseGroupChart),
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
