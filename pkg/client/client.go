package client

import (
	// "helm.sh/helm/pkg/kube"
	"log"
	"sync"

	"helm.sh/helm/v3/pkg/cli"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/resource"
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

	return &Client{
		RestClientGetter: getter,
		Factory:          cmdutil.NewFactory(getter),
	}
}

func (c *Client) Builder(namespace string, validate bool) *resource.Builder {

	schema, err := c.Factory.Validator(validate)
	if err != nil {
		return &resource.Builder{}
	}

	log.Println("creating builder...")

	return c.Factory.NewBuilder().Unstructured().Schema(schema).ContinueOnError().NamespaceParam(namespace).DefaultNamespace()
}

func (c *Client) GetResources(builder *resource.Builder, args []string) []runtime.Object {
	result := builder.
		ResourceTypeOrNameArgs(true, args...).
		Do()

	var infos []*resource.Info
	var err error

	if infos, err = result.Infos(); err != nil {
		log.Println("error on getting infos")
		log.Printf("%v", err)
		return []runtime.Object{}
	}

	objs := make([]runtime.Object, len(infos))

	for ix := range infos {
		objs[ix] = infos[ix].Object
	}

	return objs
}

func (c *Client) GetAPIResources(apiGroup string, namespaced bool, verbs ...string) ([]Resource, error) {

	var resources []Resource
	discoveryclient, err := c.Factory.ToDiscoveryClient()

	if err != nil {
		return resources, err
	}

	lists, err := discoveryclient.ServerPreferredResources()

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

	return resources, nil
}
