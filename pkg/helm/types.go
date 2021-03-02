package helm

import (
	"helm.sh/helm/v3/pkg/action"
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
	values         map[string]interface{}
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

type HelmValueTemplate struct {
	Values     map[string]string
	ValueFiles []string
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
