package helm

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/internal/types"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
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
		K8sClient: k8sclient,
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
	var specValues map[string]interface{}
	var err error
	var ok bool

	installConfig := hc.Config

	if specValues, err = hc.getValues(); err != nil {
		return err
	}

	options := &action.ChartPathOptions{
		Version:               hc.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}

	if helmChart, err = hc.getChart(hc.Chart, options, specValues); err != nil {
		return err
	}

	log.Debugf("configupdate: %v", hc.Config)

	vals := helmChart.Values
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

	client := action.NewInstall(installConfig)
	client.ReleaseName = hc.Name
	client.Namespace = namespace.Name
	client.CreateNamespace = namespace.Install
	hc.setInstallFlags(client)

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

func (hc *Release) setInstallFlags(client *action.Install) {
	if hc.Flags == nil {
		log.Debugf("no flags set for release %v", hc.Name)
		return
	}

	client.Atomic = hc.Flags.Atomic
	client.DisableHooks = hc.Flags.DisableHooks
	client.DisableOpenAPIValidation = hc.Flags.DisableOpenAPIValidation
	client.DryRun = hc.Flags.DryRun
	client.SkipCRDs = hc.Flags.SkipCRDs
	client.SubNotes = hc.Flags.SubNotes
	client.Timeout = hc.Flags.Timeout
	client.Wait = hc.Flags.Wait
}

func (hc *Release) setUpgradeFlags(client *action.Upgrade) {
	if hc.Flags == nil {
		log.Debugf("no flags set for release %v", hc.Name)
		return
	}

	client.Atomic = hc.Flags.Atomic
	client.DisableHooks = hc.Flags.DisableHooks
	client.DisableOpenAPIValidation = hc.Flags.DisableOpenAPIValidation
	client.DryRun = hc.Flags.DryRun
	client.SkipCRDs = hc.Flags.SkipCRDs
	client.SubNotes = hc.Flags.SubNotes
	client.Timeout = hc.Flags.Timeout
	client.Wait = hc.Flags.Wait
	client.Force = hc.Flags.Force
	client.Recreate = hc.Flags.Recreate
	client.CleanupOnFail = hc.Flags.CleanupOnFail
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

func (hc Release) getChart(chartName string, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) (*chart.Chart, error) {
	helmChart := &chart.Chart{
		Metadata:  &chart.Metadata{},
		Files:     []*chart.File{},
		Templates: []*chart.File{},
		Values:    make(map[string]interface{}),
	}

	chartObj := &helmv1alpha1.Chart{}

	log.Debugf("namespace: %v", hc.Namespace.Name)

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: hc.Namespace.Name,
		Name:      chartName,
	}, chartObj); err != nil {
		return helmChart, err
	}

	repoSelector := make(map[string]string)

	if _, ok := chartObj.ObjectMeta.Labels["repoGroup"]; ok {
		if len(chartObj.ObjectMeta.Labels["repoGroup"]) > 1 {
			repoSelector["repoGroup"] = chartObj.ObjectMeta.Labels["repoGroup"]
		} else {
			repoSelector["repo"] = hc.Repo
		}
	}

	versionObj := utils.GetChartVersion(chartPathOptions.Version, chartObj)
	files := hc.getFiles(chartName, versionObj.Name, chartObj)

	if len(files) < 1 {
		return helmChart, errors.New("no files detected for chart resource")
	}

	helmChart.Metadata.Name = chartName
	helmChart.Metadata.Version = versionObj.Name
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Files = files
	helmChart.Templates = hc.appendFilesFromConfigMap(chartName + "-" + versionObj.Name + "-tmpl")

	defaultValues := hc.getDefaultValuesFromConfigMap("helm-default-" + chartName + "-" + versionObj.Name)
	helmChart.Values = defaultValues
	cv := mergeValues(vals, helmChart)
	helmChart.Values = cv

	if len(versionObj.Dependencies) > 0 {
		if err := hc.addDependencies(helmChart, versionObj.Dependencies, cv, repoSelector); err != nil {
			return helmChart, err
		}
	}

	if err := helmChart.Validate(); err != nil {
		return helmChart, err
	}

	return helmChart, nil
}

