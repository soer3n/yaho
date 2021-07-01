package api

import (
	"net/http"

	"github.com/soer3n/apps-operator/pkg/client"
)

type Api struct {
	ListenAddress string
	Server        *http.Server
	Routes        *http.ServeMux
}

type Handler struct {
	ApiVersion string
	K8SClient  *client.Client
}

type APIResponse struct {
	Message string
	Data    map[string]interface{}
}
