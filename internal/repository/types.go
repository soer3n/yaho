package repository

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	getter     utils.HTTPClientInterface
	helmClient kube.Client
	logger     logr.Logger
	wg         *sync.WaitGroup
	mu         sync.Mutex
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
