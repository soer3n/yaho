package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/soer3n/apps-operator/internal/mocks"
	"github.com/soer3n/apps-operator/pkg/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestK8sApiGroupResources(t *testing.T) {

	assert := assert.New(t)

	dcMock := &mocks.K8SDiscoveryMock{}
	dcMock.On("ServerPreferredResources").Return([]*metav1.APIResourceList{
		{
			GroupVersion: "apiGroup/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "apiresource",
					Group:      "apiGroup",
					Namespaced: false,
					Verbs: metav1.Verbs{
						"foo", "bar",
					},
				},
			},
		},
	}, nil)

	k8sclient := client.New()
	k8sclient.DiscoverClient = dcMock

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	handler := NewHandler("v1", k8sclient)
	handler.K8sApiGroup(res, req)
	assert.NotNil(res)

}

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
	namespaceableMock.On("Namespace", "").Return(&resourceMock)

	objList := []unstructured.Unstructured{
		{
			Object: map[string]interface{}{"foo": "bar"},
		},
		{
			Object: map[string]interface{}{"bar": "foo"},
		},
	}
	usObj := &unstructured.UnstructuredList{
		Items: objList,
	}
	resourceMock.On("List", metav1.ListOptions{}).Return(usObj, nil)

	k8sclient := client.New()
	k8sclient.DynamicClient = dcMock

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/resources/", res.Body)

	handler := NewHandler("v1", k8sclient)
	handler.K8sApiGroupResources(res, req)
	assert.NotNil(res)

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/resources/resource", res.Body)
	req = mux.SetURLVars(req, map[string]string{
		"resource": "resource",
	})

	handler = NewHandler("v1", k8sclient)
	handler.K8sApiGroupResources(res, req)
	assert.NotNil(res)

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/resources/resource/group", res.Body)
	req = mux.SetURLVars(req, map[string]string{
		"resource": "resource",
		"group":    "group",
	})

	handler = NewHandler("v1", k8sclient)
	handler.K8sApiGroupResources(res, req)
	assert.NotNil(res)

	req = httptest.NewRequest(http.MethodGet, "/api/resources/resource/group/version", nil)
	res = httptest.NewRecorder()
	req = mux.SetURLVars(req, map[string]string{
		"resource": "resource",
		"group":    "group",
		"version":  "v1",
	})

	handler = NewHandler("v1", k8sclient)
	handler.K8sApiGroupResources(res, req)
	assert.NotNil(res)
}
