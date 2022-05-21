package helm

import (
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestChartSubCharts(t *testing.T) {
	settings := cli.New()
	cases := testcases.GetTestHelmChartMaps()
	clientMock, httpMock := helmmocks.GetChartMock()

	_ = helmv1alpha1.AddToScheme(scheme.Scheme)

	assert := assert.New(t)

	for _, v := range cases {
		ver := v.Input.(*helmv1alpha1.Chart)
		testObj, err := chart.New(ver, ver.Namespace, settings, scheme.Scheme, logf.Log, clientMock, httpMock, cli.New().RESTClientGetter(), []byte(""))
		assert.Equal(nil, err)
		err = testObj.CreateOrUpdateSubCharts()
		assert.Equal(v.ReturnError["subCharts"], err)

	}
}

func TestChartUpdate(t *testing.T) {
	settings := cli.New()
	cases := testcases.GetTestHelmChartMaps()
	clientMock, httpMock := helmmocks.GetChartMock()

	assert := assert.New(t)

	for _, v := range cases {
		ver := v.Input.(*helmv1alpha1.Chart)
		testObj, err := chart.New(ver, ver.Namespace, settings, &runtime.Scheme{}, logf.Log, clientMock, httpMock, cli.New().RESTClientGetter(), []byte(""))
		assert.Equal(nil, err)
		err = testObj.Update(ver)
		assert.Equal(v.ReturnError["update"], err)

	}
}
