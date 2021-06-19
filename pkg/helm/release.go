package helm

import (
	"encoding/json"
	actionlog "log"
	"reflect"

	"github.com/prometheus/common/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	client "github.com/soer3n/apps-operator/pkg/client"
)

func NewHelmRelease(instance *helmv1alpha1.Release, settings *cli.EnvSettings, actionconfig *action.Configuration) *HelmRelease {

	var helmRelease *HelmRelease

	log.Debugf("Trying HelmRelease %v", instance.Spec.Name)

	helmRelease = &HelmRelease{
		Name:     instance.Spec.Name,
		Repo:     instance.Spec.Repo,
		Chart:    instance.Spec.Chart,
		Settings: settings,
	}

	helmRelease.Config = actionconfig

	log.Debugf("HelmRelease config path: %v", helmRelease.Settings.RepositoryCache)

	if instance.Spec.ValuesTemplate != nil {
		if instance.Spec.ValuesTemplate.ValueRefs != nil {
			helmRelease.ValuesTemplate = &HelmValueTemplate{
				valuesRef: []*ValuesRef{},
			}
		}
	}

	return helmRelease
}

func (hc *HelmRelease) Update() error {

	log.Debugf("configinstall: %v", hc.Config)

	var release *release.Release
	var helmChart *chart.Chart
	var err error
	var ok bool

	installConfig := hc.Config
	client := action.NewInstall(installConfig)
	client.ReleaseName = hc.Name
	hc.Client = client

	options := &action.ChartPathOptions{
		Version:               hc.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}
	if helmChart, err = hc.getChart(hc.Chart, options); err != nil {
		return err
	}

	log.Debugf("configupdate: %v", hc.Config)
	release, err = hc.getRelease()

	if err = hc.setValues(); err != nil {
		return err
	}

	client.Namespace = hc.Settings.Namespace()
	vals := mergeMaps(hc.getValues(), helmChart.Values)

	// Check if something changed regarding the existing release
	if release != nil {
		if ok, err = hc.valuesChanged(); err != nil {
			return err
		}

		if ok {
			return hc.upgrade(helmChart, vals)
		}

		return nil
	}

	if err = chartutil.ProcessDependencies(helmChart, vals); err != nil {
		return err
	}

	if release, err = client.Run(helmChart, vals); err != nil {
		return err
	}

	log.Debugf("Release (%q) successfully installed.", release.Name)
	return nil
}

func (hc HelmRelease) Remove() error {
	client := action.NewUninstall(hc.Config)
	_, err := client.Run(hc.Name)
	return err
}

func (hc HelmRelease) getValues() map[string]interface{} {

	log.Debugf("init check (%v)", hc.ValuesTemplate)

	var initVals, mergedVals map[string]interface{}
	var err error

	vals := &values.Options{}
	if initVals, err = vals.MergeValues(getter.All(hc.Settings)); err != nil {
		return map[string]interface{}{}
	}

	if hc.ValuesTemplate == nil {
		return initVals
	}

	if hc.ValuesTemplate.ValueFiles != nil {
		vals.ValueFiles = hc.ValuesTemplate.ValueFiles
		log.Debugf("first check (%q)", hc.ValuesTemplate.ValueFiles)
	}

	if hc.ValuesTemplate.ValuesMap != nil {
		vals.Values = hc.getValuesAsList(hc.ValuesTemplate.ValuesMap)
		log.Debugf("second check (%q)", hc.ValuesTemplate.ValuesMap)
	}

	log.Debug("third check")

	if mergedVals, err = vals.MergeValues(getter.All(hc.Settings)); err != nil {
		return map[string]interface{}{}
	}

	return mergedVals
}

