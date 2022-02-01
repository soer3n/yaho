package client

import (

	// "helm.sh/helm/pkg/kube"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
)

var addToScheme sync.Once

// New represents initialization of needed data for running request by client
func New() *Client {
	// Add CRDs to the scheme. They are missing by default.
	addToScheme.Do(func() {
		if err := apiextv1.AddToScheme(scheme.Scheme); err != nil {
			// This should never happen.
			panic(err)
		}
		if err := apiextv1beta1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
		if err := helmv1alpha1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
	})

	rc := &Client{}

	rc.DynamicClient = getDynamicKubernetesClient()
	rc.TypedClient = getTypedKubernetesClient()
	rc.DiscoverClient = getDiscoveryKubernetesClient()

	return rc
}

func getClusterConfig() *rest.Config {
	restConfig, err := rest.InClusterConfig()
	if err != nil {

		configLoadRules := clientcmd.NewDefaultClientConfigLoadingRules()
		config, _ := configLoadRules.Load()
		copy := *config

		clientConfig := clientcmd.NewDefaultClientConfig(copy, &clientcmd.ConfigOverrides{})
		restConfig, err = clientConfig.ClientConfig()

		if err != nil {
			panic(err.Error())
		}
	}

	return restConfig
}

func getTypedKubernetesClient() *kubernetes.Clientset {
	config := getClusterConfig()
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func getDynamicKubernetesClient() dynamic.Interface {
	config := getClusterConfig()
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return client
}

func getDiscoveryKubernetesClient() *discovery.DiscoveryClient {
	config := getClusterConfig()
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		panic(err)
	}

	return client
}

// GetResource represents func for returning k8s unstructured resource by given parameters
func (c *Client) GetResource(name, namespace, resource, group, version string, opts metav1.GetOptions) ([]byte, error) {
	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	obj, err := c.DynamicClient.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), name, opts)
	if err != nil {
		return nil, err
	}

	return json.Marshal(obj.UnstructuredContent())
}

// ListResources represents func for returning k8s unstructured resource list by given parameters
func (c *Client) ListResources(namespace, resource, group, version string, opts metav1.ListOptions) ([]byte, error) {
	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	obj, err := c.DynamicClient.Resource(deploymentRes).Namespace(namespace).List(context.TODO(), opts)
	if err != nil {
		return nil, err
	}

	return json.Marshal(obj.UnstructuredContent())
}

// CreateResource represents func for returning newly created k8s unstructured resource by given parameters
func (c *Client) CreateResource(obj *unstructured.Unstructured, namespace, resource, group, version string, opts metav1.CreateOptions) ([]byte, error) {
	var err error
	var updateObj *unstructured.Unstructured

	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	updateObj = obj.DeepCopy()

	metaData, _ := obj.Object["metadata"].(map[string]interface{})
	objName, _ := metaData["name"].(string)

	if _, err = c.DynamicClient.Resource(deploymentRes).Namespace(namespace).Get(context.TODO(), objName, metav1.GetOptions{}); err != nil {
		if obj, err = c.DynamicClient.Resource(deploymentRes).Namespace(namespace).Create(context.TODO(), obj, opts); err != nil {
			fmt.Print(err.Error())
			return nil, err
		}
		return json.Marshal(obj.UnstructuredContent())
	}

	updateOpts := metav1.UpdateOptions{}

	if obj, err = c.DynamicClient.Resource(deploymentRes).Namespace(namespace).Update(context.TODO(), updateObj, updateOpts); err != nil {
		fmt.Print(err.Error())
		return nil, err
	}

	return json.Marshal(obj.UnstructuredContent())
}

// DeleteResource represents func for returning response for deletion process of k8s unstructured resource by given parameters
func (c *Client) DeleteResource(name, namespace, resource, group, version string, opts metav1.DeleteOptions) error {
	deploymentRes := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	err := c.DynamicClient.Resource(deploymentRes).Namespace(namespace).Delete(context.TODO(), name, opts)
	if err != nil {
		fmt.Print(err.Error())
		return err
	}

	return nil
}

// GetAPIResources represents func for returning resource kinds by given api group name
func (c *Client) GetAPIResources(apiGroup string, namespaced bool, verbs ...string) ([]byte, error) {
	var resources []ResourceKind
	lists, err := c.DiscoverClient.ServerPreferredResources()
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

	return json.Marshal(resources)
}

// GetAPIGroups represents func for returning resource kinds by given api group name
func (c *Client) GetAPIGroups() ([]byte, error) {
	resources := make(map[string][]string)
	lists, err := c.DiscoverClient.ServerPreferredResources()
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

			group := gv.Group

			if group == "" {
				group = "core"
			}

			if _, ok := resources[group]; !ok {
				resources[group] = append([]string{}, resource.Name)
				continue
			}

			resources[group] = append(resources[group], resource.Name)
		}
	}

	return json.Marshal(resources)
}
