package release

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	"github.com/soer3n/yaho/internal/utils"
	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/release"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const configMapLabelKey = "helm.soer3n.info/chart"

// const configMapRepoLabelKey = "helm.soer3n.info/repo"
const configMapLabelSubName = "helm.soer3n.info/subname"

// New represents initialization of internal release struct
func New(instance *helmv1alpha1.Release, settings *cli.EnvSettings, reqLogger logr.Logger, k8sclient client.Client, g utils.HTTPClientInterface, c kube.Client) (*Release, error) {
	var helmRelease *Release
	var helmChart *helmchart.Chart
	var repoObj *helmv1alpha1.Repo
	var specValues map[string]interface{}
	var err error

	reqLogger.Info("init new release", "name", instance.Spec.Name, "repo", instance.Spec.Repo)

	helmRelease = &Release{
		Name: instance.Spec.Name,
		Namespace: Namespace{
			Name: instance.ObjectMeta.Namespace,
		},
		Version:   instance.Spec.Version,
		Repo:      instance.Spec.Repo,
		Settings:  settings,
		K8sClient: k8sclient,
		getter:    g,
		logger:    reqLogger.WithValues("release", instance.Spec.Name),
		wg:        &sync.WaitGroup{},
		mu:        sync.Mutex{},
	}

	helmRelease.releaseNamespace = instance.ObjectMeta.Namespace

	if instance.Spec.Namespace != nil {
		helmRelease.releaseNamespace = *instance.Spec.Namespace
	}

	helmRelease.Config, _ = utils.InitActionConfig(settings, c)

	helmRelease.logger.Info("parsed config", "name", instance.Spec.Name, "cache", helmRelease.Settings.RepositoryCache)

	if instance.Spec.Config != nil {
		if err := helmRelease.setOptions(instance.Spec.Config, instance.Spec.Namespace); err != nil {
			helmRelease.logger.Error(err, "set options", "name", instance.Spec.Name)
		}
	}

	helmRelease.logger.Info("set options", "name", instance.Spec.Name)

	helmRelease.ValuesTemplate = values.New(instance, helmRelease.logger, helmRelease.K8sClient)

	if specValues, err = helmRelease.getValues(); err != nil {
		return helmRelease, err
	}

	helmRelease.Values = specValues

	options := &action.ChartPathOptions{
		Version:               instance.Spec.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}

	if helmChart, err = helmRelease.getChart(instance.Spec.Chart, options, specValues); err != nil {

		if repoObj, err = helmRelease.getControllerRepo(instance.Spec.Repo, instance.ObjectMeta.Namespace); err != nil {
			return helmRelease, err
		}

		releaseClient := action.NewInstall(helmRelease.Config)
		releaseClient.ReleaseName = instance.Spec.Name

		if helmChart, err = helmRelease.loadChart(instance.Spec.Chart, releaseClient, repoObj); err != nil {
			return helmRelease, err
		}
	}

	helmRelease.Chart = helmChart

	return helmRelease, nil
}

func (hc *Release) setOptions(name, namespace *string) error {

	instance := &helmv1alpha1.Config{}

	hc.logger.Info(hc.Namespace.Name)

	err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Name:      *name,
		Namespace: hc.Namespace.Name,
	}, instance)

	if err != nil {
		return err
	}

	hc.Flags = instance.Spec.Flags

	for _, v := range instance.Spec.Namespace.Allowed {
		if v == *namespace {
			return nil
		}
	}

	return errors.NewBadRequest("namespace not in allowed list")
}

