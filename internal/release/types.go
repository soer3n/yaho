package release

import (
	"sync"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	ValuesTemplate *values.ValueTemplate
	Values         map[string]interface{}
	Namespace      Namespace
	Flags          *helmv1alpha1.Flags
	Config         *action.Configuration
	Settings       *cli.EnvSettings
	Client         *action.Install
	K8sClient      client.Client
	getter         utils.HTTPClientInterface
	logger         logr.Logger
	wg             *sync.WaitGroup
	mu             *sync.Mutex
}

// Namespace represents struct with release namespace name and if it should be installed
type Namespace struct {
	Name    string
	Install bool
}