func (hc Release) getFiles(chartName, chartVersion string, helmChart *helmv1alpha1.Chart) []*chart.File {
	files := []*chart.File{}

	temp := hc.appendFilesFromConfigMap(chartName + "-" + chartVersion + "-tmpl")
	files = append(files, temp...)

	temp = hc.appendFilesFromConfigMap(chartName + "-" + chartVersion + "-crds")
	files = append(files, temp...)

	return files
}

func (hc Release) addDependencies(chart *chart.Chart, deps []*helmv1alpha1.ChartDep, vals chartutil.Values, selectors map[string]string) error {
	var chartList helmv1alpha1.ChartList
	var err error

	selectorObj := client.MatchingLabels{}

	for k, selector := range selectors {
		selectorObj[k] = selector
	}

	if err = hc.K8sClient.List(context.Background(), &chartList, selectorObj, client.InNamespace(hc.Namespace.Name)); err != nil {
		return err
	}

	options := &action.ChartPathOptions{}

	for _, item := range chartList.Items {
		for _, dep := range deps {
			if item.Spec.Name == dep.Name {
				options.RepoURL = dep.Repo
				options.Version = dep.Version
				var valueObj chartutil.Values

				depCondition := true
				conditional := strings.Split(dep.Condition, ".")

				if len(conditional) == 0 || len(conditional) > 2 {
					log.Errorf("failed to parse conditional for subchart %s", dep.Name)
					continue
				}

				// parse sub values for dependency
				subChartCondition, _ := vals[conditional[0]].(map[string]interface{})

				// getting subchart default value configmap
				subVals := hc.getDefaultValuesFromConfigMap("helm-default-" + dep.Name + "-" + dep.Version)

				// parse conditional to boolean
				if subChartCondition != nil {
					keyAsString := string(fmt.Sprint(subChartCondition[conditional[1]]))
					depCondition, _ = strconv.ParseBool(keyAsString)
				}

				// check conditional
				if depCondition {

					subChart, _ := hc.getChart(item.Spec.Name, options, subVals)

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

	// configmap := &v1.ConfigMap{}
	configmapList := v1.ConfigMapList{}
	files := []*chart.File{}

	selector := labels.NewSelector()
	requirement, _ := labels.NewRequirement(configMapLabelKey, selection.Equals, []string{name})
	selector = selector.Add(*requirement)

	if err = hc.K8sClient.List(context.Background(), &configmapList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return files
	}

	for _, configmap := range configmapList.Items {
		for key, data := range configmap.BinaryData {
			if name == "helm-crds-"+hc.Chart+"-"+hc.Version {
				key = "crds/" + key
			}

			baseName := "templates/"

			if configmap.ObjectMeta.Labels[configMapLabelSubName] != "" {
				baseName = baseName + configmap.ObjectMeta.Labels[configMapLabelSubName] + "/"
			}

			file := &chart.File{
				Name: baseName + key,
				Data: data,
			}
			files = append(files, file)
		}
	}

	return files
}

func (hc Release) getDefaultValuesFromConfigMap(name string) map[string]interface{} {
	var err error
	values := make(map[string]interface{})
	configmap := &v1.ConfigMap{}

	if err = hc.K8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: name}, configmap); err != nil {
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

	if err = hc.K8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: hc.Repo}, repoObj); err != nil {
		return *repoObj, err
	}

	log.Infof("Repo namespace: %v", hc.Namespace.Name)

	return *repoObj, nil
}

// GetParsedConfigMaps represents parsing and returning of chart related data for a release
func (hc *Release) GetParsedConfigMaps(namespace string) ([]v1.ConfigMap, []helmv1alpha1.Chart) {
	var chartRequested *chart.Chart
	var repoObj helmv1alpha1.Repo
	var chartObj helmv1alpha1.Chart
	chartObjList := &helmv1alpha1.ChartList{}
	chartObjList.Items = []helmv1alpha1.Chart{}
	var chartURL string
	var specValues map[string]interface{}
	var err error

	configmapList := []v1.ConfigMap{}
	installConfig := hc.Config
	releaseClient := action.NewInstall(installConfig)
	releaseClient.ReleaseName = hc.Name
	hc.Client = releaseClient
	chartVersion := &ChartVersion{}

	log.Debugf("configinstall: %v", hc.Config)

	if repoObj, err = hc.getRepo(); err != nil {
		return configmapList, chartObjList.Items
	}

	options := &action.ChartPathOptions{}
	options.RepoURL = hc.Repo
	options.Version = hc.Version

	if specValues, err = hc.getValues(); err != nil {
		return configmapList, chartObjList.Items
	}

	if chartRequested, err = hc.getChart(hc.Chart, options, specValues); err != nil {

		if chartURL, err = getChartURL(hc.K8sClient, hc.Chart, hc.Version, hc.Namespace.Name); err != nil {
			return configmapList, chartObjList.Items
		}

		releaseClient.ReleaseName = hc.Name
		releaseClient.Version = hc.Version
		releaseClient.ChartPathOptions.RepoURL = repoObj.Spec.URL
		credentials := &Auth{}

		if repoObj.Spec.AuthSecret != "" {
			credentials = hc.getCredentials(repoObj.Spec.AuthSecret)
		}

		if chartRequested, err = getChartByURL(chartURL, credentials, hc.getter); err != nil {
			return configmapList, chartObjList.Items
		}
	}

	if err = hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      hc.Chart,
	}, &chartObj); err != nil {
		return configmapList, chartObjList.Items
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
			return configmapList, chartObjList.Items
		}
	}

	chartVersion.Version.Metadata.Version = version.Name
	configmapList = chartVersion.createConfigMaps(hc.Namespace.Name, deps)
	// chartObjList = append(chartObjList, &chartObj)

	return configmapList, chartObjList.Items
}

