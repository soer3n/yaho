package helm

import (
	"encoding/json"
	actionlog "log"
	"os"
	"reflect"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/api/core/v1"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	client "github.com/soer3n/apps-operator/pkg/client"
)

func (hc *HelmRelease) Update() error {

	// repoChart := hc.Repo + "/" + hc.Chart
	// args := []string{hc.Name, repoChart}
	installConfig := hc.Config
	log.Infof("configinstall: %v", hc.Config)
	client := action.NewInstall(installConfig)
	// name, _, err := client.NameAndChart(args)
	//client.ReleaseName = name
	client.ReleaseName = hc.Name

	//if err != nil {
	//	return err
	//}

	options := &action.ChartPathOptions{
		Version:               hc.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}
	err, helmChart, chartPath := hc.GetChart(hc.Chart, options)

	if err != nil {
		return err
	}

	err = hc.checkDependencies(helmChart, chartPath, client)

	if err != nil {
		return err
	}

	log.Infof("configupdate: %v", hc.Config)
	release, err := hc.getRelease()

	//if err != nil {
	//	return err
	//}

	_ = hc.SetValues()

	// Check if something changed regarding the existing release
	if release != nil {
		ok, err := hc.valuesChanged()

		if err != nil {
			return err
		}

		if ok {
			return hc.upgrade(helmChart)
		}

		return nil
	}

	// helmChart.Values = vals

	// if err != nil {
	//	return err
	// }

	client.Namespace = hc.Settings.Namespace()
	// vals := hc.mergeMaps(helmChart.Values)
	vals := mergeMaps(hc.getValues(), helmChart.Values)
	release, err = client.Run(helmChart, vals)

	if err != nil {
		return err
	}

	log.Infof("Release (%q) successfully installed.", release.Name)
	return nil
}

func (hc *HelmRelease) Remove() error {
	client := action.NewUninstall(hc.Config)
	_, err := client.Run(hc.Name)
	return err
}

