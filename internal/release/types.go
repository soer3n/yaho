package release

import (
	"sync"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Release represents data needed for installing and updating a helm release
type Release struct {
	Name             string
	Repo             string
	Chart            *helmchart.Chart
	Version          string
	ValuesTemplate   *values.ValueTemplate
	Namespace        Namespace
	releaseNamespace string
	Flags            *helmv1alpha1.Flags
	Config           *action.Configuration
	Settings         *cli.EnvSettings
	Client           *action.Install
	K8sClient        client.Client
	getter           utils.HTTPClientInterface
	logger           logr.Logger
	wg               *sync.WaitGroup
	mu               sync.Mutex
}

type spec struct {
	Name             string
	Repo             string
	Version          string
	releaseNamespace string
}

type helm struct {
	client   *action.Install
	flags    *helmv1alpha1.Flags
	settings *cli.EnvSettings
	config   *action.Configuration
}

type kubernetes struct {
	client client.Client
	logger logr.Logger
}

// Namespace represents struct with release namespace name and if it should be installed
type Namespace struct {
	Name    string
	Install bool
}
