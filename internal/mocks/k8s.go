package mocks

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (client *K8SClientMock) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := client.Called(ctx, list, opts)
	err := args.Error(0)
	return err
}

func (client *K8SClientMock) Get(ctx context.Context, key types.NamespacedName, obj client.Object) error {
	args := client.Called(ctx, key, obj)
	err := args.Error(0)
	return err
}

func (getter *K8SDynamicClientMock) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	args := getter.Called(resource)
	values := args.Get(0).(dynamic.NamespaceableResourceInterface)
	return values
}

func (getter *K8SNamespaceMock) Namespace(resource string) dynamic.ResourceInterface {
	args := getter.Called(resource)
	values := args.Get(0).(dynamic.ResourceInterface)
	return values
}

func (getter *K8SResourceMock) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	args := getter.Called(name)
	values := args.Get(0).(*unstructured.Unstructured)
	err := args.Error(1)
	return values, err
}

func (getter *K8SResourceMock) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	args := getter.Called(opts)
	values := args.Get(0).(*unstructured.UnstructuredList)
	err := args.Error(1)
	return values, err
}

func (getter *K8SResourceMock) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	args := getter.Called(opts)
	values := args.Get(0).(watch.Interface)
	err := args.Error(1)
	return values, err
}

func (getter *K8SResourceMock) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	args := getter.Called(obj)
	values := args.Get(0).(*unstructured.Unstructured)
	err := args.Error(1)
	return values, err
}

func (getter *K8SResourceMock) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	args := getter.Called(options)
	values := args.Get(0).(*unstructured.Unstructured)
	err := args.Error(1)
	return values, err
}

func (getter *K8SResourceMock) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	args := getter.Called(obj)
	values := args.Get(0).(*unstructured.Unstructured)
	err := args.Error(1)
	return values, err
}

func (getter *K8SResourceMock) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	args := getter.Called(obj)
	values := args.Get(0).(*unstructured.Unstructured)
	err := args.Error(1)
	return values, err
}

func (getter *K8SResourceMock) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	args := getter.Called(name)
	err := args.Error(1)
	return err
}

func (getter *K8SResourceMock) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	args := getter.Called(options)
	err := args.Error(1)
	return err
}