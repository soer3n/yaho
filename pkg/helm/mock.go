package helm

import (
	"context"
	"net/http"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (getter *HTTPClientMock) Get(url string) (*http.Response, error) {
	args := getter.Called(url)
	values := args.Get(0).(*http.Response)
	err := args.Error(1)
	return values, err
}
