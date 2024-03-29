package helm

import (
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetTestRepoRepoListSpec returns expected chartlist spec for testing
func GetTestRepoRepoListSpec() helmv1alpha1.RepositoryList {
	repoSpec := helmv1alpha1.RepositorySpec{
		Name: "chart.Name",
		URL:  "https://dep.bar/charts",
	}

	return helmv1alpha1.RepositoryList{
		Items: []helmv1alpha1.Repository{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: repoSpec,
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
					Name:      "one",
					Namespace: "one",
					Labels: map[string]string{
						"repo": "one",
					},
				},
				Spec: helmv1alpha1.RepositorySpec{
					Name:       "one",
					URL:        "https://foo.bar/charts",
					AuthSecret: "secret",
					Charts: []helmv1alpha1.Entry{
						{
							Name:     "foo",
							Versions: []string{"0.0.1"},
						},
					},
				},
			},
			ReturnValue: "",
			ReturnError: map[string]error{
				"init":   nil,
				"update": nil,
			},
		},
		{
			Input: helmv1alpha1.Repository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "two",
					Namespace: "two",
					Labels: map[string]string{
						"repo": "two",
					},
				},
				Spec: helmv1alpha1.RepositorySpec{
					Name: "two",
					URL:  "https://bar.foo/charts",
					Charts: []helmv1alpha1.Entry{
						{
							Name: "bar",
						},
					},
				},
			},
			ReturnValue: "",
			ReturnError: map[string]error{
				"init":   nil,
				"update": nil,
			},
		},
	}
}

// GetTestRepoIndexFile returns repoIndex for testing chart cr
func GetTestRepoIndexFile(name string) *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			name: []*repo.ChartVersion{
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
			},
		},
	}
}
