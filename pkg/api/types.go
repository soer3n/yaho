package api

import (
	"net/http"

	"github.com/soer3n/yaho/pkg/client"
)

// API represents struct for handling an http server
type API struct {
	ListenAddress string
	Server        *http.Server
	Routes        *http.ServeMux
}

// Handler represents struct for managing sub handler and similar
type Handler struct {
	APIVersion string
	K8SClient  *client.Client
}

// Response represents struct for an http server json response
type Response struct {
	Message string
	Data    map[string]interface{}
}
