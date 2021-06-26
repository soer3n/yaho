package helm

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReleaseUpdate(t *testing.T) {

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

	indexFile := getTestIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
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

		configList := testObj.GetParsedConfigMaps()

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
	return chart.Chart{}
}
