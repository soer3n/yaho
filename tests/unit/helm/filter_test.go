package helm

import (
	"testing"

	"github.com/soer3n/yaho/internal/values"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
)

func TestFilterFilter(t *testing.T) {
	options := make(map[string]string)
	fitlerOptions := values.NewOptions(options)
	assert := assert.New(t)

	for _, v := range testcases.GetTestFilterSpecs() {
		i, _ := v.Input.([]*values.ValuesRef)
		returnStruct := fitlerOptions.Filter(i)
		r, _ := v.ReturnValue.([]*values.ValuesRef)
		assert.Equal(r, returnStruct, "Structs should be equal.")
	}
}
