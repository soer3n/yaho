package api

import (
	"crypto/tls"
	"net/http"
)

func NewAPI() *Api {
	return &Api{
		ListenAddress: ":9090",
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

func (api *Api) Start() error {

	if err := api.setHttpServer(); err != nil {
		return err
	}

	return api.Server.ListenAndServe()
}
