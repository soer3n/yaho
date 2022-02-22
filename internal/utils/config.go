package utils

import (
	actionlog "log"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// InitActionConfig represents the initialization of an helm configuration
func InitActionConfig(settings *cli.EnvSettings, c kube.Client) (*action.Configuration, error) {
	/*
		/ we cannot use helm init func here due to data race issues on concurrent execution (helm's kube client tries to update the namespace field on each initialization)

		// actionConfig := new(action.Configuration)
		// err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), actionlog.Printf)
	*/

	getter := settings.RESTClientGetter()
	set, _ := cmdutil.NewFactory(getter).KubernetesClientSet()
	conf := &action.Configuration{
		RESTClientGetter: getter,
		KubeClient:       &c,
		Log:              actionlog.Printf,
		Releases:         storage.Init(driver.NewSecrets(set.CoreV1().Secrets(settings.Namespace()))),
	}

	return conf, nil
}

// MergeMaps returns distinct map of two as input
// have to be called as a goroutine to avoid memory leaks
func MergeMaps(source, dest map[string]interface{}) map[string]interface{} {
	if source == nil || dest == nil {
		return dest
	}

	for k, v := range source {
		// when key already exists we have to compare also sub values
		if temp, ok := v.(map[string]interface{}); ok {
			merge, _ := dest[k].(map[string]interface{})
			dest[k] = MergeMaps(merge, temp)
			continue
		}

		dest[k] = v
	}

	return dest
}

// MergeUntypedMaps returns distinct map of two as input
// have to be called as a goroutine to avoid memory leaks
func MergeUntypedMaps(dest, source map[string]interface{}, key string) map[string]interface{} {
	for k, v := range source {
		if key == "" {
			if temp, ok := dest[k].(map[string]interface{}); ok {
				temp = MergeUntypedMaps(temp, map[string]interface{}{
					k: v,
				}, key)
				dest[k] = temp
				continue
			}

			dest[k] = v
			continue
		}

		if dest == nil {
			dest = make(map[string]interface{})
		}

		sub, ok := dest[key].(map[string]interface{})

		if !ok {
			dest[key] = make(map[string]interface{})
			sub = make(map[string]interface{})
		}

		sub[k] = v
		dest[key] = sub
	}

	return dest
}

// GetEnvSettings represents func for returning helm cli settings which are needed for helm actions
func GetEnvSettings(env map[string]string) *cli.EnvSettings {
	settings := cli.New()

	if env == nil {
		return settings
	}

	// overwrite default settings if requested
	for k, v := range env {
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
