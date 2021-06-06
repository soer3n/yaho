package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type Client struct {
	RestClientGetter genericclioptions.RESTClientGetter
	Factory          Factory
	Config           *rest.Config
	client           dynamic.Interface
	namespace        string
	opts             ClientOpts
}

type Resource struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
}

type ClientOpts interface {
}
