package api

import (
	"crypto/tls"
	"log"
	"net/http"
)

func New(listen string) *Api {
	return &Api{
		ListenAddress: ":" + listen,
	}
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

func (api *Api) getRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/paas", func(w http.ResponseWriter, r *http.Request) {

	})

	// Serve static files from the frontend/dist directory.
	fs := http.FileServer(http.Dir("./frontend/dist"))
	mux.Handle("/", fs)

	return mux
}

func (api *Api) Run() error {

	if err := api.setHttpServer(); err != nil {
		return err
	}

	log.Println("start server")

	return api.Server.ListenAndServe()
}
