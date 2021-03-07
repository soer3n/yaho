package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func NewHandler(version string) *Handler {
	return &Handler{
		ApiVersion: version,
	}
}

func (h *Handler) K8sApiGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Requested group: %v\n", vars["group"])
}

func (h *Handler) K8sApiGroupResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Requested group: %v\n", vars["group"])
}
