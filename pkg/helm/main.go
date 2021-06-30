package helm

import (
	"context"
	actionlog "log"
	"net/http"
	"os"

	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/types"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/apps-operator/internal/types"
	"github.com/soer3n/apps-operator/pkg/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func initActionConfig(settings *cli.EnvSettings) (*action.Configuration, error) {

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), actionlog.Printf)

	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err != nil {
		log.Debugf("%+v", err)
		return actionConfig, err
	}

	return actionConfig, nil
}

func getChartByURL(url string, g inttypes.HTTPClientInterface) (*chart.Chart, error) {

	var resp *http.Response
	var err error

	// Put content to buffer
	log.Infof("url: %v", url)

	if resp, err = g.Get(url); err != nil {
		log.Fatal(err)
	}

	log.Infof("%v", url)

	return loader.LoadArchive(resp.Body)
}

func getChartURL(rc client.Client, chart, version, namespace string) (string, error) {

	var err error

	chartObj := &helmv1alpha1.Chart{}

	if err = rc.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: chart}, chartObj); err != nil {
		return "", err
	}

	return utils.GetChartVersion(version, chartObj).URL, nil
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
