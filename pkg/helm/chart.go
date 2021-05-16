package helm

import (
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

func NewChart(versions []*repo.ChartVersion, settings *cli.EnvSettings, repo string) *HelmChart {

	var chartVersions []*HelmChartVersion

	for _, version := range versions {
		item := &HelmChartVersion{
			Version: version,
		}

		chartVersions = append(chartVersions, item)
	}

	config, _ := initActionConfig(settings)

	return &HelmChart{
		Versions: chartVersions,
		Client:   action.NewInstall(config),
		Settings: settings,
		Repo:     repo,
	}
}

func (chart *HelmChart) CreateTemplates() error {
	var argsList []string
	var name, chartname, cp string
	var chartRequested *helmchart.Chart
	var err error

	repo := chart.Repo
	client := chart.Client
	settings := chart.Settings

	for _, chart := range chart.Versions {
		argsList = make([]string, 0)
		argsList = append(argsList, chart.Version.Metadata.Name)
		argsList = append(argsList, repo+"/"+chart.Version.Metadata.Name)

		if name, chartname, err = client.NameAndChart(argsList); err != nil {
			return err
		}

		client.ReleaseName = name
		if cp, err = client.ChartPathOptions.LocateChart(chartname, settings); err != nil {
			return err
		}

		if chartRequested, err = loader.Load(cp); err != nil {
			return err
		}

		log.Infof("Templates: %v", chartRequested.Templates)
		log.Infof("CRDs: %v", chartRequested.CRDs())
		log.Infof("Raw: %v", chartRequested.Raw)

	}

	return nil
}

func (chart *HelmChart) AddOrUpdateChartMap(chartObjMap map[string]*helmv1alpha1.Chart, instance *helmv1alpha1.Repo) (map[string]*helmv1alpha1.Chart, error) {

	for _, version := range chart.Versions {
		if chartObjMap, err := version.AddOrUpdateChartMap(chartObjMap, instance); err != nil {
			return chartObjMap, err
		}
	}

	return chartObjMap, nil
}

func (chart *HelmChart) createConfigMaps() {}
