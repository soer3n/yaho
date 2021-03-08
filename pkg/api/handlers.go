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
	var payload []byte
	var err error
	vars := mux.Vars(r)
	rc := client.New()
	objs := rc.GetAPIResources(vars["group"], true)

	if payload, err = json.Marshal(objs); err != nil {
		fmt.Println(err.Error())
		return
	}

	log.Printf("%v", string(payload))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

func (h *Handler) K8sApiGroupResources(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var err error
	vars := mux.Vars(r)
	rc := client.New()
	apiGroup := vars["resource"] + "." + vars["group"]
	objs := rc.GetResources(rc.Builder("", true), []string{apiGroup})

	if payload, err = json.Marshal(objs); err != nil {
		fmt.Println(err.Error())
		return
	}

	log.Printf("%v", string(payload))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}
