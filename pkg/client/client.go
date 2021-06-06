package client

import (
	// "helm.sh/helm/pkg/kube"
	"context"
	"encoding/json"
	"flag"
	"path/filepath"
	"reflect"
	"sync"

	"helm.sh/helm/v3/pkg/cli"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
)

var addToScheme sync.Once

func New() *Client {
	env := cli.New()

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

	getter := env.RESTClientGetter()

	kubeconfig := new(string)
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()

	if err != nil {
		if home := homedir.HomeDir(); home != "" {
			*kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			*kubeconfig = ""
		}
		flag.Parse()

		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err)
		}
	}

	return &Client{
		RestClientGetter: getter,
		Factory:          cmdutil.NewFactory(getter),
		Config:           config,
	}
}

func (c *Client) SetClient() error {
	var err error
	c.client, err = dynamic.NewForConfig(c.Config)
	return err
}

func (c *Client) SetOptions(opts ClientOpts) *Client {
	c.opts = opts
	return c
}

func (c Client) GetResource(name, namespace, resource, group, version string) ([]byte, error) {
	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	opts := metav1.GetOptions{}

	if c.client == nil {
		_ = c.SetClient()
	}

	if c.opts != nil {
		opts = c.opts.(metav1.GetOptions)
	}

	obj, err := c.client.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), name, opts)

	if err != nil {
		return nil, err
	}
	return json.Marshal(obj.UnstructuredContent())
}

func (c Client) ListResources(namespace, resource, group, version string) ([]byte, error) {
	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	opts := metav1.ListOptions{}

	if c.client == nil {
		_ = c.SetClient()
	}

	if c.opts != nil {
		opts = c.opts.(metav1.ListOptions)
	}

	obj, err := c.client.Resource(deploymentRes).Namespace(namespace).List(context.TODO(), opts)

	if err != nil {
		return nil, err
	}
	return json.Marshal(obj.UnstructuredContent())
}

func (c *Client) GetAPIResources(apiGroup string, namespaced bool, verbs ...string) ([]byte, error) {

	var resources []Resource

	discoveryclient, err := c.Factory.ToDiscoveryClient()

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
			resources = append(resources, Resource{
				APIGroup:        gv.Group,
				APIGroupVersion: gv.String(),
				APIResource:     resource,
			})
		}
	}

	return json.Marshal(reflect.ValueOf(resources).Interface().([]map[string]interface{}))
}
