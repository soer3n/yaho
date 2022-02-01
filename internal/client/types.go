package client

import (
	"github.com/soer3n/yaho/internal/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Client represents struct needed for handling subclients
type Client struct {
	DynamicClient  dynamic.Interface
	DiscoverClient discovery.ServerResourcesInterface
	TypedClient    kubernetes.Interface
	Opts
	types.ClientInterface
}

// ResourceKind represents a kind in a k8s api group
type ResourceKind struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
}

// Opts represents an interface for collecting options in a generic way
type Opts interface{}
