package utils

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClientInterface repesents interface for mocking custom k8s client
type ClientInterface interface {
	GetResource(name, namespace, resource, group, version string, opts metav1.GetOptions) ([]byte, error)
	ListResources(namespace, resource, group, version string, opts metav1.ListOptions) ([]byte, error)
}

// HTTPClientInterface represents interface for mocking an http client
type HTTPClientInterface interface {
	Get(url string) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}
