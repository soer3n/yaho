package helm

import (
	"context"
	"testing"

	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"github.com/soer3n/yaho/internal/repository"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/client-go/kubernetes/scheme"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestRepoUpdate(t *testing.T) {
	clientMock, httpMock := helmmocks.GetRepoMock()
	settings := cli.New()
	apiObjList := testcases.GetTestRepoSpecs()

	assert := assert.New(t)

	_ = yahov1alpha2.AddToScheme(scheme.Scheme)

	for _, apiObj := range apiObjList {

		val := apiObj.Input.(yahov1alpha2.Repository)
		r := &val
		testObj := repository.New(r, val.Namespace, context.TODO(), settings, logf.Log, clientMock, httpMock, kube.Client{})
		// assert.Equal(err, apiObj.ReturnError["init"])
		selectors := make(map[string]string)

		// parse selectors string from api object meta data
		for k, v := range val.ObjectMeta.Labels {
			selectors[k] = v
		}

		err := testObj.Update(r, scheme.Scheme)
		assert.Equal(err, apiObj.ReturnError["update"])
	}
}
