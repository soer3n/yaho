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
	releaseGroupChart           *helmv1alpha1.Chart
	releaseGroupKind            *helmv1alpha1.ReleaseGroup
	releaseFirst, releaseSecond *helmv1alpha1.Release
)

var _ = Context("Install a releasegroup", func() {
	Describe("when no existing resources exist", func() {

		obj := setupNamespace()
		namespace := obj.ObjectMeta.Name

		It("should create a new Repository resource with the specified name and specified url", func() {
			ctx := context.Background()

			// wait on readiness of controllers
			time.Sleep(2 * time.Second)

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

			releaseGroupKind = &helmv1alpha1.ReleaseGroup{}
			releaseFirst = &helmv1alpha1.Release{}
			releaseSecond = &helmv1alpha1.Release{}
			releaseGroupChart = &helmv1alpha1.Chart{}

			By("should create a new Release resource with specified")

			releaseOneAssert := &ReleaseAssert{
				Name: testReleaseName,
				Obj: &helmv1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testReleaseName,
						Namespace: namespace,
					},
				},
				IsPresent: true,
				Synced:    BeTrue(),
				Status:    "success",
				Revision:  1,
			}

			releaseTwoAssert := &ReleaseAssert{
				Name: testReleaseName,
				Obj: &helmv1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testReleaseNameSecond,
						Namespace: namespace,
					},
				},
				IsPresent: false,
			}

			s := "config"

			releaseGroupKind = &helmv1alpha1.ReleaseGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testresource",
					Namespace: namespace,
				},
				Spec: helmv1alpha1.ReleaseGroupSpec{
					Name:          "ReleaseGroup",
					LabelSelector: "select",
					Releases: []helmv1alpha1.ReleaseSpec{
						{
							Name:    testReleaseName,
							Chart:   testReleaseChartName,
							Repo:    testRepoName,
							Version: testReleaseChartVersion,
							Config:  &s,
						},
					},
				},
			}

			err = testClient.Create(context.Background(), releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseOneAssert.Do(namespace)
			releaseTwoAssert.Do(namespace)

			By("should create a second Release resource with specified data")

			err = testClient.Get(context.Background(), client.ObjectKey{Name: releaseGroupKind.Name, Namespace: releaseGroupKind.Namespace}, releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to get test resource")

			releaseGroupKind.Spec = helmv1alpha1.ReleaseGroupSpec{
				Name:          "ReleaseGroup",
				LabelSelector: "select",
				Releases: []helmv1alpha1.ReleaseSpec{
					{
						Name:    testReleaseName,
						Chart:   testReleaseChartName,
						Repo:    testRepoName,
						Version: testReleaseChartVersion,
						Config:  &s,
					},
					{
						Name:    testReleaseNameSecond,
						Chart:   testReleaseChartNameSecond,
						Repo:    testRepoNameSecond,
						Version: testReleaseChartVersionSecond,
						Config:  &s,
					},
				},
			}

			err = testClient.Update(context.Background(), releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseTwoAssert.IsPresent = true
			releaseTwoAssert.Synced = BeTrue()
			releaseTwoAssert.Status = "success"
			releaseTwoAssert.Revision = 1

			releaseOneAssert.Do(namespace)
			releaseTwoAssert.Do(namespace)

			By("should delete first Release by removing it from group list")

			releaseGroupKind.Spec = helmv1alpha1.ReleaseGroupSpec{
				Name:          "ReleaseGroup",
				LabelSelector: "select",
				Releases: []helmv1alpha1.ReleaseSpec{
					{
						Name:    testReleaseNameSecond,
						Chart:   testReleaseChartNameSecond,
						Repo:    testRepoNameSecond,
						Version: testReleaseChartVersionSecond,
						Config:  &s,
					},
				},
			}

			err = testClient.Update(context.Background(), releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseOneAssert.IsPresent = false

			releaseOneAssert.Do(namespace)
			releaseTwoAssert.Do(namespace)

			By("should remove the second Release resource by deleting release group resource")

			err = testClient.Delete(context.Background(), releaseGroupKind)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseTwoAssert.IsPresent = false

			releaseOneAssert.Do(namespace)
			releaseTwoAssert.Do(namespace)

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
