package helm

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReleaseUpdate(t *testing.T) {

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
