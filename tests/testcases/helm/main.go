package helm

import (
	"os"

	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	"helm.sh/helm/v3/pkg/cli"
)

// GetTestMainEnvMaps returns testcases for environment settings
func GetTestMainEnvMaps() []inttypes.TestCase {
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
				RegistryConfig:   homeDir + "/.config/helm/registry/config.json",
				RepositoryConfig: homeDir + "/.config/helm/repositories.yaml",
				RepositoryCache:  homeDir + "/.cache/helm/repository",
				PluginsDirectory: homeDir + "/.local/share/helm/plugins",
				MaxHistory:       10,
			},
			ReturnError: nil,
		},
	}
}
