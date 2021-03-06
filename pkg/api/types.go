package api

import "net/http"

type Api struct {
	ListenAddress string
	Server        *http.Server
	Routes        *http.ServeMux
}
