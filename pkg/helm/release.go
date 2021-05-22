package helm

import (
	"encoding/json"
	"io"
	actionlog "log"
	"net/http"
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
	"helm.sh/helm/v3/pkg/repo"
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
	hc.Client = client

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

	repoSelector := "repo=" + hc.Repo

	if _, ok := chartObj.ObjectMeta.Labels["repoGroup"]; ok {
		repoSelector = "repoGroup=" + chartObj.ObjectMeta.Labels["repoGroup"]
	}

	files = hc.getFiles(rc, chartObj)

	helmChart.Metadata.Name = chartName
	helmChart.Metadata.Version = hc.Version
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Files = files
	helmChart.Templates = hc.appendFilesFromConfigMap(rc, "helm-tmpl-"+hc.Chart+"-"+hc.Version, helmChart.Templates)
	helmChart.Values = hc.getDefaultValuesFromConfigMap(rc, "helm-default-"+hc.Chart+"-"+hc.Version)

	versionObj := chartObj.GetChartVersion(hc.Version)

	if err := hc.addDependencies(rc, helmChart, versionObj.Dependencies, repoSelector); err != nil {
		return err, helmChart, "foo"
	}

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

func (hc *HelmRelease) addDependencies(rc *client.Client, chart *chart.Chart, deps []helmv1alpha1.ChartDep, selector string) error {
	args := []string{
		"charts.helm.soer3n.info",
	}

	var jsonbody []byte
	var err error

	charts := []helmv1alpha1.Chart{}
	obj := rc.GetResources(rc.Builder(hc.Namespace.Name, true).LabelSelector(selector), args)

	if jsonbody, err = json.Marshal(obj.Data[1]); err != nil {
		return err
	}

	if err = json.Unmarshal(jsonbody, &charts); err != nil {
		return err
	}

	for _, item := range charts {
		options := &action.ChartPathOptions{
			RepoURL: item.Spec.Sources[0],
			Version: item.Spec.APIVersion,
		}
		_, foo, _ := hc.GetChart(item.Spec.Name, options)
		chart.AddDependency(foo)
	}

	return nil
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
		if name == "helm-crds-"+hc.Chart+"-"+hc.Version {
			key = "crds/" + key
		}

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

	return jsonMap
}

func (hc *HelmRelease) getRepo(rc *client.Client, repo string) (error, helmv1alpha1.Repo) {

	args := []string{
		"repos",
		repo,
	}

	var jsonbody []byte
	var err error

	repoObj := &helmv1alpha1.Repo{}

	obj := rc.GetResources(rc.Builder(hc.Namespace.Name, true), args)

	if jsonbody, err = json.Marshal(obj.Data[1]); err != nil {
		return err, *repoObj
	}

	if err = json.Unmarshal(jsonbody, &repoObj); err != nil {
		return err, *repoObj
	}

	return nil, *repoObj
}

func (hc HelmRelease) GetParsedConfigMaps() []v1.ConfigMap {

	configmapList := []v1.ConfigMap{}

	//if err = chart.CreateTemplates(); err != nil {
	//	return err, configmapList
	//}

	//return nil, chart.CreateConfigMaps()

	installConfig := hc.Config
	log.Infof("configinstall: %v", hc.Config)
	releaseClient := action.NewInstall(installConfig)
	releaseClient.ReleaseName = hc.Name
	hc.Client = releaseClient

	var argsList []string
	var cp string
	var chartRequested *chart.Chart
	chartVersion := &HelmChartVersion{}
	var err error

	//settings := hc.Settings

	rc := client.New()

	argsList = make([]string, 0)
	argsList = append(argsList, hc.Chart)
	argsList = append(argsList, hc.Repo+"/"+hc.Chart)

	_, repoObj := hc.getRepo(rc, hc.Repo)

	releaseClient.ReleaseName = hc.Name
	releaseClient.Version = hc.Version
	releaseClient.ChartPathOptions.RepoURL = repoObj.Spec.Url

	//if cp, err = releaseClient.ChartPathOptions.LocateChart(hc.Chart, settings); err != nil {
	if cp, err = hc.DownloadTo(repoObj.Spec.Url, hc.Version, &releaseClient.ChartPathOptions); err != nil {
		actionlog.Printf("err: %v", err)
		return configmapList
	}

	if chartRequested, err = loader.Load(cp); err != nil {
		return configmapList
	}

	chartVersion.Version = &repo.ChartVersion{
		Metadata: &chart.Metadata{
			Name:    hc.Chart,
			Version: hc.Version,
		},
	}

	chartVersion.Templates = chartRequested.Templates
	chartVersion.CRDs = chartRequested.CRDs()
	chartVersion.DefaultValues = chartRequested.Values

	if err := removeFileByFulPath(cp); err != nil {
		return configmapList
	}

	return chartVersion.createConfigMaps(hc.Namespace.Name)
}

func (hc HelmRelease) DownloadTo(url, version string, options *action.ChartPathOptions) (string, error) {
	fullUrl := url + "/" + hc.Name + "-" + version + ".tgz"
	fileName := hc.Settings.RepositoryCache + "/" + hc.Name + "-" + version + ".tgz"
	var file *os.File
	var resp *http.Response
	var err error
	var size int64

	if file, err = os.Create(fileName); err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	// Put content on file
	if resp, err = client.Get(fullUrl); err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if size, err = io.Copy(file, resp.Body); err != nil {
		return fileName, err
	}

	defer file.Close()

	actionlog.Printf("Downloaded a file %s with size %d", fileName, size)

	return fileName, nil
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
