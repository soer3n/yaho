package utils

import (
	"context"
	"time"

	. "github.com/onsi/gomega"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var k8sClient client.Client
var ns *v1.Namespace
var repoKind *helmv1alpha1.Repo
var deployment *helmv1alpha1.Repo
var repoChart *helmv1alpha1.Chart

func CreateRepo(namespace string) func() {
	ctx := context.TODO()
	return func() {
		repoKind = &helmv1alpha1.Repo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testresource",
				Namespace: namespace,
			},
			Spec: helmv1alpha1.RepoSpec{
				Name: "deployment-name",
				Url:  "https://submariner-io.github.io/submariner-charts/charts",
			},
		}

		err := k8sClient.Create(ctx, repoKind)
		Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

		time.Sleep(1 * time.Second)

		deployment = &helmv1alpha1.Repo{}
		repoChart = &helmv1alpha1.Chart{}
		//configmap := &v1.ConfigMap{}

		Eventually(
			GetResourceFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: repoKind.Namespace}, deployment),
			time.Second*20, time.Millisecond*1500).Should(BeNil())

		Expect(*&deployment.ObjectMeta.Name).To(Equal("testresource"))

		Eventually(
			GetChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: repoKind.Namespace}, repoChart),
			time.Second*20, time.Millisecond*1500).Should(BeNil())

		Expect(*&repoChart.ObjectMeta.Name).To(Equal("submariner"))

	}
}

func DeleteRepo() func() {
	ctx := context.TODO()
	return func() {
		err := k8sClient.Delete(ctx, repoKind)
		Expect(err).NotTo(HaveOccurred(), "failed to create test MyKind resource")

		time.Sleep(1 * time.Second)

		Eventually(
			GetResourceFunc(ctx, client.ObjectKey{Name: "testresource", Namespace: repoKind.Namespace}, deployment),
			time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

		Eventually(
			GetChartFunc(ctx, client.ObjectKey{Name: "submariner", Namespace: repoKind.Namespace}, repoChart),
			time.Second*20, time.Millisecond*1500).ShouldNot(BeNil())

	}
}

func GetResourceFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Repo) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}

func GetChartFunc(ctx context.Context, key client.ObjectKey, obj *helmv1alpha1.Chart) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}
