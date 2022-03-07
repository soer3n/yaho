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
		Name:       "chart.Name",
		Versions:   []string{"0.1.0"},
		Repository: "foo",
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
			Input: helmv1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "selector",
					},
				},
				Spec: helmv1alpha1.RepositorySpec{
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
			"doo": []*repo.ChartVersion{
				{
					Metadata: &chart.Metadata{
						Name:    "doo",
						Version: "0.0.1",
					},
					URLs: []string{"nodomain.com"},
				},
			},
		},
	}
}
