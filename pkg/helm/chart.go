package helm

import (
	"github.com/prometheus/common/log"
	"helm.sh/helm/v3/pkg/action"
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
	repo := chart.Repo
	client := chart.Client
	settings := chart.Settings

	for _, chart := range chart.Versions {
		argsList = make([]string, 0)
		argsList = append(argsList, chart.Version.Metadata.Name)
		argsList = append(argsList, repo+"/"+chart.Version.Metadata.Name)
		name, chart, err := client.NameAndChart(argsList)

		if err != nil {
			return err
		}

		client.ReleaseName = name
		cp, err := client.ChartPathOptions.LocateChart(chart, settings)

		if err != nil {
			return err
		}

		chartRequested, err := loader.Load(cp)

		if err != nil {
			return err
		}

		log.Infof("Templates: %v", chartRequested.Templates)
		log.Infof("CRDs: %v", chartRequested.CRDs())
		log.Infof("Raw: %v", chartRequested.Raw)

	}

	return nil
}

func (chart *HelmChart) createConfigMaps() {}

func (chart *HelmChart) createTemplateConfigMap() {}

func (chart *HelmChart) createCRDConfigMap() {}
