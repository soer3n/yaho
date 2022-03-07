package helm

import (
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestChartSubCharts(t *testing.T) {
	settings := cli.New()
	cases := testcases.GetTestHelmChartMaps()
	clientMock, httpMock := helmmocks.GetChartMock()
	var err error

	assert := assert.New(t)

	for _, v := range cases {
		ver := v.Input.(*helmv1alpha1.Chart)
		testObj := chart.New(ver, settings, &runtime.Scheme{}, logf.Log, clientMock, httpMock, kube.Client{})
		err = testObj.CreateOrUpdateSubCharts()
		assert.Equal(v.ReturnError, err)

	}

	for _, i := range testcases.GetTestRepoChartVersions() {
		ver := i.Input.(*helmv1alpha1.Chart)
		testObj := chart.New(ver, settings, &runtime.Scheme{}, logf.Log, clientMock, httpMock, kube.Client{})
		err = testObj.CreateOrUpdateSubCharts()
		assert.Equal(i.ReturnError, err)
	}
}

func TestChartUpdate(t *testing.T) {
	settings := cli.New()
	cases := testcases.GetTestHelmChartMaps()
	clientMock, httpMock := helmmocks.GetChartMock()
	var err error

	assert := assert.New(t)

	for _, v := range cases {
		ver := v.Input.(*helmv1alpha1.Chart)
		testObj := chart.New(ver, settings, &runtime.Scheme{}, logf.Log, clientMock, httpMock, kube.Client{})
		err = testObj.Update(ver)
		assert.Equal(v.ReturnError, err)

	}

	for _, i := range testcases.GetTestRepoChartVersions() {
		ver := i.Input.(*helmv1alpha1.Chart)
		testObj := chart.New(ver, settings, &runtime.Scheme{}, logf.Log, clientMock, httpMock, kube.Client{})
		err = testObj.Update(ver)
		assert.Equal(i.ReturnError, err)
	}
}
