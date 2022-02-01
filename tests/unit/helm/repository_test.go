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
)

func TestRepoGetCharts(t *testing.T) {

	clientMock, httpMock := helmmocks.GetRepoMock()
	settings := cli.New()
	apiObjList := testcases.GetTestRepoSpecs()

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		val := apiObj.Input.(helmv1alpha1.Repo)
		testObj := helm.NewHelmRepo(&val, settings, clientMock, httpMock, kube.Client{})
		selectors := make(map[string]string)

		// parse selectors string from api object meta data
		for k, v := range val.ObjectMeta.Labels {
			selectors[k] = v
		}

		_, err := testObj.GetCharts(settings, selectors)
		assert.Equal(err, apiObj.ReturnError)
	}
}
