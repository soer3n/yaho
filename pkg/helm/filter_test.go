package helm

import (
	"testing"

	inttypes "github.com/soer3n/yaho/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestFilterFilter(t *testing.T) {
	options := make(map[string]string)
	fitlerOptions := NewOptions(options)
	assert := assert.New(t)

	for _, v := range getTestFilterSpecs() {
		i, _ := v.Input.([]*ValuesRef)
		returnStruct := fitlerOptions.Filter(i)
		r, _ := v.ReturnValue.([]*ValuesRef)
		assert.Equal(r, returnStruct, "Structs should be equal.")
	}
}

func getTestFilterSpecs() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnError: nil,
			ReturnValue: []*ValuesRef{
				{
					Parent: "parent",
				},
			},
			Input: []*ValuesRef{
				{
					Parent: "parent",
				},
			},
		},
	}
}
