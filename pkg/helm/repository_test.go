package helm

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
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

func TestGetCharts(t *testing.T) {

	clientMock := K8SClientMock{}
	httpMock := HTTPClientMock{}
	settings := cli.New()
	ObjectSpec := getTestChartListSpec()
	apiObjList := getTestRepoSpecs()
	rawObjectSpec, _ := json.Marshal(ObjectSpec)
	emptyRawObj, _ := json.Marshal(helmv1alpha1.ChartList{})

	clientMock.On("ListResources", "", "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{
		LabelSelector: "label=selector",
	}).Return(rawObjectSpec, nil)

	clientMock.On("ListResources", "", "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{
		LabelSelector: "",
	}).Return(emptyRawObj, nil)

	clientMock.On("ListResources", "", "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{
		LabelSelector: "label=notpresent",
	}).Return(emptyRawObj, nil)

	/*expected :=  getExpectedTestCharts(clientMock)*/

	indexFile := getTestIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/index.yaml").Return(httpResponse, nil)

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		testObj := NewHelmRepo(apiObj, settings, &clientMock, &httpMock)
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range apiObj.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		_, err := testObj.GetCharts(settings, selectors)

		// assert.Equal(expected, charts, "Structs should be equal.")
		assert.Nil(err)
	}
}

func getTestChartListSpec() *helmv1alpha1.ChartList {

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

	return &helmv1alpha1.ChartList{
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
}

func getTestRepoSpecs() []*helmv1alpha1.Repo {
	return []*helmv1alpha1.Repo{
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"label": "selector",
				},
			},
			Spec: helmv1alpha1.RepoSpec{
				Name: "test",
				Url:  "https://foo.bar/charts",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"label": "selector",
				},
			},
			Spec: helmv1alpha1.RepoSpec{
				Name: "test",
				Url:  "https://foo.bar/charts",
				Auth: &helmv1alpha1.Auth{
					User:     "foo",
					Password: "encrypted",
					Cert:     "certContent",
					Key:      "keyContent",
					Ca:       "certCa",
				},
			},
		},
		{
			Spec: helmv1alpha1.RepoSpec{
				Name: "notpresent",
				Url:  "https://foo.bar/charts",
			},
		},
		/*{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"label": "notpresent",
				},
			},
			Spec: helmv1alpha1.RepoSpec{
				Name: "notpresent",
				Url:  "https://foo.bar/charts",
				Auth: &helmv1alpha1.Auth{
					User:     "foo",
					Password: "encrypted",
					Cert:     "certContent",
					Key:      "keyContent",
					Ca:       "certCa",
				},
			},
		},*/
	}
}

func getTestIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}

func getExpectedTestCharts(c client.ClientInterface) []*HelmChart {
	return []*HelmChart{
		{
			Repo:      "testrepo",
			Settings:  cli.New(),
			k8sClient: c,
			Versions: HelmChartVersions{
				{},
			},
		},
	}
}
