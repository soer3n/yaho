package chart

import (
	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New represents initialization of internal chart struct
func New(versions []*repo.ChartVersion, settings *cli.EnvSettings, logger logr.Logger, repo string, k8sclient client.Client, g utils.HTTPClientInterface, c kube.Client) *Chart {
	var chartVersions []ChartVersion
	var config *action.Configuration
	var err error

	for _, version := range versions {
		item := ChartVersion{
			Version: version,
		}

		chartVersions = append(chartVersions, item)
	}

	if config, err = utils.InitActionConfig(settings, c); err != nil {
		logger.Info("Error on getting action config for chart")
		return &Chart{}
	}

	return &Chart{
		Versions:  chartVersions,
		Client:    action.NewInstall(config),
		Settings:  settings,
		Repo:      repo,
		K8sClient: k8sclient,
		getter:    g,
		logger:    logger.WithValues("repo", repo),
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
