package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

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
func (hc *Release) Update(namespace helmv1alpha1.Namespace, dependenciesConfig map[string]helmv1alpha1.DependencyConfig) error {

	log.Debugf("config install: %v", fmt.Sprint(hc.Config))

	var release *release.Release
	var helmChart *chart.Chart
	var err error
	var ok bool

	installConfig := hc.Config
	client := action.NewInstall(installConfig)
	client.ReleaseName = hc.Name
	hc.Client = client

	var specValues map[string]interface{}

	if specValues, err = hc.getValues(); err != nil {
		return err
	}

	// parsing values; goroutines are nessecarry due to tail recursion in called funcs
	defaultValues := hc.getDefaultValuesFromConfigMap("helm-default-" + hc.Chart + "-" + hc.Version)

	// we have to wait until each goroutine is finished for merging values
	var wg sync.WaitGroup
	c := make(chan map[string]interface{})
	vals := make(map[string]interface{}, VALUES_MAP_SIZE)
	log.Info(unsafe.Sizeof(specValues))
	log.Info(unsafe.Alignof(specValues))

	// iterate through first level keys and call func for merging as a goroutine to avoid memory leaks
	for k, v := range specValues {

		wg.Add(1)

		go func(k string, v interface{}, c chan map[string]interface{}, defaultValues map[string]interface{}) {
			defer wg.Done()
			temp, _ := v.(map[string]interface{})
			tempDefault, _ := defaultValues[k].(map[string]interface{})
			c <- mergeUntypedMaps(tempDefault, temp, k)
		}(k, v, c, defaultValues)

	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for i := range c {
		vals = mergeMaps(i, vals)
	}
	client.Namespace = namespace.Name
	client.CreateNamespace = namespace.Install

	options := &action.ChartPathOptions{
		Version:               hc.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}
	if helmChart, err = hc.getChart(hc.Chart, options, dependenciesConfig, vals); err != nil {
		return err
	}

	log.Debugf("configupdate: %v", hc.Config)
	release, _ = hc.getRelease()

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

	if release, err = client.Run(helmChart, vals); err != nil {
		log.Info(err.Error())
		return err
	}

	log.Debugf("Release (%q) successfully installed in namespace %v.", release.Name, namespace)
	return nil
}

// InitValuesTemplate represents initialization of value template by list of refs from kubernetes api
func (hc *Release) InitValuesTemplate(refList []*ValuesRef, version, namespace string) {
	hc.ValuesTemplate = NewValueTemplate(refList)
	hc.Namespace.Name = namespace
	hc.Version = version
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

	for key := range installedValues {
		if _, ok := vals[key]; !ok {
			log.Errorf("missing key %v", key)
		}
	}

	if len(vals) < 1 && len(installedValues) < 1 {
		return false, nil
	}

	if reflect.DeepEqual(installedValues, vals) {
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

func (hc Release) getChart(chartName string, chartPathOptions *action.ChartPathOptions, dependenciesConfig map[string]helmv1alpha1.DependencyConfig, vals map[string]interface{}) (*chart.Chart, error) {

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
		Name:      chartName,
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

	versionObj := utils.GetChartVersion(chartPathOptions.Version, chartObj)
	files := hc.getFiles(chartName, versionObj.Name, chartObj)

	helmChart.Metadata.Name = chartName
	helmChart.Metadata.Version = chartObj.Spec.APIVersion
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Files = files
	helmChart.Templates = hc.appendFilesFromConfigMap("helm-tmpl-" + chartName + "-" + versionObj.Name)

	helmChart.Values = mergeMaps(hc.getDefaultValuesFromConfigMap("helm-default-"+chartName+"-"+versionObj.Name), vals)

	if err := hc.addDependencies(helmChart, versionObj.Dependencies, vals, dependenciesConfig, repoSelector); err != nil {
		return helmChart, err
	}

	if err := helmChart.Validate(); err != nil {
		return helmChart, err
	}

	return helmChart, nil
}

func (hc Release) getFiles(chartName, chartVersion string, helmChart *helmv1alpha1.Chart) []*chart.File {

	files := []*chart.File{}

	for _, temp := range hc.appendFilesFromConfigMap("helm-tmpl-" + chartName + "-" + chartVersion) {
		files = append(files, temp)
	}

	for _, temp := range hc.appendFilesFromConfigMap("helm-crds-" + chartName + "-" + chartVersion) {
		files = append(files, temp)

	}

	return files
}

func (hc Release) addDependencies(chart *chart.Chart, deps []*helmv1alpha1.ChartDep, vals map[string]interface{}, dependenciesConfig map[string]helmv1alpha1.DependencyConfig, selectors map[string]string) error {

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
				var valueObj chartutil.Values

				if dependenciesConfig[dep.Name].Enabled {
					subVals, _ := vals[dep.Name].(map[string]interface{})
					subChart, _ := hc.getChart(item.Spec.Name, options, dependenciesConfig, subVals)

					if valueObj, err = chartutil.ToRenderValues(subChart, subVals, chartutil.ReleaseOptions{}, nil); err != nil {
						return err
					}

					// get values as interface{}
					valueMap := valueObj.AsMap()["Values"]
					// cast to struct
					castedMap, _ := valueMap.(chartutil.Values)
					subChart.Values = castedMap
					chart.AddDependency(subChart)
				}
			}
		}
	}

	return nil
}

