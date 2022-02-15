package release

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"strings"

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
func New(instance *helmv1alpha1.Release, settings *cli.EnvSettings, reqLogger logr.Logger, k8sclient client.Client, g utils.HTTPClientInterface, c kube.Client) *Release {
	var helmRelease *Release

	reqLogger.Info("init new release", "name", instance.Spec.Name, "repo", instance.Spec.Repo)

	helmRelease = &Release{
		Name: instance.Spec.Name,
		Namespace: Namespace{
			Name: instance.ObjectMeta.Namespace,
		},
		Version:   instance.Spec.Version,
		Repo:      instance.Spec.Repo,
		Chart:     instance.Spec.Chart,
		Settings:  settings,
		K8sClient: k8sclient,
		getter:    g,
		logger:    reqLogger.WithValues("release", instance.Spec.Name),
	}

	helmRelease.Config, _ = utils.InitActionConfig(settings, c)

	helmRelease.logger.Info("parsed config", "name", instance.Spec.Name, "cache", helmRelease.Settings.RepositoryCache)

	helmRelease.ValuesTemplate = values.New(instance, helmRelease.logger, helmRelease.K8sClient)

	return helmRelease
}

// Update represents update or installation process of a release
func (hc *Release) Update(namespace helmv1alpha1.Namespace) error {
	hc.logger.Info("config install: "+fmt.Sprint(hc.Config), "name", hc.Name, "repo", hc.Repo)

	var release *release.Release
	var helmChart *helmchart.Chart
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

	hc.logger.Info("configupdate: "+fmt.Sprint(hc.Config), "name", hc.Name, "repo", hc.Repo)

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
		hc.logger.Error(err, "error on installing chart", "release", hc.Name, "chart", hc.Chart, "repo", hc.Repo)
		return err
	}

	hc.logger.Info("release successfully installed.", "name", release.Name, "namespace", namespace, "chart", hc.Chart, "repo", hc.Repo)
	return nil
}

func (hc *Release) setInstallFlags(client *action.Install) {
	if hc.Flags == nil {
		hc.logger.Info("no flags set for release", "name", hc.Name, "chart", hc.Chart, "repo", hc.Repo)
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
		hc.logger.Info("no flags set for release", "name", hc.Name, "chart", hc.Chart, "repo", hc.Repo)
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
func (hc Release) Remove() error {
	client := action.NewUninstall(hc.Config)
	_, err := client.Run(hc.Name)
	return err
}

func (hc Release) getRelease() (*release.Release, error) {
	getConfig := hc.Config
	client := action.NewGet(getConfig)
	return client.Run(hc.Name)
}

func (hc Release) getChart(chartName string, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) (*helmchart.Chart, error) {
	helmChart := &helmchart.Chart{
		Metadata:  &helmchart.Metadata{},
		Files:     []*helmchart.File{},
		Templates: []*helmchart.File{},
		Values:    make(map[string]interface{}),
	}

	chartObj := &helmv1alpha1.Chart{}

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
		return helmChart, errors.NewBadRequest("no files detected for chart resource")
	}

	helmChart.Metadata.Name = chartName
	helmChart.Metadata.Version = versionObj.Name
	helmChart.Metadata.APIVersion = chartObj.Spec.APIVersion
	helmChart.Files = files
	helmChart.Templates = hc.appendFilesFromConfigMap(chartName + "-" + versionObj.Name + "-tmpl")

	defaultValues := hc.getDefaultValuesFromConfigMap("helm-default-" + chartName + "-" + versionObj.Name)
	helmChart.Values = defaultValues
	cv := values.MergeValues(vals, helmChart)
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

func (hc Release) getRepo() (helmv1alpha1.Repo, error) {
	var err error

	repoObj := &helmv1alpha1.Repo{}

	if err = hc.K8sClient.Get(context.Background(), types.NamespacedName{Namespace: hc.Namespace.Name, Name: hc.Repo}, repoObj); err != nil {
		return *repoObj, err
	}

	return *repoObj, nil
}

func (hc Release) getCredentials(secret string) *chart.Auth {
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

func (hc Release) validateChartSpec(deps []*helmchart.Chart, version *helmv1alpha1.ChartDep, chartObjList *helmv1alpha1.ChartList) error {
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

func (hc Release) upgrade(helmChart *helmchart.Chart, vals chartutil.Values, namespace string) error {
	var rel *release.Release
	var err error

	client := action.NewUpgrade(hc.Config)
	client.Namespace = namespace
	hc.setUpgradeFlags(client)

	if rel, err = client.Run(hc.Name, helmChart, vals); err != nil {
		hc.logger.Info(err.Error())
		return err
	}

	hc.logger.Info("successfully upgraded.", "name", rel.Name, "chart", hc.Chart, "repo", hc.Repo)
	return nil
}
