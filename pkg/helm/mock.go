package helm

import (
	"net/http"

	clientutils "github.com/soer3n/apps-operator/pkg/client"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8SClientMock struct {
	mock.Mock
	client.Client
}

type HTTPClientMock struct {
	mock.Mock
	clientutils.HTTPClientInterface
}

func (client *K8SClientMock) ListResources(namespace, resource, group, version string, opts metav1.ListOptions) ([]byte, error) {
	args := client.Called(namespace, resource, group, version, opts)
	values := args.Get(0).([]byte)
	err := args.Error(1)
	return values, err
}

func (client *K8SClientMock) GetResource(name, namespace, resource, group, version string, opts metav1.GetOptions) ([]byte, error) {
	args := client.Called(name, namespace, resource, group, version, opts)
	values := args.Get(0).([]byte)
	err := args.Error(1)
	return values, err
}

func (getter *HTTPClientMock) Get(url string) (*http.Response, error) {
	args := getter.Called(url)
	values := args.Get(0).(*http.Response)
	err := args.Error(1)
	return values, err
}
