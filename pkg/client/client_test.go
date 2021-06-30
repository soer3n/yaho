package client

import (
	"testing"

	"github.com/soer3n/apps-operator/internal/mocks"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetAPIResources(t *testing.T) {

	dcMock := &mocks.K8SDiscoveryMock{}
	dcMock.On("GetAPIResources", "apiGroup", false).Return([]*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "apiresource",
					Group:      "apiGroup",
					Namespaced: false,
				},
			},
		},
	}, nil)

	client := New()
	client.DiscoverClient = dcMock

	assert := assert.New(t)

	_, err := client.GetAPIResources("apiGroup", false)

	assert.Nil(err)
}

func TestListResources(t *testing.T) {

	dcMock := &mocks.K8SDynamicClientMock{}

	gvr := schema.GroupVersionResource{
		Group:    "group",
		Version:  "v1",
		Resource: "resource",
	}
	namespaceableMock := mocks.K8SNamespaceMock{}
	dcMock.On("Resource", gvr).Return(&namespaceableMock)

	resourceMock := mocks.K8SResourceMock{}
	namespaceableMock.On("Namespace", "namespace").Return(&resourceMock)

	usObj := &unstructured.UnstructuredList{}
	resourceMock.On("List", metav1.ListOptions{}).Return(usObj, nil)

	client := New()
	client.DynamicClient = dcMock

	assert := assert.New(t)

	_, err := client.ListResources("namespace", "resource", "group", "v1", metav1.ListOptions{})

	assert.Nil(err)
}

func TestGetResource(t *testing.T) {

	dcMock := &mocks.K8SDynamicClientMock{}

	gvr := schema.GroupVersionResource{
		Group:    "group",
		Version:  "v1",
		Resource: "resource",
	}
	namespaceableMock := mocks.K8SNamespaceMock{}
	dcMock.On("Resource", gvr).Return(&namespaceableMock)

	resourceMock := mocks.K8SResourceMock{}
	namespaceableMock.On("Namespace", "namespace").Return(&resourceMock)

	usObj := &unstructured.Unstructured{}
	resourceMock.On("Get", "name").Return(usObj, nil)

	client := New()
	client.DynamicClient = dcMock

	assert := assert.New(t)

	_, err := client.GetResource("name", "namespace", "resource", "group", "v1", metav1.GetOptions{})

	assert.Nil(err)
}
