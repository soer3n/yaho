package helm

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
)

var releaseKind *helmv1alpha1.Release
var release *helmv1alpha1.Release
var releaseChart *helmv1alpha1.Chart
var releaseRepo, releaseRepoSecond *helmv1alpha1.Repo

var _ = Context("Install a release", func() {

	Describe("when no existing resources exist", func() {

		It("should start with creating dependencies", func() {
			ctx := context.Background()
			namespace := "test-" + randStringRunes(7)

			By("install a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			By("creating a new repository resource with the specified name and specified url")
			releaseRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRepoName,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					URL:  testRepoURL,
				},
			}

			deployment = &helmv1alpha1.Repo{}
			repoChart = &helmv1alpha1.Chart{}
			releaseChart = &helmv1alpha1.Chart{}

			err = testClient.Create(context.Background(), releaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseRepoSecond = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testRepoNameSecond,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: testRepoNameSecond,
					URL:  testRepoURLSecond,
				},
			}

			err = testClient.Create(context.Background(), releaseRepoSecond)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName, Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName, Namespace: namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName, Namespace: namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			By("creating a new release resource with specified data")

			releaseKind = &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testReleaseName,
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name: "deployment-name",
					Namespace: helmv1alpha1.Namespace{
						Name:    namespace,
						Install: true,
					},
					Chart:   testReleaseChartName,
					Repo:    testRepoName,
					Version: testReleaseChartVersion,
					Flags: &helmv1alpha1.Flags{
						Atomic: false,
					},
				},
			}

			err = testClient.Create(context.Background(), releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			release = &helmv1alpha1.Release{}
			configmap := &v1.ConfigMap{}

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: testReleaseName, Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(release.ObjectMeta.Name).To(Equal(testReleaseName))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion))

			By("should update this Release resource with values reference")

			existigRelease := &helmv1alpha1.Release{}
			err = testClient.Get(context.Background(), types.NamespacedName{
				Name:      testReleaseName,
				Namespace: namespace,
			}, existigRelease)
			existigRelease.Spec.ValuesTemplate = &helmv1alpha1.ValueTemplate{
				ValueRefs: []string{"notpresent"},
			}
			Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

			err = testClient.Update(context.Background(), existigRelease)
			Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

			time.Sleep(3 * time.Second)

			release = &helmv1alpha1.Release{}
			releaseChart = &helmv1alpha1.Chart{}
			configmap = &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName, Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName, Namespace: namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: testReleaseName, Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(*&release.ObjectMeta.Name).To(Equal(testReleaseName))

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName, Namespace: releaseKind.Namespace}, releaseChart),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion))

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), releaseKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			Eventually(
				GetReleaseFunc(context.Background(), client.ObjectKey{Name: testReleaseName, Namespace: releaseKind.Namespace}, release),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: testRepoName, Namespace: releaseRepo.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName, Namespace: releaseRepo.Namespace}, repoChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: releaseKind.Namespace}, configmap),
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

func GetReleaseFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Release) func() error {
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

func GetConfigMapFunc(ctx context.Context, key client.ObjectKey, obj *v1.ConfigMap) func() error {
	return func() error {
		return testClient.Get(ctx, key, obj)
	}
}
