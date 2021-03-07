package client

import (
	// "helm.sh/helm/pkg/kube"
	"sync"

	"helm.sh/helm/v3/pkg/cli"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
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

	return c.Factory.NewBuilder().Unstructured().Schema(schema).ContinueOnError().NamespaceParam(namespace).DefaultNamespace()
}

func (c *Client) Get(builder *resource.Builder, args []string) []runtime.Object {
	result := builder.
		LabelSelectorParam("").
		FieldSelectorParam("").
		ResourceTypeOrNameArgs(true, args...).
		Do()

	var infos []*resource.Info
	var err error

	if infos, err = result.Infos(); err != nil {
		return []runtime.Object{}
	}

	objs := make([]runtime.Object, len(infos))

	for ix := range infos {
		objs[ix] = infos[ix].Object
	}

	return objs
}