func (hc Release) appendFilesFromConfigMap(name string) []*chart.File {

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
func (hc *Release) GetParsedConfigMaps(namespace string) ([]v1.ConfigMap, []*helmv1alpha1.Chart) {

	var chartRequested *chart.Chart
	var repoObj helmv1alpha1.Repo
	var chartObj helmv1alpha1.Chart
	chartObjList := []*helmv1alpha1.Chart{}
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
		return configmapList, chartObjList
	}
	if chartURL, err = getChartURL(hc.k8sClient, hc.Chart, hc.Version, hc.Namespace.Name); err != nil {
		return configmapList, chartObjList
	}

	releaseClient.ReleaseName = hc.Name
	releaseClient.Version = hc.Version
	releaseClient.ChartPathOptions.RepoURL = repoObj.Spec.URL

	if chartRequested, err = getChartByURL(chartURL, hc.getter); err != nil {
		return configmapList, chartObjList
	}

	if err = hc.k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      hc.Chart,
	}, &chartObj); err != nil {
		return configmapList, chartObjList
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
	deps := chartRequested.Dependencies()
	version := utils.GetChartVersion(hc.Version, &chartObj)

	for _, v := range version.Dependencies {
		if err := hc.validateChartSpec(deps, v, chartObjList); err != nil {
			return configmapList, chartObjList
		}
	}

	chartVersion.Version.Metadata.Version = version.Name
	configmapList = chartVersion.createConfigMaps(hc.Namespace.Name, deps)

	return configmapList, chartObjList
}

func (hc Release) validateChartSpec(deps []*chart.Chart, version *helmv1alpha1.ChartDep, chartObjList []*helmv1alpha1.Chart) error {

	subChartObj := &helmv1alpha1.Chart{}

	for _, d := range deps {

		if err := hc.k8sClient.Get(context.Background(), types.NamespacedName{
			Namespace: hc.Namespace.Name,
			Name:      version.Name,
		}, subChartObj); err != nil {
			return err
		}

		if version.Name == d.Name() && version.Version != d.Metadata.Version {
			version.Version = d.Metadata.Version
			chartObjList = append(chartObjList, subChartObj)
		}

		subVersion := utils.GetChartVersion(version.Version, subChartObj)
		for _, sv := range subVersion.Dependencies {
			if err := hc.validateChartSpec(d.Dependencies(), sv, chartObjList); err != nil {
				return err
			}
		}
	}

	return nil
}

func (hc Release) upgrade(helmChart *chart.Chart, vals chartutil.Values, namespace string) error {

	var rel *release.Release
	var err error

	client := action.NewUpgrade(hc.Config)
	client.Namespace = namespace

	if rel, err = client.Run(hc.Name, helmChart, vals); err != nil {
		log.Info(err.Error())
		return err
	}

	log.Debugf("(%q) has been upgraded.", rel.Name)
	return nil
}
