package helm

import (
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client represents base struct of this package
type Client struct {
	Repos    *Repos
	Releases *Releases
	Env      map[string]string
	Client   kube.Client
}

// Releases represents struct for data needed for managing releases and list of installed
type Releases struct {
	Entries  []*Release
	Config   *action.Configuration
	Settings *cli.EnvSettings
}

// Release represents data needed for installing and updating a helm release
type Release struct {
	Name           string
	Repo           string
	Chart          string
	Version        string
	ValuesTemplate *ValueTemplate
	Values         map[string]interface{}
	Namespace      Namespace
	Flags          *helmv1alpha1.Flags
	Config         *action.Configuration
	Settings       *cli.EnvSettings
	Client         *action.Install
	K8sClient      client.Client
	getter         types.HTTPClientInterface
}

// Repos represents struct for data needed for managing repos and list of installed
type Repos struct {
	Entries  []*Repo
	Settings *cli.EnvSettings
}

// Repo represents struct for data needed for managing repos and list of installed
type Repo struct {
	Name       string
	URL        string
	Auth       *Auth
	Namespace  Namespace
	Settings   *cli.EnvSettings
	K8sClient  client.Client
	getter     types.HTTPClientInterface
	helmClient kube.Client
}

// Chart represents struct for data needed for managing chart
type Chart struct {
	Versions  ChartVersions
	Client    *action.Install
	Settings  *cli.EnvSettings
	Repo      string
	K8sClient client.Client
	getter    types.HTTPClientInterface
}

// ChartVersions represents a list of internal struct for a chart version
type ChartVersions []ChartVersion

// ChartVersion represents struct with needed data for returning needed data for managing a release
type ChartVersion struct {
	Version       *repo.ChartVersion
	Templates     []*chart.File
	CRDs          []*chart.File
	DefaultValues map[string]interface{}
}

// ValueTemplate represents struct for possible value inputs
type ValueTemplate struct {
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

// Auth represents struct with auth data for a repo
type Auth struct {
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
