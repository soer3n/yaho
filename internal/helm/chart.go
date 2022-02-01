package helm

import (
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewChart represents initialization of internal chart struct
func NewChart(versions []*repo.ChartVersion, settings *cli.EnvSettings, repo string, k8sclient client.Client, g types.HTTPClientInterface, c kube.Client) *Chart {
	var chartVersions []ChartVersion
	var config *action.Configuration
	var err error

	for _, version := range versions {
		item := ChartVersion{
			Version: version,
		}

		chartVersions = append(chartVersions, item)
	}

	if config, err = initActionConfig(settings, c); err != nil {
		log.Infof("Error on getting action config for chart %v: %v", chartVersions[0].Version.Metadata.Name, err)
		return &Chart{}
	}

	return &Chart{
		Versions:  chartVersions,
		Client:    action.NewInstall(config),
		Settings:  settings,
		Repo:      repo,
		K8sClient: k8sclient,
		getter:    g,
	}
}

// AddOrUpdateChartMap represents update of a map of chart structs if needed
func (chart *Chart) AddOrUpdateChartMap(chartObjMap map[string]*helmv1alpha1.Chart, instance *helmv1alpha1.Repo) map[string]*helmv1alpha1.Chart {
	for _, version := range chart.Versions {
		if chartObjMap, err := version.AddOrUpdateChartMap(chartObjMap, instance); err != nil {
			return chartObjMap
		}
	}

	return chartObjMap
}

// CreateConfigMaps represents the creation of needed configmaps related to a chart
func (chart Chart) CreateConfigMaps() []v1.ConfigMap {
	returnList := []v1.ConfigMap{}

	for _, version := range chart.Versions {
		versionConfigMaps := version.createConfigMaps(chart.Settings.Namespace(), nil)
		returnList = append(returnList, versionConfigMaps...)
	}

	return returnList
}
