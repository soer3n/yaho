package hub

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type BackendInterface interface {
	IsActive() bool
	GetChannel() chan []byte
	GetName() string
	GetDefaults() Defaults
	GetConfig() []byte
	GetScheme() *runtime.Scheme
	Update(Defaults, *v1.Secret, *runtime.Scheme) error
	Start(context.Context, time.Duration)
	Stop() error
}

type Hub struct {
	Backends map[string]BackendInterface
}

type Cluster struct {
	name           string
	agent          clusterAgent
	WatchNamespace string
	defaults       Defaults
	remoteClient   client.WithWatch
	localClient    client.WithWatch
	config         []byte
	channel        chan []byte
	scheme         *runtime.Scheme
	logger         logr.Logger
	cancelFunc     context.CancelFunc
}

type clusterAgent struct {
	Name      string
	Namespace string
	Deploy    bool
}

type Defaults struct {
	Charts   []Chart
	Releases []string
}

type Chart struct {
	Name    string
	Version string
}
