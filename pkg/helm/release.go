package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/prometheus/common/log"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/apps-operator/internal/types"
	"github.com/soer3n/apps-operator/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewHelmRelease represents initialization of internal release struct
func NewHelmRelease(instance *helmv1alpha1.Release, settings *cli.EnvSettings, k8sclient client.Client, g inttypes.HTTPClientInterface, c kube.Client) *Release {

	var helmRelease *Release

	log.Debugf("Trying HelmRelease %v", instance.Spec.Name)

	helmRelease = &Release{
		Name:      instance.Spec.Name,
		Repo:      instance.Spec.Repo,
		Chart:     instance.Spec.Chart,
		Settings:  settings,
		k8sClient: k8sclient,
		getter:    g,
	}

	helmRelease.Config, _ = initActionConfig(settings, c)

	log.Debugf("HelmRelease config path: %v", helmRelease.Settings.RepositoryCache)

	if instance.Spec.ValuesTemplate != nil {
		if instance.Spec.ValuesTemplate.ValueRefs != nil {
			helmRelease.ValuesTemplate = &ValueTemplate{
				valuesRef: []*ValuesRef{},
			}
		}
	}

	return helmRelease
}

// Update represents update or installation process of a release
func (hc *Release) Update(namespace helmv1alpha1.Namespace) error {

	log.Debugf("config install: %v", fmt.Sprint(hc.Config))

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
	release, _ = hc.getRelease()

	var specValues map[string]interface{}

	if specValues, err = hc.getValues(); err != nil {
		return err
	}

	client.Namespace = namespace.Name
	client.CreateNamespace = namespace.Install
	vals := mergeMaps(specValues, helmChart.Values)

	// Check if something changed regarding the existing release
	if release != nil {
		if ok, err = hc.valuesChanged(vals); err != nil {
			return err
		}

		if ok {
			return hc.upgrade(helmChart, vals, namespace.Name)
		}

		return nil
	}

	if err = chartutil.ProcessDependencies(helmChart, vals); err != nil {
		return err
	}

	if release, err = client.Run(helmChart, vals); err != nil {
		return err
	}

	log.Debugf("Release (%q) successfully installed in namespace %v.", release.Name, namespace)
	return nil
}

// Remove represents removing release related resource
func (hc Release) Remove() error {
	client := action.NewUninstall(hc.Config)
	_, err := client.Run(hc.Name)
	return err
}

func (hc *Release) getValues() (map[string]interface{}, error) {

	templateObj := hc.ValuesTemplate

	returnValues, err := templateObj.ManageValues()

	if err != nil {
		return templateObj.Values, err
	}

	hc.Values = templateObj.Values
	hc.ValuesTemplate.ValuesMap = templateObj.ValuesMap

	return returnValues, nil
}

func (hc Release) getValuesAsList(values map[string]string) []string {

	var valueList []string
	var transformedVal string
	valueList = []string{}

	for k, v := range values {
		transformedVal = k + "=" + v
		valueList = append(valueList, transformedVal)
	}

	return valueList
}

func (hc Release) getInstalledValues() (map[string]interface{}, error) {

	client := action.NewGetValues(hc.Config)
	return client.Run(hc.Name)
}

func (hc Release) valuesChanged(vals map[string]interface{}) (bool, error) {

	var installedValues map[string]interface{}
	var err error

	if installedValues, err = hc.getInstalledValues(); err != nil {
		return false, err
	}

	log.Debugf("installed values: (%v)", installedValues)

	defaultValues := hc.getDefaultValuesFromConfigMap("helm-default-" + hc.Chart + "-" + hc.Version)
	requestedValues := mergeMaps(vals, defaultValues)

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

func (hc Release) getRelease() (*release.Release, error) {
	log.Debugf("config: %v", hc.Config)
	getConfig := hc.Config
	client := action.NewGet(getConfig)
	return client.Run(hc.Name)
}

func (hc Release) getChart(chartName string, chartPathOptions *action.ChartPathOptions) (*chart.Chart, error) {

	helmChart := &chart.Chart{
		Metadata:  &chart.Metadata{},
		Files:     []*chart.File{},
		Templates: []*chart.File{},
		Values:    make(map[string]interface{}),
	}

	chartObj := &helmv1alpha1.Chart{}

	log.Debugf("namespace: %v", hc.Namespace.Name)

	if err := hc.k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: hc.Namespace.Name,
		Name:      hc.Chart,
	}, chartObj); err != nil {
		return helmChart, err
	}

	repoSelector := make(map[string]string, 0)

	if _, ok := chartObj.ObjectMeta.Labels["repoGroup"]; ok {
		if len(chartObj.ObjectMeta.Labels["repoGroup"]) > 1 {
			repoSelector["repoGroup"] = chartObj.ObjectMeta.Labels["repoGroup"]
		} else {
			repoSelector["repo"] = hc.Repo
		}
	}

	files := hc.getFiles(chartObj)

	helmChart.Metadata.Name = chartName
	helmChart.Metadata.Version = hc.Version
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Files = files
	helmChart.Templates = hc.appendFilesFromConfigMap("helm-tmpl-"+hc.Chart+"-"+hc.Version, helmChart.Templates)
	helmChart.Values = hc.getDefaultValuesFromConfigMap("helm-default-" + hc.Chart + "-" + hc.Version)

	versionObj := utils.GetChartVersion(chartPathOptions.Version, chartObj)

	if err := hc.addDependencies(helmChart, versionObj.Dependencies, repoSelector); err != nil {
		return helmChart, err
	}

	if err := helmChart.Validate(); err != nil {
		return helmChart, err
	}

	return helmChart, nil
}

