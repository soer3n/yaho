package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/soer3n/apps-operator/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NewHandler represents func for returning struct for managing server routes logic
func NewHandler(version string, c *client.Client) *Handler {
	return &Handler{
		APIVersion: version,
		K8SClient:  c,
	}
}

// K8sAPIGroup represents func for returning  resource kinds related to an api group
func (h *Handler) K8sAPIGroup(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var err error

	vars := mux.Vars(r)
	data := make([]map[string]interface{}, 0)
	response := &Response{
		Message: "Fail",
	}

	objs, err := h.K8SClient.GetAPIResources(vars["group"], true)

	if err := json.Unmarshal(objs, &data); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Data = map[string]interface{}{"objects": data}
	response.Message = "Success"

	if payload, err = json.Marshal(response); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("%v", string(payload))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

// K8sAPIObject represents func for returning  resource object related to an api group
func (h *Handler) K8sAPIObject(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var resource, version, group, namespace, name string
	var ok bool
	var err error

	vars := mux.Vars(r)
	data := make(map[string]interface{}, 0)
	response := &Response{
		Message: "Fail",
	}

	if resource, ok = vars["resource"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if group, ok = vars["group"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if version, ok = vars["version"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if namespace, ok = vars["namespace"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if name, ok = vars["name"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	objs, err := h.K8SClient.GetResource(name, namespace, resource, group, version, metav1.GetOptions{})

	if err := json.Unmarshal(objs, &data); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Data = data
	response.Message = "Success"

	if payload, err = json.Marshal(response); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("%v", string(payload))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

// K8sCreateAPIObject represents func for returning  resource object related to an api group
func (h *Handler) K8sCreateAPIObject(w http.ResponseWriter, r *http.Request) {
	var payload []byte
	var resource, version, group, namespace string
	var ok bool
	var err error
	var obj *unstructured.Unstructured

	vars := mux.Vars(r)
	data := make(map[string]interface{}, 0)
	response := &Response{
		Message: "Fail",
	}

	if resource, ok = vars["resource"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if group, ok = vars["group"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if version, ok = vars["version"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if namespace, ok = vars["namespace"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&obj)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	objs, err := h.K8SClient.CreateResource(obj, namespace, resource, group, version, metav1.CreateOptions{})

	if err := json.Unmarshal(objs, &data); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Data = data
	response.Message = "Success"

	if payload, err = json.Marshal(response); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("%v", string(payload))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

// K8sAPIGroupResources represents func for returning resources related to a resource kind of an api group
func (h *Handler) K8sAPIGroupResources(w http.ResponseWriter, r *http.Request) {

	var payload []byte
	var resource, version, group string
	var ok bool
	var err error

	vars := mux.Vars(r)
	data := make(map[string]interface{}, 0)
	response := &Response{
		Message: "Fail",
	}

	if resource, ok = vars["resource"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if group, ok = vars["group"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	if version, ok = vars["version"]; !ok {
		w.WriteHeader(http.StatusPreconditionFailed)
		return
	}

	namespace := ""

	if _, ok = vars["namespace"]; ok {
		namespace = vars["namespace"]
	}

	objs, err := h.K8SClient.ListResources(namespace, resource, group, version, metav1.ListOptions{})

	if err := json.Unmarshal(objs, &data); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Data = data
	response.Message = "Success"

	if payload, err = json.Marshal(response); err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("%v", string(payload))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}
