package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFilterSpec(t *testing.T) {
	options := make(map[string]string)
	fitlerOptions := NewOptions(options)

	valuesObj := []*ValuesRef{
		{
			Parent: "parent",
		},
	}

	expectedReturnStruct := []*ValuesRef{
		{
			Parent: "parent",
		},
	}

	returnStruct := fitlerOptions.Filter(valuesObj)

	assert := assert.New(t)
	assert.Equal(expectedReturnStruct, returnStruct, "Structs should be equal.")
}
