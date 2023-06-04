package utils

import (
	"net/http"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LeaderElection struct {
	Enabled    bool   `yaml:"enabled"`
	ResourceID string `yaml:"resourceName"`
}

type Config struct {
	HealthProbeBindAddress string         `yaml:"healthProbeBindAddress"`
	MetricsBindAddress     string         `yaml:"metricsBindAddress"`
	LeaderElection         LeaderElection `yaml:"leaderElection"`
	WebhookPort            int            `yaml:"webhookPort"`
}

type HelmRESTClientGetter struct {
	Namespace        string
	ReleaseNamespace string
	KubeConfig       string
	IsLocal          bool
	HelmConfig       *helmv1alpha1.Config
	Client           client.Client
	logger           logr.Logger
}

// ClientInterface repesents interface for mocking custom k8s client
type ClientInterface interface {
	GetResource(name, namespace, resource, group, version string, opts metav1.GetOptions) ([]byte, error)
	ListResources(namespace, resource, group, version string, opts metav1.ListOptions) ([]byte, error)
}

// HTTPClientInterface represents interface for mocking an http client
type HTTPClientInterface interface {
	Get(url string) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}
