package helm

import (
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetTestRepoChartVersions returns testcases for testing chart cr with helm chartversion struct
func GetTestRepoChartVersions() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			Input: []*repo.ChartVersion{
				{
					Metadata: &chart.Metadata{
						Name:    "foo",
						Version: "0.0.1",
						Dependencies: []*chart.Dependency{
							{
								Name:       "dep",
								Version:    "0.1.1",
								Repository: "repo",
							},
						},
					},
					URLs: []string{"https://foo.bar/charts/foo-0.0.1.tgz"},
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
			Input: map[string]*helmv1alpha1.Chart{
				"foo": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "",
					},
					Spec: helmv1alpha1.ChartSpec{
						Name: "baz",
						Versions: []helmv1alpha1.ChartVersion{
							{
								Name: "0.0.2",
								URL:  "nodomain.com",
							},
						},
					},
				},
			},
			ReturnError: nil,
			ReturnValue: 1,
		},
		{
			Input: map[string]*helmv1alpha1.Chart{
				"bar": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "",
					},
					Spec: helmv1alpha1.ChartSpec{
						Name: "baz",
						Versions: []helmv1alpha1.ChartVersion{
							{
								Name: "0.0.2",
								URL:  "nodomain.com",
								Dependencies: []*helmv1alpha1.ChartDep{
									{
										Name:    "dep",
										Repo:    "repo",
										Version: "0.1.1",
									},
								},
							},
						},
					},
				},
			},
			ReturnError: nil,
			ReturnValue: 2,
		},
	}
}

// GetTestChartRepo returns repo cr for testing chart cr
func GetTestChartRepo() *helmv1alpha1.Repo {
	return &helmv1alpha1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo",
			Namespace: "",
		},
		Spec: helmv1alpha1.RepoSpec{
			Name: "repo",
		},
	}
}
