package helm

import (
	"helm.sh/helm/v3/pkg/repo"
)

func NewChart(versions []*repo.ChartVersion) *HelmChart {

	var chartVersions []*HelmChartVersion

	for _, version := range versions {
		item := &HelmChartVersion{
			Version: version,
		}

		chartVersions = append(chartVersions, item)
	}

	return &HelmChart{
		Versions: chartVersions,
	}
}

func (chart *HelmChart) createConfigMaps() {}

func (chart *HelmChart) createTemplateConfigMap() {}

func (chart *HelmChart) createCRDConfigMap() {}
