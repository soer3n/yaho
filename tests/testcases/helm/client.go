package helm

import (
	"github.com/soer3n/yaho/internal/release"
	"helm.sh/helm/v3/pkg/repo"
)

// GetTestClientRelease returns release cr for testing client
func GetTestClientRelease() *release.Release {
	return &release.Release{
		Name:    "far",
		Repo:    "boo",
		Chart:   "foo",
		Version: "0.0.1",
	}
}

// GetTestClientIndexFile returns helm index file struct for testing client
func GetTestClientIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}
