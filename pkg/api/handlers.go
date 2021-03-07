package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/soer3n/apps-operator/pkg/client"
)

func NewHandler(version string) *Handler {
	return &Handler{
		ApiVersion: version,
	}
}

func (h *Handler) K8sApiGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rc := client.New()
	objs, _ := rc.GetAPIResources(vars["group"], false)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Requested group: %v\n", vars["group"])
	log.Printf("Requested groups: %v\n", objs)
}

func (h *Handler) K8sApiGroupResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rc := client.New()
	apiGroup := vars["resource"] + "." + vars["group"]
	objs := rc.GetResources(rc.Builder("", true), []string{apiGroup})
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Requested resource: %v\n", vars["resource"])
	log.Printf("Requested resources: %v\n", objs)
}
