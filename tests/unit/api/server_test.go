package api

import (
	"testing"

	"github.com/soer3n/yaho/internal/api"
	"github.com/soer3n/yaho/internal/client"
	"github.com/stretchr/testify/assert"
)

func TestServerRun(t *testing.T) {
	k8sclient := &client.Client{}
	assert := assert.New(t)
	assert.NotNil(api.New("9090", k8sclient))
}
