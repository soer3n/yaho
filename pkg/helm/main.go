package helm

import (
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func initActionConfig(settings *cli.EnvSettings) (*action.Configuration, error) {

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf)

	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err != nil {
		log.Printf("%+v", err)
		return actionConfig, err
	}

	return actionConfig, nil
}

func removeFile(path, name string) error {
	idx := filepath.Join(path, name)

	if _, err := os.Stat(idx); err == nil {
		os.Remove(idx)
	}

	idx = filepath.Join(path, name)

	if _, err := os.Stat(idx); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "can't remove file %s", idx)
	}

	return os.Remove(idx)
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {

	if a == nil || b == nil {
		return b
	}

	for k, v := range a {
		b[k] = v
	}
	return b
}

func (c HelmClient) GetEnvSettings() *cli.EnvSettings {
	settings := cli.New()

	if c.Env == nil {
		return settings
	}

	//overwrite default settings if requested
	for k, v := range c.Env {
		switch k {
		case "KubeConfig":
			settings.KubeConfig = v
		case "KubeContext":
			settings.KubeContext = v
		case "KubeToken":
			settings.KubeToken = v
		case "KubeAsUser":
			settings.KubeAsUser = v
		case "KubeAsGroups":
			settings.KubeAsGroups = []string{v}
		case "KubeAPIServer":
			settings.KubeAPIServer = v
		// case "KubeCaFile":
		//	settings.KubeCaFile = v
		// case "Debug":
		//	settings.Debug = v
		case "RegistryConfig":
			settings.RegistryConfig = v
		case "RepositoryConfig":
			settings.RepositoryConfig = v
		case "RepositoryCache":
			settings.RepositoryCache = v
		case "PluginsDirectory":
			settings.PluginsDirectory = v
			// case "MaxHistory":
			//	settings.MaxHistory = v
		}
	}

	return settings
}
