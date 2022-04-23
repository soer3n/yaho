package helm

import (
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

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
					Namespace: "one",
					Labels: map[string]string{
						"repo": "one",
					},
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "bar",
					Repository: "one",
					Versions: []string{
						"0.0.1",
					},
				},
			},
			ChartVersion: "0.0.1",
			ReturnError: map[string]error{
				"init":         nil,
				"prepare":      nil,
				"update":       nil,
				"subResources": nil,
				"subCharts":    nil,
			},
			ReturnValue: 1,
		},
		{
			Input: &helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "baz",
					Namespace: "two",
					Labels: map[string]string{
						"repo": "two",
					},
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "baz",
					Repository: "two",
					Versions: []string{
						"0.0.2",
					},
				},
			},
			ChartVersion: "0.0.2",
			ReturnError: map[string]error{
				"init":         nil,
				"prepare":      nil,
				"update":       nil,
				"subResources": nil,
				"subCharts":    nil,
			},
			ReturnValue: 2,
		},
		{
			Input: &helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "three",
					Labels: map[string]string{
						"repo": "three",
					},
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "bar",
					Repository: "three",
					Versions: []string{
						"0.0.3",
					},
				},
			},
			ChartVersion: "0.0.3",
			ReturnError: map[string]error{
				"init":         nil,
				"prepare":      nil,
				"update":       nil,
				"subResources": nil,
				"subCharts":    nil,
			},
			ReturnValue: 2,
		},
		{
			Input: &helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "four",
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "foo",
					Repository: "four",
					Versions:   []string{"0.0.4"},
					CreateDeps: true,
				},
			},
			ReturnValue:  "",
			ChartVersion: "0.0.4",
			ReturnError: map[string]error{
				"init":         nil,
				"prepare":      k8serrors.NewBadRequest("could not load subchart testing-dep"),
				"update":       k8serrors.NewBadRequest("could not load subchart testing-dep"),
				"subResources": nil,
				"subCharts":    nil,
			},
		},
		{
			Input: &helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "five",
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       "foo",
					Repository: "five",
					Versions:   []string{"0.0.5"},
					CreateDeps: true,
				},
			},
			ReturnValue:  "",
			ChartVersion: "0.0.5",
			ReturnError: map[string]error{
				"init":         nil,
				"prepare":      k8serrors.NewBadRequest("could not load subchart testing-dep"),
				"update":       k8serrors.NewBadRequest("could not load subchart testing-dep"),
				"subResources": nil,
				"subCharts":    nil,
			},
		},
	}
}

func GetChartVersions(name string) repo.ChartVersions {
	return repo.ChartVersions{
		{
			Metadata: &chart.Metadata{
				Name:       name,
				Version:    "0.0.1",
				APIVersion: "v2",
			},
			URLs: []string{"https://foo.bar/charts/" + name + "-0.0.1.tgz"},
		},
		{
			Metadata: &chart.Metadata{
				Name:       name,
				Version:    "0.0.2",
				APIVersion: "v2",
				Dependencies: []*chart.Dependency{
					{
						Name:       "testing-dep",
						Version:    "0.1.0",
						Repository: "https://foo.bar/charts/",
					},
				},
			},
			URLs: []string{"https://foo.bar/charts/" + name + "-0.0.2.tgz"},
		},
		{
			Metadata: &chart.Metadata{
				Name:       name,
				Version:    "0.0.3",
				APIVersion: "v2",
			},
			URLs: []string{"https://foo.bar/charts/" + name + "-0.0.3.tgz"},
		},
		{
			Metadata: &chart.Metadata{
				Name:       name,
				Version:    "0.0.4",
				APIVersion: "v2",
				Dependencies: []*chart.Dependency{
					{
						Name:       "testing-dep",
						Version:    "0.1.x",
						Repository: "https://foo.bar/charts/",
					},
				},
			},
			URLs: []string{"https://foo.bar/charts/" + name + "-0.0.4.tgz"},
		},
		{
			Metadata: &chart.Metadata{
				Name:       name,
				Version:    "0.0.5",
				APIVersion: "v2",
				Dependencies: []*chart.Dependency{
					{
						Name:       "testing-dep",
						Version:    "0.1.x",
						Repository: "https://bar.foo/charts/",
					},
				},
			},
			URLs: []string{"https://bar.foo/charts/" + name + "-0.0.5.tgz"},
		},
	}
}
