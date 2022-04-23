package helm

import (
	"github.com/soer3n/yaho/internal/release"
	"helm.sh/helm/v3/pkg/chart"
)

// GetTestClientRelease returns release cr for testing client
func GetTestClientRelease() *release.Release {
	return &release.Release{
		Name: "far",
		Repo: "boo",
		Chart: &chart.Chart{
			Metadata: &chart.Metadata{
				Name: "foo",
			},
		},
		Version: "0.0.1",
	}
}
