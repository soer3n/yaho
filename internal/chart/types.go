package chart

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/chartversion"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Chart represents struct for data needed for managing chart
type Chart struct {
	Name       string
	Namespace  string
	Deprecated *bool
	Type       *string
	Tags       *string
	Versions   ChartVersions
	Client     *action.Install
	Settings   *cli.EnvSettings
	index      repo.ChartVersions
	helmConfig *action.Configuration
	Repo       string
	K8sClient  client.WithWatch
	getter     utils.HTTPClientInterface
	logger     logr.Logger
	mu         *sync.Mutex
	URL        string
}

// ChartVersions represents a list of internal struct for a chart version
type ChartVersions []*chartversion.ChartVersion
