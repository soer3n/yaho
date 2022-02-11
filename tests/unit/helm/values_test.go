package helm

import (
	"testing"

	"github.com/soer3n/yaho/internal/values"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestValues(t *testing.T) {
	assert := assert.New(t)

	for _, testcase := range testcases.GetTestValueSpecs() {
		vList := testcase.Input.([]*values.ValuesRef)
		testObj := values.New(vList, logf.Log)
		_, err := testObj.ManageValues()

		assert.Equal(testcase.ReturnError, err)
	}
}
