package helm

import (
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

type HelmClient struct {
	Repos    *HelmRepos
	Releases *HelmReleases
	Env      map[string]string
}

type HelmReleases struct {
	Entries     []*HelmRelease
	Conditional ResourceConditional
	Config      *action.Configuration
	Settings    *cli.EnvSettings
}

type HelmRelease struct {
	Name           string
	Repo           string
	Chart          string
	Version        string
	ValuesTemplate *HelmValueTemplate
	Values         map[string]interface{}
	Namespace      Namespace
	Conditional    ResourceConditional
	Config         *action.Configuration
	Settings       *cli.EnvSettings
}

type HelmRepos struct {
	Entries   []*HelmRepo
	Settings  *cli.EnvSettings
	installed *repo.File
}

type HelmRepo struct {
	Name     string
	Url      string
	Auth     HelmAuth
	Settings *cli.EnvSettings
}

type HelmCharts struct {
	Versions []HelmChart
}

type HelmChart struct {
	Version   *repo.ChartVersion
	Templates []*chart.File
	CRDs      []*chart.File
}

type HelmValueTemplate struct {
	valuesRef  []*ValuesRef
	Values     map[string]interface{}
	ValuesMap  map[string]string
	ValueFiles []string
}

type ValuesRef struct {
	Ref    *helmv1alpha1.Values `json:"Ref" filter:"ref"`
	Parent string               `json:"Parent" filter:"parent"`
}

type HelmAuth struct {
	User     string
	Password string
	Cert     string
	Key      string
	Ca       string
}

type Namespace struct {
	Name    string
	Install bool
}

type ResourceConditional struct {
}

type ListOptions struct {
	filter map[string]string
}
