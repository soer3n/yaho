package helm

import (
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/types"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewChart represents initialization of internal chart struct
func NewChart(versions []*repo.ChartVersion, settings *cli.EnvSettings, repo string, k8sclient client.Client, g types.HTTPClientInterface) *HelmChart {

	var chartVersions []HelmChartVersion
	var config *action.Configuration
	var err error

	for _, version := range versions {
		item := HelmChartVersion{
			Version: version,
		}

		chartVersions = append(chartVersions, item)
	}

	if config, err = initActionConfig(settings); err != nil {
		log.Infof("Error on getting action config for chart %v: %v", chartVersions[0].Version.Metadata.Name, err)
		return &HelmChart{}
	}

	return &HelmChart{
		Versions:  chartVersions,
		Client:    action.NewInstall(config),
		Settings:  settings,
		Repo:      repo,
		k8sClient: k8sclient,
		getter:    g,
	}
}

// CreateTemplates represents func to parse compress chart data into configmaps
func (chart *HelmChart) CreateTemplates() error {

	var name, chartURL string
	var chartRequested *helmchart.Chart
	var err error

	client := chart.Client
	k8sClient := chart.k8sClient
	g := chart.getter

	for _, chart := range chart.Versions {

		client.ReleaseName = name
		client.Version = chart.Version.Version

		if chartURL, err = getChartURL(k8sClient, chart.Version.Name, chart.Version.Version, client.Namespace); err != nil {
			return err
		}

		if chartRequested, err = getChartByURL(chartURL, g); err != nil {
			return err
		}

		log.Debugf("Templates: %v", chartRequested.Templates)
		chart.Templates = chartRequested.Templates

		log.Debugf("CRDs: %v", chartRequested.CRDs())
		chart.CRDs = chartRequested.CRDs()

		log.Debugf("Default Values: %v", chartRequested.Values)
		chart.DefaultValues = chartRequested.Values
	}

	return nil
}

// AddOrUpdateChartMap represents update of a map of chart structs if needed
func (chart *HelmChart) AddOrUpdateChartMap(chartObjMap map[string]*helmv1alpha1.Chart, instance *helmv1alpha1.Repo) map[string]*helmv1alpha1.Chart {

	for _, version := range chart.Versions {
		if chartObjMap, err := version.AddOrUpdateChartMap(chartObjMap, instance); err != nil {
			return chartObjMap
		}
	}

	return chartObjMap
}

// CreateConfigMaps represents the creation of needed configmaps related to a chart
func (chart HelmChart) CreateConfigMaps() []v1.ConfigMap {

	returnList := []v1.ConfigMap{}

	for _, version := range chart.Versions {
		versionConfigMaps := version.createConfigMaps(chart.Settings.Namespace())

		for _, configmap := range versionConfigMaps {
			returnList = append(returnList, configmap)
		}
	}

	return returnList
}
