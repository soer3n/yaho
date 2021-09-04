package helm

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/mocks"
	inttypes "github.com/soer3n/yaho/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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

	req, _ := http.NewRequest(http.MethodGet, "https://foo.bar/charts/foo-0.0.1.tgz", nil)

	httpMock.On("Do",
		req).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)

	reqEmpty, _ := http.NewRequest(http.MethodGet, "https://foo.bar/charts/foo-0.0.2.tgz", nil)
	httpMock.On("Do",
		reqEmpty).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)

	assert := assert.New(t)

	for _, v := range getTestRepoChartVersions() {
		ver := v.Input.([]*repo.ChartVersion)
		testObj := NewChart(ver, settings, "test", &clientMock, &httpMock, kube.Client{})
		maps := testObj.CreateConfigMaps()
		assert.NotNil(maps)
	}
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

	assert := assert.New(t)

	for _, v := range getTestHelmChartMaps() {
		for _, i := range getTestRepoChartVersions() {
			ver := i.Input.([]*repo.ChartVersion)
			testObj := NewChart(ver, settings, "test", &clientMock, &httpMock, kube.Client{})
			rel, _ := v.Input.(map[string]*helmv1alpha1.Chart)
			maps := testObj.AddOrUpdateChartMap(rel, getTestChartRepo())
			expectedLen, _ := v.ReturnValue.(int)
			assert.Equal(len(maps), expectedLen)
		}
	}
}

func getTestRepoChartVersions() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			Input: []*repo.ChartVersion{
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
			},
			ReturnValue: "",
		},
	}
}

func getTestHelmChartMaps() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			Input: map[string]*helmv1alpha1.Chart{
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
			},
			ReturnError: nil,
			ReturnValue: 1,
		},
		{
			Input: map[string]*helmv1alpha1.Chart{
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
								Dependencies: []*helmv1alpha1.ChartDep{
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
			},
			ReturnError: nil,
			ReturnValue: 2,
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
