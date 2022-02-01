package helm

import (
	"context"
	actionlog "log"
	"net/http"

	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/types"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/internal/types"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VALUES_MAP_SIZE represents limit for keys on first level in values
const valuesMapSize = 100

func initActionConfig(settings *cli.EnvSettings, c kube.Client) (*action.Configuration, error) {

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

func getChartByURL(url string, opts *Auth, g inttypes.HTTPClientInterface) (*chart.Chart, error) {

	var resp *http.Response
	var err error

	// Put content to buffer
	log.Infof("url: %v", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return &chart.Chart{}, err
	}

	if opts != nil {
		if opts.User != "" && opts.Password != "" {
			req.SetBasicAuth(opts.User, opts.Password)
		}
	}

	if resp, err = g.Do(req); err != nil {
		return &chart.Chart{}, err
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

func mergeValues(specValues map[string]interface{}, helmChart *chart.Chart) map[string]interface{} {
	// parsing values; goroutines are nessecarry due to tail recursion in called funcs
	// init buffered channel for coalesce values
	c := make(chan map[string]interface{}, 1)

	// run coalesce values in separate goroutine to avoid memory leak in main goroutine
	go func(c chan map[string]interface{}, specValues map[string]interface{}, helmChart *chart.Chart) {
		cv, _ := chartutil.CoalesceValues(helmChart, specValues)
		c <- cv
	}(c, specValues, helmChart)

	return <-c
}

// have to be called as a goroutine to avoid memory leaks
func mergeMaps(source, dest map[string]interface{}) map[string]interface{} {

	if source == nil || dest == nil {
		return dest
	}

	for k, v := range source {
		// when key already exists we have to compare also sub values
		if temp, ok := v.(map[string]interface{}); ok {
			merge, _ := dest[k].(map[string]interface{})
			dest[k] = mergeMaps(merge, temp)
			continue
		}

		dest[k] = v
	}

	return dest
}

// have to be called as a goroutine to avoid memory leaks
func mergeUntypedMaps(dest, source map[string]interface{}, key string) map[string]interface{} {

	for k, v := range source {
		if key == "" {
			if temp, ok := dest[k].(map[string]interface{}); ok {
				temp = mergeUntypedMaps(temp, map[string]interface{}{
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
func (c Client) GetEnvSettings() *cli.EnvSettings {
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
