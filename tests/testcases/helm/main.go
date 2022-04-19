package helm

import (
	"os"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// GetTestMainObjSpecs returns cr managed by client
func GetTestMainObjSpecs() []interface{} {
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