// Update represents update or installation process of a release
func (hc *Release) Update() error {

	if hc.Chart == nil {
		return errors.NewBadRequest("chart not loaded on action update")
	}

	hc.logger.Info("config install: "+fmt.Sprint(hc.Config), "name", hc.Name, "repo", hc.Repo)

	var release *release.Release
	var err error
	var ok bool

	installConfig := hc.Config

	hc.logger.Info("configupdate: "+fmt.Sprint(hc.Config), "name", hc.Name, "repo", hc.Repo)

	vals := hc.Chart.Values
	release, _ = hc.getRelease()

	// Check if something changed regarding the existing release
	if release != nil {
		if ok, err = hc.valuesChanged(vals); err != nil {
			return err
		}

		if ok {
			if err := hc.upgrade(hc.Chart, vals); err != nil {
				return err
			}
			hc.logger.Info("release updated.", "name", release.Name, "namespace", release.Namespace, "chart", hc.Chart.Name(), "repo", hc.Repo)
		}

		hc.logger.Info("nothing changed for release.", "name", release.Name, "namespace", release.Namespace, "chart", hc.Chart.Name(), "repo", hc.Repo)
		return nil
	}

	client := action.NewInstall(installConfig)
	client.ReleaseName = hc.Name
	client.Namespace = hc.releaseNamespace
	client.CreateNamespace = false
	hc.setInstallFlags(client)

	if release, err = client.Run(hc.Chart, vals); err != nil {
		hc.logger.Error(err, "error on installing chart", "release", hc.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
		return err
	}

	hc.logger.Info("release successfully installed.", "name", release.Name, "namespace", release.Namespace, "chart", hc.Chart.Name(), "repo", hc.Repo)
	return nil
}

func (hc *Release) setInstallFlags(client *action.Install) {
	if hc.Flags == nil {
		hc.logger.Info("no flags set for release", "name", hc.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
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
		hc.logger.Info("no flags set for release", "name", hc.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
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

func (hc *Release) getControllerRepo(name, namespace string) (*helmv1alpha1.Repo, error) {
	instance := &helmv1alpha1.Repo{}

	err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			hc.logger.Info("HelmRepo resource not found. Ignoring since object must be deleted")
			return instance, err
		}
		// Error reading the object - requeue the request.
		hc.logger.Error(err, "Failed to get ControllerRepo")
		return instance, err
	}

	return instance, nil
}

// Remove represents removing release related resource
func (hc *Release) Remove() error {
	client := action.NewUninstall(hc.Config)
	_, err := client.Run(hc.Name)
	return err
}

func (hc *Release) getRelease() (*release.Release, error) {
	getConfig := hc.Config
	client := action.NewGet(getConfig)
	return client.Run(hc.Name)
}

func (hc *Release) loadChart(name string, releaseClient *action.Install, repoObj *helmv1alpha1.Repo) (*helmchart.Chart, error) {
	var chartRequested *helmchart.Chart
	var chartURL string
	var err error

	if chartURL, err = chart.GetChartURL(hc.K8sClient, name, hc.Version, hc.Namespace.Name); err != nil {
		return chartRequested, err
	}

	releaseClient.ReleaseName = hc.Name
	releaseClient.Version = hc.Version
	releaseClient.ChartPathOptions.RepoURL = repoObj.Spec.URL
	credentials := &chart.Auth{}

	if repoObj.Spec.AuthSecret != "" {
		credentials = hc.getCredentials(repoObj.Spec.AuthSecret)
	}

	if chartRequested, err = chart.GetChartByURL(chartURL, credentials, hc.getter); err != nil {
		return chartRequested, err
	}

	return chartRequested, nil
}

func (hc *Release) getChart(chartName string, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) (*helmchart.Chart, error) {

	helmChart := &helmchart.Chart{}
	chartObj := &helmv1alpha1.Chart{}
	repoSelector := make(map[string]string)

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: hc.Namespace.Name,
		Name:      chartName,
	}, chartObj); err != nil {
		return nil, err
	}

	hc.wg.Add(3)

	go func() {
		defer hc.wg.Done()
		hc.setVersion(helmChart, chartName, chartObj)
	}()

	go func() {
		defer hc.wg.Done()
		hc.setValues(helmChart, chartName, vals)
	}()

	go func() {
		defer hc.wg.Done()
		hc.setFiles(helmChart, chartName, chartObj)
	}()

	hc.wg.Wait()

	if _, ok := chartObj.ObjectMeta.Labels["repoGroup"]; ok {
		if len(chartObj.ObjectMeta.Labels["repoGroup"]) > 1 {
			repoSelector["repoGroup"] = chartObj.ObjectMeta.Labels["repoGroup"]
		} else {
			repoSelector["repo"] = hc.Repo
		}
	}

	tempVersion := utils.GetChartVersion(hc.Version, chartObj)

	if len(tempVersion.Dependencies) > 0 {
		if err := hc.addDependencies(helmChart, tempVersion.Dependencies, helmChart.Values, repoSelector); err != nil {
			return helmChart, err
		}
	}

	if len(helmChart.Files) < 1 {
		return helmChart, errors.NewBadRequest("no files detected for chart resource")
	}

	// validate after channels are closed
	if err := helmChart.Validate(); err != nil {
		return helmChart, err
	}

	return helmChart, nil
}

func (hc *Release) setValues(helmChart *helmchart.Chart, chartName string, vals map[string]interface{}) {
	defer hc.mu.Unlock()
	hc.mu.Lock()
	defaultValues := hc.getDefaultValuesFromConfigMap("helm-default-" + chartName + "-" + hc.Version)
	helmChart.Values = defaultValues
	cv := values.MergeValues(vals, helmChart)
	helmChart.Values = cv
}

func (hc *Release) setVersion(helmChart *helmchart.Chart, chartName string, chartObj *helmv1alpha1.Chart) {
	defer hc.mu.Unlock()
	hc.mu.Lock()
	if helmChart.Metadata == nil {
		helmChart.Metadata = &helmchart.Metadata{}
	}
	helmChart.Metadata.Version = hc.Version
	helmChart.Metadata.Name = chartName
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Templates = hc.appendFilesFromConfigMap(chartName, hc.Version, "tmpl")
}

func (hc *Release) setFiles(helmChart *helmchart.Chart, chartName string, chartObj *helmv1alpha1.Chart) {
	defer hc.mu.Unlock()
	hc.mu.Lock()
	files := hc.getFiles(chartName, hc.Version, chartObj)
	helmChart.Files = files
}

func (hc *Release) getCredentials(secret string) *chart.Auth {
	secretObj := &v1.Secret{}
	creds := &chart.Auth{}

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: secret}, secretObj); err != nil {
		return nil
	}

	if _, ok := secretObj.Data["user"]; !ok {
		hc.logger.Info("Username empty for repo auth")
	}

	if _, ok := secretObj.Data["password"]; !ok {
		hc.logger.Info("Password empty for repo auth")
	}

	username, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["user"]))
	pw, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["password"]))
	creds.User = strings.TrimSuffix(string(username), "\n")
	creds.Password = strings.TrimSuffix(string(pw), "\n")

	return creds
}

