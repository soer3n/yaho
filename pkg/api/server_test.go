package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerRun(t *testing.T) {

	assert := assert.New(t)
	assert.NotNil(New("9090"))
}
