package helm

import (
	"log"
	"testing"

	"github.com/soer3n/yaho/internal/helm"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
)

func TestFinalizerHandleRelease(t *testing.T) {

	clientMock, httpMock := helmmocks.GetFinalizerMock()
	assert := assert.New(t)

	testObj := helm.NewHelmClient(testcases.GetTestFinalizerRelease(), clientMock, httpMock)
	testObj.Releases.Entries[0].Config = testcases.GetTestFinalizerFakeActionConfig(t)

	if err := testObj.Releases.Entries[0].Config.Releases.Create(testcases.GetTestFinalizerDeployedReleaseObj()); err != nil {
		log.
			Print(err)
	}

	for _, v := range testcases.GetTestFinalizerSpecsRelease() {

		ok, err := helm.HandleFinalizer(testObj, v)
		assert.Equal(v.ReturnValue, ok)
		assert.Equal(v.ReturnError, err)
	}

}

func TestFinalizerHandleRepo(t *testing.T) {

	clientMock, httpMock := helmmocks.GetFinalizerMock()
	assert := assert.New(t)

	testObj := helm.NewHelmClient(testcases.GetTestFinalizerRepo(), clientMock, httpMock)

	for _, v := range testcases.GetTestFinalizerSpecsRepo() {

		ok, err := helm.HandleFinalizer(testObj, v)
		assert.Equal(v.ReturnValue, ok)
		assert.Equal(v.ReturnError, err)
	}

}
