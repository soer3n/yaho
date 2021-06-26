package helm

import (
	"net/http"

	"github.com/soer3n/apps-operator/pkg/client"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/getter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8SClientMock struct {
	mock.Mock
	client.ClientInterface
}

type HTTPClientMock struct {
	mock.Mock
	getter.Getter
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
