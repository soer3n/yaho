package chart

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	helmrepo "helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New represents initialization of internal chart struct
func New(name, repoURL string, versions []*helmrepo.ChartVersion, settings *cli.EnvSettings, logger logr.Logger, repo string, k8sclient client.Client, g utils.HTTPClientInterface, c kube.Client) *Chart {
	var chartVersions []*ChartVersion
	var config *action.Configuration
	var err error

	for _, version := range versions {
		item := &ChartVersion{
			Version: version,
			mu:      &sync.Mutex{},
		}

		chartVersions = append(chartVersions, item)
	}

	if config, err = utils.InitActionConfig(settings, c); err != nil {
		logger.Info("Error on getting action config for chart")
		return &Chart{}
	}

	chartURL, err := helmrepo.ResolveReferenceURL(repoURL, versions[0].URLs[0])

	if err != nil {
		return &Chart{}
	}

	return &Chart{
		Name:      name,
		Versions:  chartVersions,
		Client:    action.NewInstall(config),
		Settings:  settings,
		Repo:      repo,
		K8sClient: k8sclient,
		URL:       chartURL,
		getter:    g,
		logger:    logger.WithValues("repo", repo),
		mu:        &sync.Mutex{},
	}
}