func (hc Release) getCredentials(secret string) *Auth {
	secretObj := &v1.Secret{}
	creds := &Auth{}

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: secret}, secretObj); err != nil {
		return nil
	}

	if _, ok := secretObj.Data["user"]; !ok {
		log.Info("Username empty for repo auth")
	}

	if _, ok := secretObj.Data["password"]; !ok {
		log.Info("Password empty for repo auth")
	}

	username, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["user"]))
	pw, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["password"]))
	creds.User = strings.TrimSuffix(string(username), "\n")
	creds.Password = strings.TrimSuffix(string(pw), "\n")

	return creds
}

func (hc Release) validateChartSpec(deps []*chart.Chart, version *helmv1alpha1.ChartDep, chartObjList *helmv1alpha1.ChartList) error {
	subChartObj := &helmv1alpha1.Chart{}

	for _, d := range deps {

		if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
			Namespace: hc.Namespace.Name,
			Name:      version.Name,
		}, subChartObj); err != nil {
			return err
		}

		if version.Name == d.Name() {

			subVersion := utils.GetChartVersion(d.Metadata.Version, subChartObj)
			for _, sv := range subVersion.Dependencies {
				if err := hc.validateChartSpec(d.Dependencies(), sv, chartObjList); err != nil {
					return err
				}
			}

			/*
				set dependency version to fixed from loaded metadata
				instead of using semver version from parent chart
				and update parent chart resource
			*/
			version.Version = d.Metadata.Version
			chartObjList.Items = append(chartObjList.Items, *subChartObj)
		}

	}

	return nil
}

func (hc Release) upgrade(helmChart *chart.Chart, vals chartutil.Values, namespace string) error {
	var rel *release.Release
	var err error

	client := action.NewUpgrade(hc.Config)
	client.Namespace = namespace
	hc.setUpgradeFlags(client)

	if rel, err = client.Run(hc.Name, helmChart, vals); err != nil {
		log.Info(err.Error())
		return err
	}

	log.Debugf("(%q) has been upgraded.", rel.Name)
	return nil
}
