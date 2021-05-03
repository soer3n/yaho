package main

import (
	"testing"

	helmv1 "github.com/soer3n/apps-operator/pkg/helm"
	"github.com/stretchr/testify/assert"
)

func TestGetFilterSpec(t *testing.T) {
	options := make(map[string]string)
	fitlerOptions := helmv1.NewOptions(options)

	valuesObj := []*helmv1.ValuesRef{}

	expectedReturnStruct := []*helmv1.ValuesRef{}

	returnStruct := fitlerOptions.Filter(valuesObj)

	assert := assert.New(t)
	assert.Equal(expectedReturnStruct, returnStruct, "Structs shoudl be equal.")
}
