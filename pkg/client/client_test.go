package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetAPIResources(t *testing.T) {

	client := New()
	client.DynamicClient = &K8SClientMock{}

	assert := assert.New(t)

	_, _ = client.GetResource("", "", "", "", "", v1.GetOptions{})

	assert.NotNil(nil)
}

func TestListResources(t *testing.T) {

	assert := assert.New(t)
	assert.NotNil("gfo")
}

func TestGetResource(t *testing.T) {

	assert := assert.New(t)
	assert.NotNil("gfo")
}
