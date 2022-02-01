package client

import (
	"testing"

	"github.com/soer3n/yaho/internal/client"
	clientmocks "github.com/soer3n/yaho/tests/mocks/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetAPIResources(t *testing.T) {

	client := &client.Client{}
	client.DiscoverClient = clientmocks.GetClientDiscoveryMock()

	assert := assert.New(t)

	_, err := client.GetAPIResources("apiGroup", false)

	assert.Nil(err)
}

func TestListResources(t *testing.T) {

	client := &client.Client{}
	client.DynamicClient = clientmocks.GetClientDynamicMock()

	assert := assert.New(t)

	_, err := client.ListResources("namespace", "resource", "group", "v1", metav1.ListOptions{})

	assert.Nil(err)
}

func TestGetResource(t *testing.T) {

	client := &client.Client{}
	client.DynamicClient = clientmocks.GetClientDynamicMock()

	assert := assert.New(t)

	_, err := client.GetResource("name", "namespace", "resource", "group", "v1", metav1.GetOptions{})

	assert.Nil(err)
}
