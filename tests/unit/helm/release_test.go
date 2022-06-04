package helm

import (
	"log"
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	"github.com/soer3n/yaho/internal/release"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/kubectl/pkg/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestReleaseUpdate(t *testing.T) {
	clientMock, httpMock := helmmocks.GetReleaseMock()
	apiObjList := testcases.GetTestReleaseSpecs()
	assert := assert.New(t)

	_ = helmv1alpha1.AddToScheme(scheme.Scheme)

	for _, apiObj := range apiObjList {

		current := apiObj.Input.(*helmv1alpha1.Release)
		testObj, err := release.New(current, current.Namespace, scheme.Scheme, logf.Log, clientMock, httpMock, cli.New().RESTClientGetter(), []byte(""))
		assert.Equal(apiObj.ReturnError["init"], err)

		testObj.Config = testcases.GetTestReleaseFakeActionConfig(t)

		if err := testObj.Config.Releases.Create(testcases.GetTestReleaseDeployedReleaseObj()); err != nil {
			log.Print(err)
		}

		err = testObj.Update()
		assert.Equal(apiObj.ReturnError["update"], err)

		err = testObj.RemoveRelease()
		assert.Equal(apiObj.ReturnError["remove"], err)
	}
}
