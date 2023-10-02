package helm

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	releaseRepoGroup *helmv1alpha1.RepoGroup
)

var _ = Context("Install a release", func() {
	Describe("when no existing resources exist", func() {

		obj := setupNamespace()
		namespace := obj.ObjectMeta.Name

		It("should start with creating dependencies", func() {
			ctx := context.Background()

			By("install a new namespace")
			releaseNamespace := &v1.Namespace{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}

			err = testClient.Create(ctx, releaseNamespace)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert := &ChartAssert{
				Name:    testRepoChartNameAssert,
				Version: testRepoChartNameAssertqVersion,
			}

			chartOneAssert.setDefault(testRepoName)

			chartTwoAssert := &ChartAssert{
				Name:    testRepoChartSecondNameAssert,
				Version: testRepoChartSecondNameAssertVersion,
			}

			chartTwoAssert.setDefault(testRepoNameSecond)

			chartThreeAssert := &ChartAssert{
				Name:    testRepoChartThirdNameAssert,
				Version: testRepoChartThirdNameAssertVersion,
			}

			chartThreeAssert.setDefault(testRepoName)

			repoOneAssert := RepositoryAssert{
				Name:          testRepoName,
				ManagedCharts: []*ChartAssert{chartOneAssert, chartThreeAssert},
			}

			repoOneAssert.setDefault()

			repoTwoAssert := RepositoryAssert{
				Name:          testRepoNameSecond,
				ManagedCharts: []*ChartAssert{chartTwoAssert},
			}

			repoTwoAssert.setDefault()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			SetupRBAC(namespace)
			SetupConfig(namespace)

			By("creating needed repository group resource")

			releaseRepoGroup = &helmv1alpha1.RepoGroup{
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
									Name:     testReleaseChartName,
									Versions: []string{testReleaseChartVersion},
								},
								{
									Name:     testReleaseChartThirdNameAssert,
									Versions: []string{testReleaseChartThirdNameAssertVersion},
								},
							},
						},
						{
							Name: testRepoNameSecond,
							URL:  testRepoURLSecond,
							Charts: []helmv1alpha1.Entry{
								{
									Name:     testReleaseChartNameSecond,
									Versions: []string{testReleaseChartVersionSecond},
								},
							},
						},
					},
				},
			}

			err = testClient.Create(ctx, releaseRepoGroup)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert.setEverythingInstalled()
			chartTwoAssert.setEverythingInstalled()
			chartThreeAssert.setEverythingInstalled()

			repoOneAssert.setEverythingInstalled()
			repoTwoAssert.setEverythingInstalled()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating a new release resource with not valid repository")

			releaseAssert := &ReleaseAssert{
				Name: testReleaseName,
				Obj: &helmv1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testReleaseName,
						Namespace: namespace,
					},
				},
				IsPresent: true,
				Synced:    BeFalse(),
				Status:    "initError",
				Revision:  0,
			}

			releaseAssert.Obj.Spec = helmv1alpha1.ReleaseSpec{
				Name:      "deployment-name",
				Namespace: &namespace,
				Chart:     testReleaseChartName,
				Repo:      "fail",
				Version:   testReleaseChartVersion,
			}

			err = testClient.Create(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseAssert.Do(namespace)

			By("updating the release resource with not valid chart name")

			releaseAssert.Obj.Spec = helmv1alpha1.ReleaseSpec{
				Name:      "deployment-name",
				Namespace: &namespace,
				Chart:     testReleaseChartName + "foo",
				Repo:      testRepoName,
				Version:   testReleaseChartVersion,
			}

			err = testClient.Update(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseAssert.Do(namespace)

			By("updating the release resource with not valid chart version")

			releaseAssert.Obj.Spec = helmv1alpha1.ReleaseSpec{
				Name:      "deployment-name",
				Namespace: &namespace,
				Chart:     testReleaseChartName,
				Repo:      testRepoName,
				Version:   testReleaseChartNotValidVersion,
			}

			err = testClient.Update(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseAssert.Do(namespace)

			By("updating the release resource with wrong config name")

			s := "foo"

			releaseAssert.Obj.Spec = helmv1alpha1.ReleaseSpec{
				Name:      "deployment-name",
				Namespace: &namespace,
				Config:    &s,
				Chart:     testReleaseChartName,
				Repo:      testRepoName,
				Version:   testReleaseChartNotValidVersion,
			}

			err = testClient.Update(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseAssert.Do(namespace)

			By("updating the release resource with not allowed namespace")

			s = "config"
			n := "notallowed"

			releaseAssert.Obj.Spec = helmv1alpha1.ReleaseSpec{
				Name:      "deployment-name",
				Namespace: &n,
				Config:    &s,
				Chart:     testReleaseChartName,
				Repo:      testRepoName,
				Version:   testReleaseChartNotValidVersion,
			}

			err = testClient.Update(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseAssert.Do(namespace)

			By("updating release resource with valid data")

			releaseAssert.Obj.Spec = helmv1alpha1.ReleaseSpec{
				Name:      "deployment-name",
				Namespace: &namespace,
				Config:    &s,
				Chart:     testReleaseChartName,
				Repo:      testRepoName,
				Version:   testReleaseChartVersion,
			}

			err = testClient.Update(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseAssert.Status = "success"
			releaseAssert.Revision = 1
			releaseAssert.Synced = BeTrue()

			releaseAssert.Do(namespace)

			/*
				By("should update this Release resource with values reference")

				existigRelease := &helmv1alpha1.Release{}
				err = testClient.Get(context.Background(), types.NamespacedName{
					Name:      testReleaseName,
					Namespace: namespace,
				}, existigRelease)
				existigRelease.Spec.Values = []string{"notpresent"}
				Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

				err = testClient.Update(context.Background(), existigRelease)
				Expect(err).NotTo(HaveOccurred(), "failed to update test resource")

				time.Sleep(3 * time.Second)

				release = &helmv1alpha1.Release{}
				releaseChart = &helmv1alpha1.Chart{}
				configmap = &v1.ConfigMap{}

				Eventually(
					GetRepositoryFunc(context.Background(), client.ObjectKey{Name: testRepoName}, deployment),
					time.Second*20, time.Millisecond*1500).Should(BeNil())

				Eventually(
					GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName + "-" + testRepoName}, repoChart),
					time.Second*20, time.Millisecond*1500).Should(BeNil())

				Eventually(
					GetReleaseFunc(context.Background(), client.ObjectKey{Name: testReleaseName, Namespace: release.Namespace}, release),
					time.Second*20, time.Millisecond*1500).Should(BeNil())

				Expect(release.ObjectMeta.Name).To(Equal(testReleaseName))

				Eventually(
					GetChartFunc(context.Background(), client.ObjectKey{Name: testReleaseChartName + "-" + testRepoName}, releaseChart),
					time.Second*20, time.Millisecond*1500).Should(BeNil())

				Eventually(
					GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: release.Namespace}, configmap),
					time.Second*20, time.Millisecond*1500).Should(BeNil())

				Expect(configmap.ObjectMeta.Name).To(Equal("helm-tmpl-" + testReleaseChartName + "-" + testReleaseChartVersion))

				Eventually(
					GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: release.Namespace}, configmap),
					time.Second*20, time.Millisecond*1500).Should(BeNil())

				Expect(configmap.ObjectMeta.Name).To(Equal("helm-crds-" + testReleaseChartName + "-" + testReleaseChartVersion))

				Eventually(
					GetConfigMapFunc(context.Background(), client.ObjectKey{Name: "helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion, Namespace: release.Namespace}, configmap),
					time.Second*20, time.Millisecond*1500).Should(BeNil())

				Expect(configmap.ObjectMeta.Name).To(Equal("helm-default-" + testReleaseChartName + "-" + testReleaseChartVersion))
			*/

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseAssert.IsPresent = false

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseRepoGroup)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert.setDefault(testRepoName)
			chartTwoAssert.setDefault(testRepoNameSecond)
			chartThreeAssert.setDefault(testRepoName)

			repoOneAssert.setDefault()
			repoTwoAssert.setDefault()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			RemoveRBAC(namespace)
			RemoveConfig(namespace)

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
