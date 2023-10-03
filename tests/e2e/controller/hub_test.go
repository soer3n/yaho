package helm

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	releaseHubRepoOne, releaseHubRepoTwo *yahov1alpha2.Repository
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

			kubeconfigPath := os.Getenv("KUBECONFIG")
			if kubeconfigPath == "" {
				kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
			}
			SetupKubeconfigSecret(kubeconfigPath, "https://127.0.0.1:6443", "yaho-local-kubeconfig", namespace)

			chartOneAssert := &ChartAssert{
				Name:               testRepoChartNameAssert,
				Version:            testRepoChartNameAssertqVersion,
				IsPresent:          BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-testresource")),
				IndicesInstalled:   BeNil(),
				ResourcesInstalled: BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present")),
				Synced:             BeFalse(),
				Deps:               BeFalse(),
			}

			chartTwoAssert := &ChartAssert{
				Name:               testRepoChartSecondNameAssert,
				Version:            testRepoChartSecondNameAssertVersion,
				IsPresent:          BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-dep-testresource-2")),
				IndicesInstalled:   BeNil(),
				ResourcesInstalled: BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present")),
				Synced:             BeFalse(),
				Deps:               BeFalse(),
			}

			chartThreeAssert := &ChartAssert{
				Name:               testRepoChartThirdNameAssert,
				Version:            testRepoChartThirdNameAssertVersion,
				IsPresent:          BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "charts", Group: "yaho.soer3n.dev"}, "testing-nested-testresource")),
				IndicesInstalled:   BeNil(),
				ResourcesInstalled: BeEquivalentTo(k8serrors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "related configmaps not present")),
				Synced:             BeFalse(),
				Deps:               BeFalse(),
			}

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

			chartOneAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartOneAssert.Name}}
			chartTwoAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartTwoAssert.Name}}
			chartThreeAssert.Obj = &yahov1alpha2.Chart{ObjectMeta: metav1.ObjectMeta{Name: chartThreeAssert.Name}}

			By("setup requirements")

			SetupRBAC(namespace)
			SetupConfig(namespace)

			time.Sleep(3 * time.Second)

			By("creating needed hub resource")
			enableDeployment := false

			localHub := &yahov1alpha2.Hub{
				ObjectMeta: metav1.ObjectMeta{
					Name: "local",
				},
				Spec: yahov1alpha2.HubSpec{
					Interval: "10s",
					HubSelector: yahov1alpha2.HubSelector{
						Kind:      "Secret",
						Namespace: namespace,
						Labels: map[string]string{
							"yaho.soer3n.dev/hub": "local",
						},
					},
					Agent: &yahov1alpha2.HubClusterAgent{
						Deploy: &enableDeployment,
					},
				},
			}

			err := testClient.Create(context.TODO(), localHub)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			By("creating needed repository resources")

			releaseHubRepoOne = &yahov1alpha2.Repository{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoName},
				Spec: yahov1alpha2.RepositorySpec{
					Source: yahov1alpha2.RepositorySource{
						URL:  testRepoURL,
						Type: "helm",
					},
					Charts: yahov1alpha2.RepositoryCharts{
						Sync: yahov1alpha2.Sync{
							Enabled:  true,
							Interval: "10s",
						},
						Items: []yahov1alpha2.Entry{},
					},
				},
			}

			err = testClient.Create(ctx, releaseHubRepoOne)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			releaseHubRepoTwo = &yahov1alpha2.Repository{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{Name: testRepoNameSecond},
				Spec: yahov1alpha2.RepositorySpec{
					Source: yahov1alpha2.RepositorySource{
						URL:  testRepoURLSecond,
						Type: "helm",
					},
					Charts: yahov1alpha2.RepositoryCharts{
						Sync: yahov1alpha2.Sync{
							Enabled:  true,
							Interval: "10s",
						},
						Items: []yahov1alpha2.Entry{},
					},
				},
			}

			err = testClient.Create(ctx, releaseHubRepoTwo)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			repoOneAssert.setEverythingInstalled()
			repoTwoAssert.setEverythingInstalled()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			By("creating a new release resource with a valid repository")

			releaseAssert := &ReleaseAssert{
				Name: testReleaseName,
				Obj: &yahov1alpha2.Release{
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

			s := "config"

			releaseAssert.Obj.Spec = yahov1alpha2.ReleaseSpec{
				Name:      "deployment-name",
				Namespace: &namespace,
				Config:    &s,
				Chart:     testReleaseChartName,
				Repo:      testRepoName,
				Version:   testReleaseChartVersion,
			}

			err = testClient.Create(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			chartOneAssert.setEverythingInstalled()

			repoOneAssert.Do(namespace)
			repoTwoAssert.Do(namespace)

			releaseAssert.Status = "success"
			releaseAssert.Revision = 1
			releaseAssert.Synced = BeTrue()

			releaseAssert.Do(namespace)

			By("should remove this Release resource with the specified configmaps after deletion")

			err = testClient.Delete(context.Background(), releaseAssert.Obj)
			Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

			releaseAssert.IsPresent = false

			By("should remove this Repository resource with the specified name and specified url")

			err = testClient.Delete(context.Background(), releaseHubRepoOne)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

			err = testClient.Delete(context.Background(), releaseHubRepoTwo)
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

			err = testClient.Delete(context.Background(), localHub)
			Expect(err).NotTo(HaveOccurred(), "failed to create test resource")

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
