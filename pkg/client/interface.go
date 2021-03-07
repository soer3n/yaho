package client

import (
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/validation"
)

// Factory provides abstractions that allow the Kubectl command to be extended across multiple types
// of resources and different API sets.
type Factory interface {
	// ToRawKubeConfigLoader return kubeconfig loader as-is
	ToRawKubeConfigLoader() clientcmd.ClientConfig
	// KubernetesClientSet gives you back an external clientset
	KubernetesClientSet() (*kubernetes.Clientset, error)
	// NewBuilder returns an object that assists in loading objects from both disk and the server
	// and which implements the common patterns for CLI interactions with generic resources.
	NewBuilder() *resource.Builder
	// Returns a schema that can validate objects stored on disk.
	Validator(validate bool) (validation.Schema, error)
}
