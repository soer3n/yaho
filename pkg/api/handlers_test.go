package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soer3n/apps-operator/internal/mocks"
	"github.com/soer3n/apps-operator/pkg/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestK8sApiGroup(t *testing.T) {

	assert := assert.New(t)

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

	k8sclient := client.New()
	k8sclient.DynamicClient = dcMock

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", res.Body)

	handler := NewHandler("v1", k8sclient)
	handler.K8sApiGroupResources(res, req)
	assert.NotNil("gfo")
}

func TestK8sApiGroupResources(t *testing.T) {

	assert := assert.New(t)
	assert.NotNil("gfo")
}
