package api

import "net/http"

type Api struct {
	ListenAddress string
	Server        *http.Server
	Routes        *http.ServeMux
}

type Handler struct {
	ApiVersion string
}

type APIResponse struct {
	Message string
	Data    []map[string]interface{}
}