func (hc Release) getFiles(helmChart *helmv1alpha1.Chart) []*chart.File {

	files := []*chart.File{}

	files = hc.appendFilesFromConfigMap("helm-tmpl-"+hc.Chart+"-"+hc.Version, files)
	files = hc.appendFilesFromConfigMap("helm-crds-"+hc.Chart+"-"+hc.Version, files)

	return files
}

func (hc Release) addDependencies(chart *chart.Chart, deps []helmv1alpha1.ChartDep, selectors map[string]string) error {

	var chartList helmv1alpha1.ChartList
	var err error

	selectorObj := client.MatchingLabels{}

	for k, selector := range selectors {
		selectorObj[k] = selector
	}

	if err = hc.k8sClient.List(context.Background(), &chartList, selectorObj, client.InNamespace(hc.Namespace.Name)); err != nil {
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

func (hc Release) appendFilesFromConfigMap(name string, list []*chart.File) []*chart.File {

	var err error

	configmap := &v1.ConfigMap{}
	files := []*chart.File{}

	if err = hc.k8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: name}, configmap); err != nil {
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

func (hc Release) getDefaultValuesFromConfigMap(name string) map[string]interface{} {

	var err error
	values := make(map[string]interface{})
	configmap := &v1.ConfigMap{}

	if err = hc.k8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: name}, configmap); err != nil {
		return values
	}

	jsonMap := make(map[string]interface{})

	if err = json.Unmarshal([]byte(configmap.Data["values"]), &jsonMap); err != nil {
		panic(err)
	}

	return jsonMap
}

func (hc Release) getRepo() (helmv1alpha1.Repo, error) {

	var err error

	repoObj := &helmv1alpha1.Repo{}

	if err = hc.k8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: hc.Repo}, repoObj); err != nil {
		return *repoObj, err
	}

	log.Infof("Repo namespace: %v", hc.Namespace.Name)

	return *repoObj, nil
}

// GetParsedConfigMaps represents parsing and returning of chart related data for a release
func (hc *Release) GetParsedConfigMaps() []v1.ConfigMap {

	var chartRequested *chart.Chart
	var repoObj helmv1alpha1.Repo
	var chartURL string
	var err error

	configmapList := []v1.ConfigMap{}
	installConfig := hc.Config
	releaseClient := action.NewInstall(installConfig)
	releaseClient.ReleaseName = hc.Name
	hc.Client = releaseClient
	chartVersion := &ChartVersion{}

	log.Debugf("configinstall: %v", hc.Config)

	if repoObj, err = hc.getRepo(); err != nil {
		return configmapList
	}
	if chartURL, err = getChartURL(hc.k8sClient, hc.Chart, hc.Version, hc.Namespace.Name); err != nil {
		return configmapList
	}

	releaseClient.ReleaseName = hc.Name
	releaseClient.Version = hc.Version
	releaseClient.ChartPathOptions.RepoURL = repoObj.Spec.URL

	if chartRequested, err = getChartByURL(chartURL, hc.getter); err != nil {
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

	return chartVersion.createConfigMaps(hc.Namespace.Name)
}

func (hc Release) upgrade(helmChart *chart.Chart, vals chartutil.Values, namespace string) error {

	var rel *release.Release
	var err error

	client := action.NewUpgrade(hc.Config)
	client.Namespace = namespace

	if err = chartutil.ProcessDependencies(helmChart, vals); err != nil {
		return err
	}

	if rel, err = client.Run(hc.Name, helmChart, vals); err != nil {
		return err
	}

	log.Debugf("(%q) has been upgraded.", rel.Name)
	return nil
}
