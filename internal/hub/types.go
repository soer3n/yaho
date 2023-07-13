package hub

import (
	"context"
	"time"

	"github.com/go-logr/logr"
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
	Update(Defaults, []byte, *runtime.Scheme) error
	Start(context.Context, time.Duration)
	Stop() error
}

type Hub struct {
	Backends map[string]BackendInterface
}

type Cluster struct {
	name           string
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

type Defaults struct {
	Charts   []Chart
	Releases []string
}

type Chart struct {
	Name    string
	Version string
}
