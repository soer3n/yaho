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

var releaseGroupKind *helmv1alpha1.ReleaseGroup
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

			time.Sleep(2 * time.Second)

			By("should create a new Repository resource with the specified name and specified url")
			releaseGroupRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-releasegroup-123",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					URL:  "https://submariner-io.github.io/submariner-charts/charts",
				},
			}

			err = testClient.Create(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(3 * time.Second)

			By("should create a new Repository resource with the specified name and specified url")
			releaseGroupRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-releasegroup-321",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name2",
					URL:  "https://jfelten.github.io/helm-charts/charts",
				},
			}

			err = testClient.Create(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(5 * time.Second)

			deployment = &helmv1alpha1.Repo{}
			releaseGroupChart = &helmv1alpha1.Chart{}
			configmap := &v1.ConfigMap{}

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: "test-releasegroup-123", Namespace: namespace}, deployment),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "submariner-operator", Namespace: namespace}, releaseGroupChart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

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
							Name:    "deployment-name",
							Chart:   "submariner-operator",
							Repo:    "test-releasegroup-123",
							Version: "0.7.0",
						},
						{
							Name:    "deployment-name2",
							Chart:   "busybox",
							Repo:    "test-releasegroup-321",
							Version: "0.1.0",
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

			time.Sleep(5 * time.Second)

			releaseGroup = &helmv1alpha1.ReleaseGroup{}
			releaseGroupChart = &helmv1alpha1.Chart{}

			Eventually(
				GetReleaseGroupFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseGroupKind.Namespace}, releaseGroup),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(releaseGroup.ObjectMeta.Name).To(Equal("testresource"))

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "busybox", Namespace: releaseGroupKind.Namespace}, releaseGroupChart),
				time.Second*20, time.Millisecond*1500).Should(BeTrue())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-submariner-operator-0.7.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-submariner-operator-0.7.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-submariner-operator-0.7.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-submariner-operator-0.7.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-busybox-0.1.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-busybox-0.1.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-busybox-0.1.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-busybox-0.1.0"))

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-busybox-0.1.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).Should(BeNil())

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(10 * time.Second)

			Eventually(
				GetReleaseGroupFunc(context.Background(), client.ObjectKey{Name: "testresource", Namespace: releaseGroupKind.Namespace}, releaseGroup),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test MyKind resource")

			By("should remove this Repository resource with the specified name and specified url")

			releaseGroupRepo = &helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-releasegroup-123",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "deployment-name",
					URL:  "https://submariner-io.github.io/submariner-charts/charts",
				},
			}

			err = testClient.Delete(context.Background(), releaseGroupRepo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			time.Sleep(1 * time.Second)

			Eventually(
				GetResourceFunc(context.Background(), client.ObjectKey{Name: "testresource-123", Namespace: releaseGroupRepo.Namespace}, deployment),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetChartFunc(context.Background(), client.ObjectKey{Name: "submariner-operator", Namespace: releaseGroupRepo.Namespace}, releaseGroupChart),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeTrue())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-submariner-operator-0.7.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-submariner-operator-0.7.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-submariner-operator-0.7.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-busybox-0.1.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-busybox-0.1.0", Namespace: releaseGroupKind.Namespace}, configmap),
				time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

			Eventually(
				GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-busybox-0.1.0", Namespace: releaseGroupKind.Namespace}, configmap),
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
