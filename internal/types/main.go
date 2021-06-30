package types

import (
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClientInterface interface {
	GetResource(name, namespace, resource, group, version string, opts metav1.GetOptions) ([]byte, error)
	ListResources(namespace, resource, group, version string, opts metav1.ListOptions) ([]byte, error)
}

type HTTPClientInterface interface {
	Get(url string) (*http.Response, error)
}
