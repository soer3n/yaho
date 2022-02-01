package helm

import (
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetTestRepoChartListSpec returns expected chartlist spec for testing
func GetTestRepoChartListSpec() *helmv1alpha1.ChartList {

	chartSpec := helmv1alpha1.ChartSpec{
		Name:        "chart.Name",
		Home:        "chart.Spec.Home",
		Sources:     []string{"chart.Spec.Sources"},
		Description: "chart.Spec.Description",
		Keywords:    []string{"chart.Spec.Keywords"},
		Versions: []helmv1alpha1.ChartVersion{
			{
				Name: "0.1.0",
				URL:  "https://foo.bar/repo/foo.tar.gz",
			},
		},
		Maintainers: []*chart.Maintainer{
			{
				Name: "chart.Spec.Maintainers",
			},
		},
		Icon:        "chart.Spec.Icon",
		APIVersion:  "chart.Spec.APIVersion",
		Condition:   "chart.Spec.Condition",
		Tags:        "chart.Spec.Tags",
		AppVersion:  "chart.Spec.AppVersion",
		Deprecated:  false,
		Annotations: map[string]string{},
		KubeVersion: "chart.Spec.KubeVersion",
		Type:        "chart.Spec.Type",
	}

	return &helmv1alpha1.ChartList{
		Items: []helmv1alpha1.Chart{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: chartSpec,
			},
		},
	}
}

// GetTestRepoSpecs returns testcases for testing chart cr
func GetTestRepoSpecs() []inttypes.TestCase {
	return []inttypes.TestCase{

		{
			Input: helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "selector",
					},
				},
				Spec: helmv1alpha1.RepoSpec{
					Name:       "test",
					URL:        "https://bar.foo/charts",
					AuthSecret: "secret",
				},
			},
			ReturnValue: "",
			ReturnError: nil,
		},
	}
}

// GetTestRepoIndexFile returns repoIndex for testing chart cr
func GetTestRepoIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}
