package mocks

import (
	clientutils "github.com/soer3n/apps-operator/pkg/client"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8SClientMock struct {
	mock.Mock
	client.Client
}

type HTTPClientMock struct {
	mock.Mock
	clientutils.HTTPClientInterface
}

type K8SResourceMock struct {
	mock.Mock
	dynamic.ResourceInterface
}

type K8SDiscoveryMock struct {
	mock.Mock
	discovery.ServerResourcesInterface
}

type K8SDynamicClientMock struct {
	mock.Mock
	dynamic.Interface
}

type K8SNamespaceMock struct {
	mock.Mock
	dynamic.NamespaceableResourceInterface
}
