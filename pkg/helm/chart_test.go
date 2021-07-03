package helm

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestChartCreateTemplates(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "foo", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)
	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.2.tgz").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)

	assert := assert.New(t)

	testObj := NewChart(getTestRepoChartVersions(), settings, "test", &clientMock, &httpMock)
	err := testObj.CreateTemplates()

	assert.Nil(err)
}

func TestChartCreateConfigMaps(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "foo", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)
	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.2.tgz").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)

	assert := assert.New(t)

	testObj := NewChart(getTestRepoChartVersions(), settings, "test", &clientMock, &httpMock)
	maps := testObj.CreateConfigMaps()

	assert.NotNil(maps)
}

func TestChartAddOrUpdateMap(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "foo", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	assert := assert.New(t)

	testObj := NewChart(getTestRepoChartVersions(), settings, "test", &clientMock, &httpMock)
	maps := testObj.AddOrUpdateChartMap(getTestHelmChartMapNotFound(), getTestChartRepo())
	assert.Len(maps, 2)
	maps = testObj.AddOrUpdateChartMap(getTestHelmChartMap(), getTestChartRepo())
	assert.Len(maps, 1)
}

func getTestRepoChartVersions() []*repo.ChartVersion {
	return []*repo.ChartVersion{
		{
			Metadata: &chart.Metadata{
				Name:    "foo",
				Version: "0.0.1",
				Dependencies: []*chart.Dependency{
					{
						Name:       "dep",
						Version:    "0.1.1",
						Repository: "repo",
					},
				},
			},
			URLs: []string{"https://foo.bar/charts/foo-0.0.1.tgz"},
		},
	}
}

func getTestHelmChartMap() map[string]*helmv1alpha1.Chart {
	return map[string]*helmv1alpha1.Chart{
		"foo": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bar",
				Namespace: "",
			},
			Spec: helmv1alpha1.ChartSpec{
				Name: "baz",
				Versions: []helmv1alpha1.ChartVersion{
					{
						Name: "0.0.2",
						URL:  "nodomain.com",
					},
				},
			},
		},
	}
}

func getTestHelmChartMapNotFound() map[string]*helmv1alpha1.Chart {
	return map[string]*helmv1alpha1.Chart{
		"bar": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bar",
				Namespace: "",
			},
			Spec: helmv1alpha1.ChartSpec{
				Name: "baz",
				Versions: []helmv1alpha1.ChartVersion{
					{
						Name: "0.0.2",
						URL:  "nodomain.com",
						Dependencies: []helmv1alpha1.ChartDep{
							{
								Name:    "dep",
								Repo:    "repo",
								Version: "0.1.1",
							},
						},
					},
				},
			},
		},
	}
}

func getTestChartRepo() *helmv1alpha1.Repo {
	return &helmv1alpha1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo",
			Namespace: "",
		},
		Spec: helmv1alpha1.RepoSpec{
			Name: "repo",
		},
	}
}
