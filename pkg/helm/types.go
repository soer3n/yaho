package helm

import (
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmClient represents base struct of this package
type HelmClient struct {
	Repos    *HelmRepos
	Releases *HelmReleases
	Env      map[string]string
}

// HelmReleases represents struct for data needed for managing releases and list of installed
type HelmReleases struct {
	Entries  []*HelmRelease
	Config   *action.Configuration
	Settings *cli.EnvSettings
}

// HelmRelease represents data needed for installing and updating a helm release
type HelmRelease struct {
	Name           string
	Repo           string
	Chart          string
	Version        string
	ValuesTemplate *HelmValueTemplate
	Values         map[string]interface{}
	Namespace      Namespace
	Config         *action.Configuration
	Settings       *cli.EnvSettings
	Client         *action.Install
	k8sClient      client.Client
	getter         types.HTTPClientInterface
}

// HelmRepos represents struct for data needed for managing repos and list of installed
type HelmRepos struct {
	Entries   []*HelmRepo
	Settings  *cli.EnvSettings
	installed *repo.File
}

// HelmRepo represents struct for data needed for managing repos and list of installed
type HelmRepo struct {
	Name      string
	Url       string
	Auth      *HelmAuth
	Namespace Namespace
	Settings  *cli.EnvSettings
	k8sClient client.Client
	getter    types.HTTPClientInterface
}

// HelmChart represents struct for data needed for managing chart
type HelmChart struct {
	Versions  HelmChartVersions
	Client    *action.Install
	Settings  *cli.EnvSettings
	Repo      string
	k8sClient client.Client
	getter    types.HTTPClientInterface
}

// HelmChartVersions represents a list of internal struct for a chart version
type HelmChartVersions []HelmChartVersion

// HelmChartVersion represents struct with needed data for returning needed data for managing a release
type HelmChartVersion struct {
	Version       *repo.ChartVersion
	Templates     []*chart.File
	CRDs          []*chart.File
	DefaultValues map[string]interface{}
}

// HelmValueTemplate represents struct for possible value inputs
type HelmValueTemplate struct {
	valuesRef  []*ValuesRef
	Values     map[string]interface{}
	ValuesMap  map[string]string
	ValueFiles []string
}

// ValuesRef represents struct for filtering values kubernetes resources by json tag
type ValuesRef struct {
	Ref    *helmv1alpha1.Values `json:"Ref" filter:"ref"`
	Parent string               `json:"Parent" filter:"parent"`
}

// HelmAuth represents struct with auth data for a repo
type HelmAuth struct {
	User     string
	Password string
	Cert     string
	Key      string
	Ca       string
}

// Namespace represents struct with release namespace name and if it should be installed
type Namespace struct {
	Name    string
	Install bool
}

// ListOptions represents struct for filters for searching in values kubernetes resources
type ListOptions struct {
	filter map[string]string
}
