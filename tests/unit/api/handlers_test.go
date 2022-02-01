package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/soer3n/yaho/internal/api"
	"github.com/soer3n/yaho/internal/client"
	apimocks "github.com/soer3n/yaho/tests/mocks/api"
	"github.com/stretchr/testify/assert"
)

func TestK8sApiGroupResources(t *testing.T) {
	assert := assert.New(t)

	k8sclient := &client.Client{}
	k8sclient.DiscoverClient = apimocks.GetClientDiscoveryMock()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	handler := api.NewHandler("v1", k8sclient)
	handler.K8sAPIGroup(res, req)
	assert.NotNil(res)
}

func TestK8sApiGroup(t *testing.T) {
	assert := assert.New(t)

	k8sclient := &client.Client{}
	k8sclient.DynamicClient = apimocks.GetClientDynamicMock()

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/resources/", res.Body)

	handler := api.NewHandler("v1", k8sclient)
	handler.K8sAPIGroupResources(res, req)
	assert.NotNil(res)

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/resources/resource", res.Body)
	req = mux.SetURLVars(req, map[string]string{
		"resource": "resource",
	})

	handler = api.NewHandler("v1", k8sclient)
	handler.K8sAPIGroupResources(res, req)
	assert.NotNil(res)

	res = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/resources/resource/group", res.Body)
	req = mux.SetURLVars(req, map[string]string{
		"resource": "resource",
		"group":    "group",
	})

	handler = api.NewHandler("v1", k8sclient)
	handler.K8sAPIGroupResources(res, req)
	assert.NotNil(res)

	req = httptest.NewRequest(http.MethodGet, "/api/resources/resource/group/version", nil)
	res = httptest.NewRecorder()
	req = mux.SetURLVars(req, map[string]string{
		"resource": "resource",
		"group":    "group",
		"version":  "v1",
	})

	handler = api.NewHandler("v1", k8sclient)
	handler.K8sAPIGroupResources(res, req)
	assert.NotNil(res)
}
