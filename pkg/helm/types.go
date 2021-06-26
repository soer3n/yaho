package helm

import (
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	client "github.com/soer3n/apps-operator/pkg/client"
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
	Client         *action.Install
	k8sClient      client.ClientInterface
	getter         client.HTTPClientInterface
}

type HelmRepos struct {
	Entries   []*HelmRepo
	Settings  *cli.EnvSettings
	installed *repo.File
}

type HelmRepo struct {
	Name      string
	Url       string
	Auth      *HelmAuth
	Namespace Namespace
	Settings  *cli.EnvSettings
	k8sClient client.ClientInterface
	getter    client.HTTPClientInterface
}

type HelmChart struct {
	Versions  HelmChartVersions
	Client    *action.Install
	Settings  *cli.EnvSettings
	Repo      string
	k8sClient client.ClientInterface
	getter    client.HTTPClientInterface
}

type HelmChartVersions []HelmChartVersion

type HelmChartVersion struct {
	Version       *repo.ChartVersion
	Templates     []*chart.File
	CRDs          []*chart.File
	DefaultValues map[string]interface{}
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
