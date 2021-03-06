package client

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type PaaSInterface interface {
	List(opts metav1.ListOptions) (*v1alpha1.AppsList, error)
	Get(name string, options metav1.GetOptions) (*v1alpha1.Apps, error)
	Create(*v1alpha1.Apps) (*v1alpha1.Apps, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type AppsClient struct {
	restClient rest.Interface
	ns         string
}

func (c *AppsClient) List(opts metav1.ListOptions) (*v1alpha1.AppsList, error) {
	result := v1alpha1.AppsList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("paas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(context.TODO()).
		Into(&result)

	return &result, err
}

func (c *AppsClient) Get(name string, opts metav1.GetOptions) (*v1alpha1.Apps, error) {
	result := v1alpha1.Apps{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("paas").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(context.TODO()).
		Into(&result)

	return &result, err
}

func (c *paasClient) Create(paas *v1alpha1.PaaS) (*v1alpha1.PaaS, error) {
	result := v1alpha1.PaaS{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource("paas").
		Body(paas).
		Do(context.TODO()).
		Into(&result)

	return &result, err
}

func (c *paasClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource("paas").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(context.TODO())
}
