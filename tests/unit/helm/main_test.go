package helm

import (
	"testing"

	"github.com/soer3n/yaho/internal/helm"
	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
)

func TestSetEnv(t *testing.T) {

	clientMock := unstructuredmocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	assert := assert.New(t)

	for _, v := range testcases.GetTestMainObjSpecs() {

		testObj := helm.NewHelmClient(v, &clientMock, &httpMock)

		for _, e := range testcases.GetTestMainEnvMaps() {
			testObj.Env, _ = e.Input.(map[string]string)

			settings := testObj.GetEnvSettings()
			val, _ := e.ReturnValue.(cli.EnvSettings)

			// whole struct cannot be checked due to unexported fields
			assert.Equal(val.Debug, settings.Debug)
			assert.Equal(val.KubeAPIServer, settings.KubeAPIServer)
			assert.Equal(val.KubeConfig, settings.KubeConfig)
			assert.Equal(val.KubeContext, settings.KubeContext)
			assert.Equal(val.KubeAsGroups, settings.KubeAsGroups)
			assert.Equal(val.KubeAsUser, settings.KubeAsUser)
			assert.Equal(val.MaxHistory, settings.MaxHistory)
			assert.Equal(val.RepositoryCache, settings.RepositoryCache)
			assert.Equal(val.RepositoryConfig, settings.RepositoryConfig)
			assert.Equal(val.PluginsDirectory, settings.PluginsDirectory)
			assert.Equal(val.RegistryConfig, settings.RegistryConfig)
			assert.Equal(val.KubeToken, settings.KubeToken)
		}
	}
}
