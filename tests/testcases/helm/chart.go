package helm

import (
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetTestRepoChartVersions returns testcases for testing chart cr with helm chartversion struct
func GetTestRepoChartVersions() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			Input: &helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "foo",
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "foo",
					Repository: "repo",
					Versions:   []string{"0.0.1"},
					CreateDeps: true,
				},
			},
			ReturnValue: "",
		},
	}
}

// GetTestHelmChartMaps returns testcases for testing chart cr
func GetTestHelmChartMaps() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			Input: &helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "",
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "baz",
					Repository: "foo",
					Versions: []string{
						"0.0.2",
					},
				},
			},
			ReturnError: nil,
			ReturnValue: 1,
		},
		{
			Input: &helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "",
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "baz",
					Repository: "foo",
					Versions: []string{
						"0.0.2",
					},
				},
			},
			ReturnError: nil,
			ReturnValue: 2,
		},
	}
}

// GetTestChartRepo returns repo cr for testing chart cr
func GetTestChartRepo() *helmv1alpha1.Repository {
	return &helmv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo",
			Namespace: "",
		},
		Spec: helmv1alpha1.RepositorySpec{
			Name: "repo",
		},
	}
}
