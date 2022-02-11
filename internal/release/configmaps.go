package release

import (
	"context"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// GetParsedConfigMaps represents parsing and returning of chart related data for a release
func (hc *Release) GetParsedConfigMaps(namespace string) ([]v1.ConfigMap, []helmv1alpha1.Chart) {
	var chartRequested *helmchart.Chart
	var repoObj helmv1alpha1.Repo
	var chartObj helmv1alpha1.Chart
	chartObjList := &helmv1alpha1.ChartList{}
	chartObjList.Items = []helmv1alpha1.Chart{}
	var chartURL string
	var specValues map[string]interface{}
	var err error

	configmapList := []v1.ConfigMap{}
	installConfig := hc.Config
	releaseClient := action.NewInstall(installConfig)
	releaseClient.ReleaseName = hc.Name
	hc.Client = releaseClient
	chartVersion := &chart.ChartVersion{}

	if repoObj, err = hc.getRepo(); err != nil {
		return configmapList, chartObjList.Items
	}

	options := &action.ChartPathOptions{}
	options.RepoURL = hc.Repo
	options.Version = hc.Version

	if specValues, err = hc.getValues(); err != nil {
		return configmapList, chartObjList.Items
	}

	if chartRequested, err = hc.getChart(hc.Chart, options, specValues); err != nil {

		if chartURL, err = chart.GetChartURL(hc.K8sClient, hc.Chart, hc.Version, hc.Namespace.Name); err != nil {
			return configmapList, chartObjList.Items
		}

		releaseClient.ReleaseName = hc.Name
		releaseClient.Version = hc.Version
		releaseClient.ChartPathOptions.RepoURL = repoObj.Spec.URL
		credentials := &chart.Auth{}

		if repoObj.Spec.AuthSecret != "" {
			credentials = hc.getCredentials(repoObj.Spec.AuthSecret)
		}

		if chartRequested, err = chart.GetChartByURL(chartURL, credentials, hc.getter); err != nil {
			return configmapList, chartObjList.Items
		}
	}

	if err = hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      hc.Chart,
	}, &chartObj); err != nil {
		return configmapList, chartObjList.Items
	}

	chartVersion.Version = &repo.ChartVersion{
		Metadata: &helmchart.Metadata{
			Name:    hc.Chart,
			Version: hc.Version,
		},
	}

	chartVersion.Templates = chartRequested.Templates
	chartVersion.CRDs = chartRequested.CRDs()
	chartVersion.DefaultValues = chartRequested.Values
	deps := chartRequested.Dependencies()
	version := utils.GetChartVersion(hc.Version, &chartObj)

	for _, v := range version.Dependencies {
		if err := hc.validateChartSpec(deps, v, chartObjList); err != nil {
			return configmapList, chartObjList.Items
		}
	}

	chartVersion.Version.Metadata.Version = version.Name
	configmapList = chartVersion.CreateConfigMaps(hc.Namespace.Name, deps)
	// chartObjList = append(chartObjList, &chartObj)

	return configmapList, chartObjList.Items
}
