package api

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/soer3n/yaho/internal/client"
)

// New represents func for returning struct for managing an api http server
func New(listen string, c *client.Client) *API {
	api := &API{
		ListenAddress: ":" + listen,
		Client:        c,
	}

	if err := api.setHTTPServer(); err != nil {
		return nil
	}

	return api
}

func (api *API) setHTTPServer() error {
	api.Server = &http.Server{
		Addr:    api.ListenAddress,
		Handler: api.getRoutes(),
		TLSConfig: &tls.Config{
			NextProtos: []string{"h2", "http/1.1"},
			MinVersion: tls.VersionTLS12,
		},
	}
	return nil
}

func (api *API) getRoutes() *mux.Router {
	m := mux.NewRouter()
	h := NewHandler("", api.Client)
	m.HandleFunc("/api/resources", h.K8sAPIs).Methods("GET")
	m.HandleFunc("/api/resources/{group}", h.K8sAPIGroup).Methods("GET")
	m.HandleFunc("/api/resources/{group}/{version}/{resource}", h.K8sAPIGroupResources).Methods("GET")
	m.HandleFunc("/api/resources/{group}/{version}/{resource}", h.K8sAPIGroupResources).Methods("GET")
	m.Path("/api/resources/{group}/{version}/{resource}").Queries("namespace", "{namespace}").HandlerFunc(h.K8sCreateAPIObject).Methods("POST")
	m.Path("/api/resources/{group}/{version}/{resource}/{name}").Queries("namespace", "{namespace}").HandlerFunc(h.K8sAPIObject).Methods("GET")
	m.Path("/api/resources/{group}/{version}/{resource}/{name}").Queries("namespace", "{namespace}").HandlerFunc(h.K8sDeleteAPIObject).Methods("DELETE")
	// Serve static files from the frontend/dist directory.
	fs := http.FileServer(http.Dir("./frontend/dist"))
	m.Handle("/", fs)

	return m
}

// Run represents func for starting an http server
func (api *API) Run() error {

	log.Println("start server")

	return api.Server.ListenAndServe()
}