func (hc *HelmRelease) setValues() error {

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

func (hc HelmRelease) valuesChanged() (bool, error) {

	var installedValues map[string]interface{}
	var err error

	if installedValues, err = hc.getInstalledValues(); err != nil {
		return false, err
	}

	log.Debugf("installed values: (%v)", installedValues)

	defaultVals := hc.getDefaultValuesFromConfigMap(client.New(), "helm-default-"+hc.Chart+"-"+hc.Version)
	requestedValues := mergeMaps(hc.getValues(), defaultVals)

	for key := range installedValues {
		if _, ok := requestedValues[key]; !ok {
			log.Errorf("missing key %v", key)
		}
	}

	if len(requestedValues) < 1 && len(installedValues) < 1 {
		return false, nil
	}

	if reflect.DeepEqual(installedValues, requestedValues) {
		return false, nil
	}

	return true, nil
}

func (hc HelmRelease) getRelease() (*release.Release, error) {
	log.Debugf("config: %v", hc.Config)
	getConfig := hc.Config
	client := action.NewGet(getConfig)
	return client.Run(hc.Name)
}

func (hc HelmRelease) getChart(chartName string, chartPathOptions *action.ChartPathOptions) (*chart.Chart, error) {

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
	rc := client.New()

	log.Debugf("namespace: %v", hc.Namespace.Name)

	if jsonbody, err = rc.GetResource(chartName, hc.Namespace.Name, "charts", "helm.soer3n.info", "v1alpha1"); err != nil {
		return helmChart, err
	}

	if err = json.Unmarshal(jsonbody, &chartObj); err != nil {
		return helmChart, err
	}

	repoSelector := "repo=" + hc.Repo

	if _, ok := chartObj.ObjectMeta.Labels["repoGroup"]; ok {
		if len(chartObj.ObjectMeta.Labels["repoGroup"]) > 1 {
			repoSelector = "repoGroup=" + chartObj.ObjectMeta.Labels["repoGroup"]
		}
	}

	files = hc.getFiles(rc, chartObj)

	helmChart.Metadata.Name = chartName
	helmChart.Metadata.Version = hc.Version
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Files = files
	helmChart.Templates = hc.appendFilesFromConfigMap(rc, "helm-tmpl-"+hc.Chart+"-"+hc.Version, helmChart.Templates)
	helmChart.Values = hc.getDefaultValuesFromConfigMap(rc, "helm-default-"+hc.Chart+"-"+hc.Version)

	versionObj := chartObj.GetChartVersion(chartPathOptions.Version)

	if err := hc.addDependencies(rc, helmChart, versionObj.Dependencies, repoSelector); err != nil {
		return helmChart, err
	}

	if err := helmChart.Validate(); err != nil {
		return helmChart, err
	}

	return helmChart, nil
}

func (hc HelmRelease) getFiles(rc *client.Client, helmChart *helmv1alpha1.Chart) []*chart.File {

	files := []*chart.File{}

	files = hc.appendFilesFromConfigMap(rc, "helm-tmpl-"+hc.Chart+"-"+hc.Version, files)
	files = hc.appendFilesFromConfigMap(rc, "helm-crds-"+hc.Chart+"-"+hc.Version, files)

	return files
}

func (hc HelmRelease) addDependencies(rc *client.Client, chart *chart.Chart, deps []helmv1alpha1.ChartDep, selector string) error {

	var jsonbody []byte
	var chartList helmv1alpha1.ChartList
	var err error

	jsonbody, err = rc.SetOptions(metav1.ListOptions{
		LabelSelector: selector,
	}).ListResources(hc.Namespace.Name, "charts", "helm.soer3n.info", "v1alpha1")

	if err = json.Unmarshal(jsonbody, &chartList); err != nil {
		return err
	}

	options := &action.ChartPathOptions{}

	for _, item := range chartList.Items {
		for _, dep := range deps {
			if item.Spec.Name == dep.Name {
				options.RepoURL = dep.Repo
				options.Version = dep.Version

				subChart, _ := hc.getChart(item.Spec.Name, options)
				chart.AddDependency(subChart)
			}
		}
	}

	return nil
}

func (hc HelmRelease) appendFilesFromConfigMap(rc *client.Client, name string, list []*chart.File) []*chart.File {

	var jsonbody []byte
	var err error

	configmap := &v1.ConfigMap{}
	files := []*chart.File{}

	jsonbody, err = rc.GetResource(name, hc.Namespace.Name, "configmaps", "", "v1")

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

func (hc HelmRelease) getDefaultValuesFromConfigMap(rc *client.Client, name string) map[string]interface{} {

	var jsonbody []byte
	var err error
	values := make(map[string]interface{})
	configmap := &v1.ConfigMap{}

	jsonbody, err = rc.GetResource(name, hc.Namespace.Name, "configmaps", "", "v1")

	if err = json.Unmarshal(jsonbody, &configmap); err != nil {
		return values
	}

	jsonMap := make(map[string]interface{})

	if err = json.Unmarshal([]byte(configmap.Data["values"]), &jsonMap); err != nil {
		panic(err)
	}

	return jsonMap
}

func (hc HelmRelease) getRepo(rc *client.Client, repo string) (error, helmv1alpha1.Repo) {

	var jsonbody []byte
	var err error

	repoObj := &helmv1alpha1.Repo{}

	if jsonbody, err = rc.GetResource(repo, hc.Namespace.Name, "repos", "helm.soer3n.info", "v1alpha1"); err != nil {
		return err, *repoObj
	}

	log.Infof("Repo namespace: %v", hc.Namespace.Name)

	if err = json.Unmarshal(jsonbody, &repoObj); err != nil {
		return err, *repoObj
	}

	return nil, *repoObj
}

func (hc *HelmRelease) GetParsedConfigMaps() []v1.ConfigMap {

	var cp string
	var chartRequested *chart.Chart
	var err error

	configmapList := []v1.ConfigMap{}
	installConfig := hc.Config
	releaseClient := action.NewInstall(installConfig)
	releaseClient.ReleaseName = hc.Name
	hc.Client = releaseClient
	chartVersion := &HelmChartVersion{}

	log.Debugf("configinstall: %v", hc.Config)

	rc := client.New()

	_, repoObj := hc.getRepo(rc, hc.Repo)
	chartURL, _ := GetChartURL(rc, hc.Chart, hc.Version, hc.Namespace.Name)

	releaseClient.ReleaseName = hc.Name
	releaseClient.Version = hc.Version
	releaseClient.ChartPathOptions.RepoURL = repoObj.Spec.Url

	if cp, err = DownloadTo(chartURL, hc.Version, hc.Repo, hc.Settings, &releaseClient.ChartPathOptions); err != nil {
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

func (hc HelmRelease) upgrade(helmChart *chart.Chart, vals chartutil.Values) error {

	var rel *release.Release
	var err error

	client := action.NewUpgrade(hc.Config)

	if err = chartutil.ProcessDependencies(helmChart, vals); err != nil {
		return err
	}

	if rel, err = client.Run(hc.Name, helmChart, vals); err != nil {
		return err
	}

	log.Debugf("(%q) has been upgraded.", rel.Name)
	return nil
}
