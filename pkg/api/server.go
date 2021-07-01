package api

import (
	"crypto/tls"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/soer3n/apps-operator/pkg/client"
)

func New(listen string) *Api {
	api := &Api{
		ListenAddress: ":" + listen,
	}

	if err := api.setHttpServer(); err != nil {
		return nil
	}

	return api
}

func (api *Api) setHttpServer() error {
	api.Server = &http.Server{
		Addr:    api.ListenAddress,
		Handler: api.getRoutes(),
		TLSConfig: &tls.Config{
			NextProtos: []string{"h2", "http/1.1"},
		},
	}
	return nil
}

func (api *Api) getRoutes() *mux.Router {
	m := mux.NewRouter()
	h := NewHandler("", client.New())
	m.HandleFunc("/api/resources/{group}", h.K8sApiGroup)
	m.HandleFunc("/api/resources/{group}/{version}/{resource}", h.K8sApiGroupResources)

	// Serve static files from the frontend/dist directory.
	fs := http.FileServer(http.Dir("./frontend/dist"))
	m.Handle("/", fs)

	return m
}

func (api *Api) Run() error {

	log.Println("start server")

	return api.Server.ListenAndServe()
}
