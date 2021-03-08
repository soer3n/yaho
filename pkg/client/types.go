package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Client struct {
	RestClientGetter genericclioptions.RESTClientGetter
	Factory          Factory
}

type Resource struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
}

type APIResources struct {
	List []Resource
}

type Resources struct {
	List []runtime.Object
}
