package helm

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReleaseConfigMaps(t *testing.T) {

	clientMock := K8SClientMock{}
	httpMock := HTTPClientMock{}
	settings := cli.New()
	ObjectSpec := getTestChartListSpec()
	apiObjList := getTestReleaseSpecs()
	rawObjectSpec, _ := json.Marshal(ObjectSpec)
	chartRawObj, _ := json.Marshal(getTestChartSpec())

	clientMock.On("GetResource", "repo", "", "repos", "helm.soer3n.info", "v1alpha1", metav1.GetOptions{}).Return(rawObjectSpec, nil)
	clientMock.On("GetResource", "chart", "", "charts", "helm.soer3n.info", "v1alpha1", metav1.GetOptions{}).Return(chartRawObj, nil)

	/*expected :=  getExpectedTestCharts(clientMock)*/

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(httpResponse, nil)

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		testObj := NewHelmRelease(apiObj, settings, &clientMock, &httpMock)
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range apiObj.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		testObj.Version = apiObj.Spec.Version
		configList := testObj.GetParsedConfigMaps()

		// assert.Equal(expected, charts, "Structs should be equal.")
		assert.NotNil(configList)
	}
}

func TestReleaseUpdate(t *testing.T) {

	clientMock := K8SClientMock{}
	httpMock := HTTPClientMock{}
	settings := cli.New()
	ObjectSpec := getTestChartListSpec()
	apiObjList := getTestReleaseSpecs()
	rawObjectSpec, _ := json.Marshal(ObjectSpec)
	chartRawObj, _ := json.Marshal(getTestChartSpec())
	values := map[string]string{
		"foo": "bar",
		"bar": "foo",
	}
	castedValues, _ := json.Marshal(values)
	configmapRaw, _ := json.Marshal(v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "",
		},
		Data: map[string]string{
			"values": string(castedValues),
		},
	})

	chartList := helmv1alpha1.ChartList{
		Items: []helmv1alpha1.Chart{},
	}
	chartListRawObj, _ := json.Marshal(chartList)

	clientMock.On("GetResource", "repo", "", "repos", "helm.soer3n.info", "v1alpha1", metav1.GetOptions{}).Return(rawObjectSpec, nil)
	clientMock.On("ListResources", "", "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{
		LabelSelector: "repo=repo",
	}).Return(chartListRawObj, nil)
	clientMock.On("GetResource", "chart", "", "charts", "helm.soer3n.info", "v1alpha1", metav1.GetOptions{}).Return(chartRawObj, nil)
	clientMock.On("GetResource", "helm-tmpl-chart-0.0.1", "", "configmaps", "", "v1", metav1.GetOptions{}).Return(configmapRaw, nil)
	clientMock.On("GetResource", "helm-crds-chart-0.0.1", "", "configmaps", "", "v1", metav1.GetOptions{}).Return(configmapRaw, nil)
	clientMock.On("GetResource", "helm-default-chart-0.0.1", "", "configmaps", "", "v1", metav1.GetOptions{}).Return(configmapRaw, nil)

	/*expected :=  getExpectedTestCharts(clientMock)*/

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(httpResponse, nil)

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		testObj := NewHelmRelease(apiObj, settings, &clientMock, &httpMock)
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range apiObj.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		testObj.Version = apiObj.Spec.Version
		configList := testObj.Update()

		// assert.Equal(expected, charts, "Structs should be equal.")
		assert.NotNil(configList)
	}
}

func getTestReleaseSpecs() []*helmv1alpha1.Release {
	return []*helmv1alpha1.Release{
		{
			Spec: helmv1alpha1.ReleaseSpec{
				Name:    "test",
				Chart:   "chart",
				Repo:    "repo",
				Version: "0.0.1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"label": "selector",
				},
			},
			Spec: helmv1alpha1.ReleaseSpec{
				Name:    "test",
				Chart:   "chart",
				Repo:    "repo",
				Version: "0.0.1",
			},
		},
	}
}

func getTestChartSpec() helmv1alpha1.Chart {
	return helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "chart",
		},
		Spec: helmv1alpha1.ChartSpec{
			Name: "chart",
			Versions: []helmv1alpha1.ChartVersion{
				{
					Name: "0.0.1",
					URL:  "https://foo.bar/charts/foo-0.0.1.tgz",
				},
			},
		},
	}
}

func getTestChartForCompressing() chart.Chart {
	return chart.Chart{
		Templates: []*chart.File{},
		Values:    map[string]interface{}{},
		Metadata: &chart.Metadata{
			Name:    "meta",
			Version: "0.0.1",
		},
	}
}
