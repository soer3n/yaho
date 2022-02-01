package helm

import (
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/helm"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
)

func TestChartCreateConfigMaps(t *testing.T) {

	settings := cli.New()
	clientMock, httpMock := helmmocks.GetChartMock()

	assert := assert.New(t)

	for _, v := range testcases.GetTestRepoChartVersions() {
		ver := v.Input.([]*repo.ChartVersion)
		testObj := helm.NewChart(ver, settings, "test", clientMock, httpMock, kube.Client{})
		maps := testObj.CreateConfigMaps()
		assert.NotNil(maps)
	}
}

func TestChartAddOrUpdateMap(t *testing.T) {

	settings := cli.New()
	clientMock, httpMock := helmmocks.GetChartMock()

	assert := assert.New(t)

	for _, v := range testcases.GetTestHelmChartMaps() {
		for _, i := range testcases.GetTestRepoChartVersions() {
			ver := i.Input.([]*repo.ChartVersion)
			testObj := helm.NewChart(ver, settings, "test", clientMock, httpMock, kube.Client{})
			rel, _ := v.Input.(map[string]*helmv1alpha1.Chart)
			maps := testObj.AddOrUpdateChartMap(rel, testcases.GetTestChartRepo())
			expectedLen, _ := v.ReturnValue.(int)
			assert.Equal(len(maps), expectedLen)
		}
	}
}
