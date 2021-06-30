package helm

import (
	"testing"

	"github.com/soer3n/apps-operator/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestSetEnv(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

	assert := assert.New(t)

	testObj := NewHelmClient(getTestFinalizerRelease(), &clientMock, &httpMock)
	testObj.Env = map[string]string{
		"KubeConfig":       "a",
		"KubeContext":      "b",
		"KubeAsUser":       "c",
		"KubeAsGroups":     "d",
		"KubeAPIServer":    "e",
		"RegistryConfig":   "f",
		"RepositoryConfig": "g",
		"RepositoryCache":  "h",
		"PluginsDirectory": "i",
		"KubeToken":        "j",
	}

	settings := testObj.GetEnvSettings()

	// assert.Equal(expected, charts, "Structs should be equal.")
	assert.NotNil(settings)
}
