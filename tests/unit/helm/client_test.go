package helm

import (
	"testing"

	"github.com/soer3n/yaho/internal/helm"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	clientMock, httpMock := helmmocks.GetClientMock()
	assert := assert.New(t)

	testObj := helm.NewHelmClient(testcases.GetTestClientRepo(), clientMock, httpMock)
	repoObj := testObj.GetRepo("")

	assert.Nil(repoObj)

	testObj = helm.NewHelmClient(testcases.GetTestClientRelease(), clientMock, httpMock)
	releaseObj := testObj.GetRelease("", "")

	assert.Nil(releaseObj)
}
