package api

import (
	"encoding/json"
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
	objs, _ := rc.GetAPIResources(vars["group"], true)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	payload, err := json.Marshal(objs)

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	log.Printf("%v", string(payload))
	fmt.Fprintf(w, "%v", string(payload))
}

func (h *Handler) K8sApiGroupResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	rc := client.New()
	apiGroup := vars["resource"] + "." + vars["group"]
	objs := rc.GetResources(rc.Builder("", true), []string{apiGroup})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	payload, err := json.Marshal(objs)

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	log.Printf("%v", string(payload))
	fmt.Fprintf(w, "%v", string(payload))

}