func (hc *Release) validateChartSpec(c chan helmv1alpha1.Chart, deps []*helmchart.Chart, version *helmv1alpha1.ChartDep) error {

	for _, d := range deps {

		subChartObj := helmv1alpha1.Chart{}

		if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
			Namespace: hc.Namespace.Name,
			Name:      version.Name,
		}, &subChartObj); err != nil {
			return err
		}

		if version.Name == d.Name() {

			subVersion := utils.GetChartVersion(d.Metadata.Version, &subChartObj)
			for _, sv := range subVersion.Dependencies {
				if err := hc.validateChartSpec(c, d.Dependencies(), sv); err != nil {
					return err
				}
			}

			/*
				set dependency version to fixed from loaded metadata
				instead of using semver version from parent chart
				and update parent chart resource
			*/
			version.Version = d.Metadata.Version
			c <- subChartObj
		}

	}

	return nil
}

func (hc *Release) upgrade(helmChart *helmchart.Chart, vals chartutil.Values) error {
	var rel *release.Release
	var err error

	client := action.NewUpgrade(hc.Config)
	client.Namespace = hc.Namespace.Name
	hc.setUpgradeFlags(client)

	if rel, err = client.Run(hc.Name, helmChart, vals); err != nil {
		hc.logger.Info(err.Error())
		return err
	}

	hc.logger.Info("successfully upgraded.", "name", rel.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
	return nil
}
