package helm

import (
	"encoding/json"
	"log"
	"sync"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8SClientMock struct {
	mock.Mock
	client.ClientInterface
	mu  sync.Mutex
	wg  sync.WaitGroup
	err error
}

func (client *K8SClientMock) ListResources(namespace, resource, group, version string, opts metav1.ListOptions) ([]byte, error) {
	defer client.mu.Lock()
	if client.err != nil {
		return nil, client.err
	}
	client.mu.Lock()
	args := client.Called(namespace, resource, group, version, opts)
	values := args.Get(0).([]byte)
	err := args.Error(1)
	client.mu.Unlock()
	return values, err
}

func (client *K8SClientMock) GetResource(name, namespace, resource, group, version string, opts metav1.GetOptions) ([]byte, error) {
	defer client.wg.Done()
	if client.err != nil {
		return nil, client.err
	}
	client.mu.Lock()
	args := client.Called(name, namespace, resource, group, version, opts)
	values := args.Get(0).([]byte)
	err := args.Error(1)
	client.mu.Unlock()
	return values, err
}

func TestGetEntryObject(t *testing.T) {

	clientMock := K8SClientMock{}
	chartSpec := helmv1alpha1.ChartSpec{
		Name:        "chart.Name",
		Home:        "chart.Spec.Home",
		Sources:     []string{"chart.Spec.Sources"},
		Description: "chart.Spec.Description",
		Keywords:    []string{"chart.Spec.Keywords"},
		Versions: []helmv1alpha1.ChartVersion{
			{
				Name: "0.1.0",
				URL:  "https://foo.bar/repo/foo.tar.gz",
			},
		},
		Maintainers: []*chart.Maintainer{
			{
				Name: "chart.Spec.Maintainers",
			},
		},
		Icon:        "chart.Spec.Icon",
		APIVersion:  "chart.Spec.APIVersion",
		Condition:   "chart.Spec.Condition",
		Tags:        "chart.Spec.Tags",
		AppVersion:  "chart.Spec.AppVersion",
		Deprecated:  false,
		Annotations: map[string]string{},
		KubeVersion: "chart.Spec.KubeVersion",
		Type:        "chart.Spec.Type",
	}

	ObjectSpec := &helmv1alpha1.ChartList{
		Items: []helmv1alpha1.Chart{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: chartSpec,
			},
		},
	}
	// selector := "foo"
	rawObjectSpec, _ := json.Marshal(ObjectSpec)

	clientMock.On("ListResources", "foo", "repo", "helm.soer3n.info", "v1alpha1").Return(rawObjectSpec, nil)

	apiObj := &helmv1alpha1.Repo{}
	settings := &cli.EnvSettings{}
	clientMock.wg.Add(1)
	testObj := NewHelmRepo(apiObj, settings, &clientMock)
	clientMock.wg.Wait()
	log.Printf("%v", testObj)

	assert := assert.New(t)
	assert.Equal("foo", "foo", "Structs should be equal.")
}
