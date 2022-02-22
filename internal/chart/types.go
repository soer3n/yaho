package chart

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Chart represents struct for data needed for managing chart
type Chart struct {
	Name      string
	Versions  ChartVersions
	Client    *action.Install
	Settings  *cli.EnvSettings
	Repo      string
	K8sClient client.Client
	getter    utils.HTTPClientInterface
	logger    logr.Logger
	mu        *sync.Mutex
	URL       string
}

// ChartVersions represents a list of internal struct for a chart version
type ChartVersions []*ChartVersion

// ChartVersion represents struct with needed data for returning needed data for managing a release
type ChartVersion struct {
	Version       *repo.ChartVersion
	Templates     []*chart.File
	CRDs          []*chart.File
	DefaultValues map[string]interface{}
	mu            *sync.Mutex
}

// Auth represents struct with auth data for a repo
type Auth struct {
	User     string
	Password string
	Cert     string
	Key      string
	Ca       string
}
