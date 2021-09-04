package helm

import (
	"os"
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/mocks"
	inttypes "github.com/soer3n/yaho/internal/types"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetEnv(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	assert := assert.New(t)

	for _, v := range getTestMainObjSpecs() {

		testObj := NewHelmClient(v, &clientMock, &httpMock)

		for _, e := range getTestMainEnvMaps() {
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

func getTestMainEnvMaps() []inttypes.TestCase {
	homeDir, _ := os.UserHomeDir()
	return []inttypes.TestCase{
		{
			Input: map[string]string{
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
			},
			ReturnValue: cli.EnvSettings{
				KubeConfig:       "a",
				KubeContext:      "b",
				KubeToken:        "j",
				KubeAsUser:       "c",
				KubeAsGroups:     []string{"d"},
				KubeAPIServer:    "e",
				Debug:            false,
				RegistryConfig:   "f",
				RepositoryConfig: "g",
				RepositoryCache:  "h",
				PluginsDirectory: "i",
				MaxHistory:       10,
			},
			ReturnError: nil,
		},
		{
			Input: nil,
			ReturnValue: cli.EnvSettings{
				KubeConfig:       "",
				KubeContext:      "",
				KubeToken:        "",
				KubeAsUser:       "",
				KubeAsGroups:     nil,
				KubeAPIServer:    "",
				Debug:            false,
				RegistryConfig:   homeDir + "/.config/helm/registry.json",
				RepositoryConfig: homeDir + "/.config/helm/repositories.yaml",
				RepositoryCache:  homeDir + "/.cache/helm/repository",
				PluginsDirectory: homeDir + "/.local/share/helm/plugins",
				MaxHistory:       10,
			},
			ReturnError: nil,
		},
	}
}

func getTestMainObjSpecs() []interface{} {
	return []interface{}{
		&helmv1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "release",
				Namespace: "",
			},
			Spec: helmv1alpha1.ReleaseSpec{
				Name:  "release",
				Repo:  "repo",
				Chart: "chart",
			},
		},
	}
}
