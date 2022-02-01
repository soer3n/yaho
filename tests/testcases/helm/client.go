package helm

import (
	"github.com/soer3n/yaho/internal/helm"
	"helm.sh/helm/v3/pkg/repo"
)

func GetTestClientRepo() *helm.Repo {
	return &helm.Repo{
		Name: "boo",
		URL:  "https://unknown.domain/charts",
	}
}

func GetTestClientRelease() *helm.Release {
	return &helm.Release{
		Name:    "far",
		Repo:    "boo",
		Chart:   "foo",
		Version: "0.0.1",
	}
}

func GetTestClientIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}
