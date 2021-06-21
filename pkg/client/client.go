package client

import (
	// "helm.sh/helm/pkg/kube"
	"context"
	"encoding/json"
	"reflect"
	"sync"

	"helm.sh/helm/v3/pkg/cli"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

var addToScheme sync.Once

func New() *Client {

	env := cli.New()
	getter := env.RESTClientGetter()

	// Add CRDs to the scheme. They are missing by default.
	addToScheme.Do(func() {
		if err := apiextv1.AddToScheme(scheme.Scheme); err != nil {
			// This should never happen.
			panic(err)
		}
		if err := apiextv1beta1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
	})

	foo, err := cmdutil.NewFactory(getter).DynamicClient()

	if err != nil {
		panic(err)
	}

	rc := &Client{}

	rc.DynamicClient = foo

	return rc
}

func (c *Client) GetResource(name, namespace, resource, group, version string, opts metav1.GetOptions) ([]byte, error) {

	c.mu.Lock()
	defer c.mu.Lock()

	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	obj, err := c.DynamicClient.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), name, opts)

	if err != nil {
		return nil, err
	}

	return json.Marshal(obj.UnstructuredContent())
}

func (c *Client) ListResources(namespace, resource, group, version string, opts metav1.ListOptions) ([]byte, error) {

	c.mu.Lock()
	defer c.mu.Lock()

	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	obj, err := c.DynamicClient.Resource(deploymentRes).Namespace(namespace).List(context.TODO(), opts)

	if err != nil {
		return nil, err
	}

	return json.Marshal(obj.UnstructuredContent())
}

func (c *Client) GetAPIResources(apiGroup string, namespaced bool, verbs ...string) ([]byte, error) {

	var resources []ResourceKind

	env := cli.New()
	getter := env.RESTClientGetter()

	// Add CRDs to the scheme. They are missing by default.
	addToScheme.Do(func() {
		if err := apiextv1.AddToScheme(scheme.Scheme); err != nil {
			// This should never happen.
			panic(err)
		}
		if err := apiextv1beta1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
	})

	discoveryclient, err := cmdutil.NewFactory(getter).ToDiscoveryClient()

	if err != nil {
		return []byte{}, err
	}

	lists, err := discoveryclient.ServerPreferredResources()

	if err != nil {
		return []byte{}, err
	}

	for _, list := range lists {

		if len(list.APIResources) == 0 {
			continue
		}

		gv, err := schema.ParseGroupVersion(list.GroupVersion)

		if err != nil {
			continue
		}

		for _, resource := range list.APIResources {

			if len(resource.Verbs) == 0 {
				continue
			}
			// filter apiGroup
			if apiGroup != gv.Group {
				continue
			}
			// filter namespaced
			if namespaced != resource.Namespaced {
				continue
			}
			// filter to resources that support the specified verbs
			if len(verbs) > 0 && !sets.NewString(resource.Verbs...).HasAll(verbs...) {
				continue
			}
			resources = append(resources, ResourceKind{
				APIGroup:        gv.Group,
				APIGroupVersion: gv.String(),
				APIResource:     resource,
			})
		}
	}

	return json.Marshal(reflect.ValueOf(resources).Interface().([]map[string]interface{}))
}
