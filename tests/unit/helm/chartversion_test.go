package helm

import (
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	"github.com/soer3n/yaho/internal/chartversion"
	"github.com/soer3n/yaho/internal/utils"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestChartVersion(t *testing.T) {
	cases := testcases.GetTestHelmChartMaps()
	clientMock, httpMock := helmmocks.GetChartMock()

	assert := assert.New(t)

	_ = helmv1alpha1.AddToScheme(scheme.Scheme)

	for _, v := range cases {
		ver := v.Input.(*helmv1alpha1.Chart)
		cv := testcases.GetChartVersions(ver.Spec.Name)
		testObj, err := chartversion.New(v.ChartVersion, ver.Namespace, ver, map[string]interface{}{}, cv, scheme.Scheme, logf.Log, clientMock, httpMock)
		assert.Equal(v.ReturnError["init"], err)
		config, _ := utils.InitActionConfig(cli.New().RESTClientGetter(), []byte(""), logf.Log)
		err = testObj.Prepare(config)
		assert.Equal(v.ReturnError["prepare"], err)
		err = testObj.ManageSubResources()
		assert.Equal(v.ReturnError["subResources"], err)
		err = testObj.CreateOrUpdateSubCharts()
		assert.Equal(v.ReturnError["subCharts"], err)
	}
}
