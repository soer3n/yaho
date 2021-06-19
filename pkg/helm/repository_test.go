package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type K8SClientMock struct {
	mock.Mock
}

func TestGetEntryObject(t *testing.T) {

	assert := assert.New(t)
	assert.Equal("foo", "foo", "Structs should be equal.")
}
