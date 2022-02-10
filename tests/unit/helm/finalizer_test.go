package helm

import (
	"log"
	"testing"

	"github.com/soer3n/yaho/internal/helm"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"

	"helm.sh/helm/v3/pkg/kube"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestFinalizerHandleRelease(t *testing.T) {
	clientMock, httpMock := helmmocks.GetFinalizerMock()
	assert := assert.New(t)

	settings := helm.GetEnvSettings(map[string]string{})
	testObj := helm.NewHelmRelease(testcases.GetTestFinalizerRelease(), settings, logf.Log, clientMock, httpMock, kube.Client{})
	testObj.Config = testcases.GetTestFinalizerFakeActionConfig(t)

	if err := testObj.Config.Releases.Create(testcases.GetTestFinalizerDeployedReleaseObj()); err != nil {
		log.
			Print(err)
	}

	for _, v := range testcases.GetTestFinalizerSpecsRelease() {

		ok, err := helm.HandleFinalizer(testObj)
		assert.Equal(v.ReturnValue, ok)
		assert.Equal(v.ReturnError, err)
	}
}

func TestFinalizerHandleRepo(t *testing.T) {
	// clientMock, httpMock := helmmocks.GetFinalizerMock()
	assert := assert.New(t)

	for _, v := range testcases.GetTestFinalizerSpecsRepo() {

		ok, err := helm.HandleFinalizer(v.Input)
		assert.Equal(v.ReturnValue, ok)
		assert.Equal(v.ReturnError, err)
	}
}
