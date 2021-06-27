package helm

import (
	"encoding/json"
	actionlog "log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prometheus/common/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	client "github.com/soer3n/apps-operator/pkg/client"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
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

func getChartByURL(url string, g client.HTTPClientInterface) (*chart.Chart, error) {

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

func getChartURL(rc client.ClientInterface, chart, version, namespace string) (string, error) {

	var jsonbody []byte
	var err error

	chartObj := &helmv1alpha1.Chart{}
	foo := &helmv1alpha1.ChartList{}

	if jsonbody, err = rc.ListResources("", "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{}); err != nil {
		return "", err
	}

	if err = json.Unmarshal(jsonbody, &foo); err != nil {
		return "", err
	}

	log.Info("Charts....")

	for _, bar := range foo.Items {
		log.Info(bar)
	}

	if jsonbody, err = rc.ListResources(namespace, "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{}); err != nil {
		return "", err
	}

	if err = json.Unmarshal(jsonbody, &foo); err != nil {
		return "", err
	}

	log.Info("Charts....")

	for _, bar := range foo.Items {
		log.Info(bar)
	}

	if jsonbody, err = rc.GetResource(chart, namespace, "charts", "helm.soer3n.info", "v1alpha1", metav1.GetOptions{}); err != nil {
		return "", err
	}

	if err = json.Unmarshal(jsonbody, &chartObj); err != nil {
		return "", err
	}

	return chartObj.GetChartVersion(version).URL, nil
}

func removeFile(path, name string) error {
	idx := filepath.Join(path, name)
	return removeFileByPath(idx)
}

func removeFileByFulPath(fullpath string) error {
	return removeFileByPath(fullpath)
}

func removeFileByPath(idx string) error {
	if _, err := os.Stat(idx); err == nil {
		os.Remove(idx)
	}

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
