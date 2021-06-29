package client

import (
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

type Client struct {
	DynamicClient  dynamic.Interface
	DiscoverClient discovery.CachedDiscoveryInterface
	ClientOpts
	ClientInterface
}

type ResourceKind struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
}

type ClientOpts interface {
}

type K8SClientMock struct {
	mock.Mock
	dynamic.Interface
}

type K8SNamespaceMock struct {
	mock.Mock
	dynamic.NamespaceableResourceInterface
}

type K8SResourceMock struct {
	mock.Mock
	dynamic.ResourceInterface
}
