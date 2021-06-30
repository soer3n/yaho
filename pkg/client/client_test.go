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

	dcMock := &mocks.K8SDynamicClientMock{}
	client := New()
	client.DynamicClient = dcMock

	//assert := assert.New(t)

	//_, _ = client.GetResource("", "", "", "", "", v1.GetOptions{})

	//assert.NotNil(nil)
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
