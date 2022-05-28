package helm

import (
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/values"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestValues(t *testing.T) {
	assert := assert.New(t)
	clientMock, _ := helmmocks.GetValueMock()

	for _, testcase := range testcases.GetTestValueSpecs() {
		release := testcase.Input.(*helmv1alpha1.Release)
		testObj := values.New(release, logf.Log, clientMock)
		v, err := testObj.ManageValues()

		assert.Equal(testcase.ReturnError["manage"], err)
		assert.Equal(testcase.ReturnValue, v)
	}
}
