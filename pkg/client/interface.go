package client

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const GroupName = "helm.soer3n.info"
const GroupVersion = "v1alpha1"

type AppsV1Alpha1Interface interface {
	AppsResources(namespace string) AppsInterface
}

type AppsV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*AppsV1Alpha1Client, error) {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &AppsV1Alpha1Client{restClient: client}, nil
}

func (c *AppsV1Alpha1Client) AppsResources(namespace string) AppsInterface {
	return &AppsClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}