func (hc *HelmReleases) Remove() error {

	installedReleases, err := hc.getReleases()
	client := action.NewUninstall(hc.Config)

	if err != nil {
		return err
	}

	for key, release := range installedReleases {
		if !hc.shouldBeInstalled(release) {
			log.Infof("Removing release: index: (%q) name: (%q)", key, release.Name)
			// purge releases
			client.KeepHistory = false
			_, err := client.Run(release.Name)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (hc HelmRelease) getValues() map[string]interface{} {

	log.Infof("init check (%v)", hc.ValuesTemplate)

	vals := &values.Options{}
	initVals, _ := vals.MergeValues(getter.All(hc.Settings))

	if hc.ValuesTemplate == nil {
		return initVals
	}

	if hc.ValuesTemplate.ValueFiles != nil {
		vals.ValueFiles = hc.ValuesTemplate.ValueFiles
		log.Infof("first check (%q)", hc.ValuesTemplate.ValueFiles)
	}

	if hc.ValuesTemplate.ValuesMap != nil {
		vals.Values = hc.getValuesAsList(hc.ValuesTemplate.ValuesMap)
		log.Infof("second check (%q)", hc.ValuesTemplate.ValuesMap)
	}

	log.Info("third check")

	mergedVals, _ := vals.MergeValues(getter.All(hc.Settings))
	return mergedVals
}

func (hc *HelmRelease) SetValues() error {

	templateObj := hc.ValuesTemplate
	values := make(map[string]interface{})
	var err error

	if values, err = templateObj.ManageValues(); err != nil {
		return err
	}

	hc.Values = values
	hc.ValuesTemplate.ValuesMap = templateObj.ValuesMap

	return nil
}

func (hc HelmRelease) getValuesAsList(values map[string]string) []string {

	var valueList []string
	var transformedVal string
	valueList = []string{}
	for k, v := range values {
		transformedVal = k + "=" + v
		valueList = append(valueList, transformedVal)
	}

	return valueList
}

func (hc HelmRelease) getInstalledValues() (map[string]interface{}, error) {

	client := action.NewGetValues(hc.Config)
	return client.Run(hc.Name)
}

func (hc *HelmRelease) valuesChanged() (bool, error) {

	installedValues, err := hc.getInstalledValues()

	log.Infof("installed values: (%v)", installedValues)

	if err != nil {
		return false, err
	}

	requestedValues := hc.Values

	if err != nil {
		return false, err
	}

	log.Infof("values installed: (%v)", installedValues)
	log.Infof("values requested: (%v)", requestedValues)

	if len(requestedValues) < 1 && len(installedValues) < 1 {
		return false, nil
	}

	if reflect.DeepEqual(installedValues, requestedValues) {
		return false, nil
	}

	return true, nil
}

func (hc *HelmRelease) getRelease() (*release.Release, error) {
	log.Infof("config: %v", hc.Config)
	getConfig := hc.Config
	client := action.NewGet(getConfig)
	return client.Run(hc.Name)
}

func (hc *HelmRelease) GetChart(chartName string, chartPathOptions *action.ChartPathOptions) (error, *chart.Chart, string) {

	var jsonbody []byte
	var err error
	helmChart := &chart.Chart{
		Metadata:  &chart.Metadata{},
		Files:     []*chart.File{},
		Templates: []*chart.File{},
		Values:    make(map[string]interface{}),
	}
	chartObj := &helmv1alpha1.Chart{}
	files := []*chart.File{}
	args := make([]string, 0)
	namespace := "default"
	rc := client.New()
	args = []string{
		"charts.helm.soer3n.info",
		chartName,
	}

	obj := rc.GetResources(rc.Builder(namespace, true), args)
	if jsonbody, err = json.Marshal(obj.Data[1]); err != nil {
		return err, helmChart, "foo"
	}

	if err = json.Unmarshal(jsonbody, &chartObj); err != nil {
		return err, helmChart, "foo"
	}

	files = hc.getFiles(rc, chartObj)

	helmChart.Metadata.Name = chartName
	helmChart.Metadata.Version = hc.Version
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Files = files
	helmChart.Templates = hc.appendFilesFromConfigMap(rc, "helm-tmpl-"+hc.Chart+"-"+hc.Version, helmChart.Templates)
	helmChart.Values = hc.getDefaultValuesFromConfigMap(rc, "helm-default-"+hc.Chart+"-"+hc.Version)

	if err := helmChart.Validate(); err != nil {
		return err, helmChart, "foo"
	}

	return nil, helmChart, "foo"
}

func (hc *HelmRelease) getFiles(rc *client.Client, helmChart *helmv1alpha1.Chart) []*chart.File {

	files := []*chart.File{}

	files = hc.appendFilesFromConfigMap(rc, "helm-tmpl-"+hc.Chart+"-"+hc.Version, files)
	files = hc.appendFilesFromConfigMap(rc, "helm-crds-"+hc.Chart+"-"+hc.Version, files)

	return files
}

func (hc *HelmRelease) appendFilesFromConfigMap(rc *client.Client, name string, list []*chart.File) []*chart.File {

	args := []string{
		"configmaps",
		name,
	}

	var jsonbody []byte
	var err error

	configmap := &v1.ConfigMap{}
	files := []*chart.File{}

	obj := rc.GetResources(rc.Builder(hc.Namespace.Name, true), args)

	if jsonbody, err = json.Marshal(obj.Data[1]); err != nil {
		return files
	}

	if err = json.Unmarshal(jsonbody, &configmap); err != nil {
		return files
	}

	for key, data := range configmap.BinaryData {
		file := &chart.File{
			Name: key,
			Data: data,
		}
		files = append(files, file)
	}

	return files
}

func (hc *HelmRelease) getDefaultValuesFromConfigMap(rc *client.Client, name string) map[string]interface{} {

	values := make(map[string]interface{})
	args := []string{
		"configmaps",
		name,
	}

	var jsonbody []byte
	var err error

	configmap := &v1.ConfigMap{}

	obj := rc.GetResources(rc.Builder(hc.Namespace.Name, true), args)

	if jsonbody, err = json.Marshal(obj.Data[1]); err != nil {
		return values
	}

	if err = json.Unmarshal(jsonbody, &configmap); err != nil {
		return values
	}

	jsonMap := make(map[string]interface{})
	if err = json.Unmarshal([]byte(configmap.Data["values"]), &jsonMap); err != nil {
		panic(err)
	}

	return values
}

func (hc HelmRelease) configure() {

}

func (hc HelmRelease) validate() error {
	return nil
}

func (hc *HelmRelease) upgrade(helmChart *chart.Chart) error {
	client := action.NewUpgrade(hc.Config)

	// vals := hc.getValues()
	vals := mergeMaps(hc.getValues(), helmChart.Values)
	hc.Values = vals

	helmChart.Values = vals
	client.Namespace = hc.Settings.Namespace()
	rel, err := client.Run(hc.Name, helmChart, vals)

	if err != nil {
		return err
	}

	log.Infof("(%q) has been upgraded.", rel.Name)
	return nil
}

func (hc *HelmRelease) checkDependencies(ch *chart.Chart, cp string, client *action.Install) error {

	if req := ch.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(ch, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              os.Stdout,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(hc.Settings),
					RepositoryConfig: hc.Settings.RepositoryConfig,
					RepositoryCache:  hc.Settings.RepositoryCache,
					Debug:            hc.Settings.Debug,
				}

				if err := man.Update(); err != nil {
					return err
				}

				// Reload the chart with the updated Chart.lock file.
				if _, err = loader.Load(cp); err != nil {
					return errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {

				return err
			}
		}
	}

	return nil
}

func (hc HelmRelease) IsAlreadyInstalled() (error, bool) {
	return nil, false
}

func (hc HelmReleases) getCharts() (error, []*chart.Chart) {

	return nil, []*chart.Chart{}
}

func (hc *HelmReleases) shouldBeInstalled(release *release.Release) bool {

	for key, chart := range hc.Entries {

		if chart.Name == release.Name {
			log.Debugf("Release %v (index: %v) already installed.", chart.Name, key)
			return true
		}
	}

	return false
}

func (hc HelmRelease) GetActionConfig(settings *cli.EnvSettings) (*action.Configuration, error) {

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), actionlog.Printf)

	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err != nil {
		log.Infof("%+v", err)
		return actionConfig, err
	}

	return actionConfig, nil
}

func (hc *HelmReleases) getRelease(name string) (*release.Release, error) {
	client := action.NewGet(hc.Config)
	return client.Run(name)
}

func (hc HelmReleases) getReleases() ([]*release.Release, error) {

	// Init cmd
	client := action.NewList(hc.Config)

	// Only list deployed
	client.Deployed = true

	// Run cmd
	return client.Run()
}
