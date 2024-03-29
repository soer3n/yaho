package unstructured

import (
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8SClientMock represents mock struct for k8s runtime client
type K8SClientMock struct {
	mock.Mock
	client.WithWatch
}

// K8SResourceMock represents struct for mocking a unstructured resource
type K8SResourceMock struct {
	mock.Mock
	dynamic.ResourceInterface
}

// K8SDiscoveryMock represents struct for mocking a runtime discovery client
type K8SDiscoveryMock struct {
	mock.Mock
	discovery.ServerResourcesInterface
}

// K8SDynamicClientMock represents struct for mocking a dynamic resource client in k8s
type K8SDynamicClientMock struct {
	mock.Mock
	dynamic.Interface
}

// K8SNamespaceMock represents struct for mocking a namespaced unstructured resource
type K8SNamespaceMock struct {
	mock.Mock
	dynamic.NamespaceableResourceInterface
}
