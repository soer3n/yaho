package client

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

type Client struct {
	DynamicClient  dynamic.Interface
	DiscoverClient discovery.ServerResourcesInterface
	ClientOpts
	ClientInterface
}

type ResourceKind struct {
	APIGroup        string
	APIGroupVersion string
	APIResource     metav1.APIResource
}

type ClientOpts interface {
}
