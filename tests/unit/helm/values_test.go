package helm

import (
	"testing"

	"github.com/soer3n/yaho/internal/helm"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
)

func TestValues(t *testing.T) {
	assert := assert.New(t)

	for _, testcase := range testcases.GetTestValueSpecs() {
		vList := testcase.Input.([]*helm.ValuesRef)
		testObj := helm.NewValueTemplate(vList)
		_, err := testObj.ManageValues()

		assert.Equal(testcase.ReturnError, err)
	}
}
