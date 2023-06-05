package utils

import (
	"errors"
	"fmt"
	actionlog "log"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func ManagerOptions(config string) (*manager.Options, error) {

	c, err := parseOperatorConfig(config)

	if err != nil {
		return nil, err
	}

	return &manager.Options{
		HealthProbeBindAddress: c.HealthProbeBindAddress,
		LeaderElection:         c.LeaderElection.Enabled,
		LeaderElectionID:       c.LeaderElection.ResourceID,
		MetricsBindAddress:     c.MetricsBindAddress,
	}, nil
}

func parseOperatorConfig(path string) (*Config, error) {
	fd, err := os.Open(filepath.Clean(filepath.Join(path)))
	if err != nil {
		return nil, fmt.Errorf("could not open the configuration file: %v", err)
	}
	defer fd.Close()

	cfg := Config{}

	if err = yaml.NewDecoder(fd).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("could not decode configuration file: %v", err)
	}

	return &cfg, nil
}

// InitActionConfig represents the initialization of an helm configuration
func InitActionConfig(getter genericclioptions.RESTClientGetter, kubeconfig []byte, logger logr.Logger) (*action.Configuration, error) {
	/*
		/ we cannot use helm init func here due to data race issues on concurrent execution (helm's kube client tries to update the namespace field on each initialization)

		// actionConfig := new(action.Configuration)
		err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), actionlog.Printf)
	*/

	if getter == nil {
		logger.Info("getter is nil")
		return nil, errors.New("getter is nil")
	}

	f := cmdutil.NewFactory(getter)
	set, err := f.KubernetesClientSet()

	if err != nil {
		fmt.Print(err.Error())
		return nil, err
	}

	casted, ok := getter.(*HelmRESTClientGetter)
	namespace := "default"

	if ok {
		namespace = casted.ReleaseNamespace
	}

	c := &kube.Client{
		Factory:   f,
		Log:       actionlog.Printf,
		Namespace: namespace,
	}

	conf := &action.Configuration{
		RESTClientGetter: getter,
		KubeClient:       c,
		Log:              actionlog.Printf,
		Releases:         storage.Init(driver.NewSecrets(set.CoreV1().Secrets(namespace))),
	}

	return conf, nil
}

// MergeMaps returns distinct map of two as input
// have to be called as a goroutine to avoid memory leaks
func MergeMaps(source, dest map[string]interface{}) map[string]interface{} {
	if source == nil || dest == nil {
		return dest
	}

	copy := make(map[string]interface{})

	for k, v := range dest {
		copy[k] = v
	}

	for k, v := range source {
		// when key already exists we have to compare also sub values
		if temp, ok := v.(map[string]interface{}); ok {
			merge, _ := copy[k].(map[string]interface{})
			copy[k] = MergeMaps(merge, temp)
			continue
		}

		copy[k] = v
	}

	return copy
}

// CopyUntypedMap return a copy of a map with strings as keys and empty interface as value
func CopyUntypedMap(source map[string]interface{}) map[string]interface{} {

	vals := make(map[string]interface{})

	for k, v := range source {
		vals[k] = v
	}

	return vals
}

// MergeUntypedMaps returns distinct map of two as input
func MergeUntypedMaps(dest, source map[string]interface{}, keys ...string) map[string]interface{} {

	trimedKeys := []string{}
	copy := make(map[string]interface{})

	for k, v := range dest {
		copy[k] = v
	}

	for _, v := range keys {
		if v == "" {
			continue
		}
		trimedKeys = append(trimedKeys, v)
	}

	for l, k := range trimedKeys {
		if l == 0 {
			_, ok := copy[k].(map[string]interface{})

			if !ok {
				copy[k] = make(map[string]interface{})
			}

			if len(trimedKeys) == 1 {
				helper := copy[k].(map[string]interface{})
				for kv, v := range source {
					helper[kv] = v
				}
				copy[k] = helper
			}

			continue
		} else {
			if l > 1 {
				break
			}
			if _, ok := copy[k].(map[string]interface{}); ok {
				helper := copy[trimedKeys[0]].(map[string]interface{})
				if l == len(trimedKeys)-1 {
					subHelper, ok := helper[k].(map[string]interface{})

					if ok {
						for sk, sv := range source {
							subHelper[sk] = sv
						}
						helper[k] = subHelper
					} else {
						helper[k] = source
					}
					copy[trimedKeys[0]] = helper
				} else {
					sub := MergeUntypedMaps(helper, source, trimedKeys[2:]...)
					helper[k] = sub
					copy[trimedKeys[0]] = helper
				}
			}
		}
	}

	if len(trimedKeys) == 0 {
		for k, v := range source {
			copy[k] = v
		}
	}

	return copy
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
