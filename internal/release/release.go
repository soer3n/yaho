package release

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"github.com/soer3n/yaho/internal/chart"
	"github.com/soer3n/yaho/internal/chartversion"
	"github.com/soer3n/yaho/internal/utils"
	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const configMapLabelKey = "yaho.soer3n.dev/chart"
const configMapRepoLabelKey = "yaho.soer3n.dev/repo"

// const configMapLabelSubName = "yaho.soer3n.dev/subname"

// New represents initialization of internal release struct
func New(instance *yahov1alpha2.Release, watchNamespace string, scheme *runtime.Scheme, reqLogger logr.Logger, k8sclient client.WithWatch, g utils.HTTPClientInterface, getter genericclioptions.RESTClientGetter, kubeconfig []byte) (*Release, error) {
	var helmRelease *Release
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
		K8sClient: k8sclient,
		scheme:    scheme,
		getter:    g,
		logger:    reqLogger.WithValues("release", instance.Spec.Name),
		wg:        &sync.WaitGroup{},
		mu:        sync.Mutex{},
	}

	helmRelease.releaseNamespace = instance.ObjectMeta.Namespace

	if instance.Spec.Namespace != nil {
		helmRelease.releaseNamespace = *instance.Spec.Namespace
	}

	helmRelease.Config, _ = utils.InitActionConfig(getter, kubeconfig, reqLogger)

	if instance.Spec.Config != nil {
		config, err := helmRelease.getConfig(instance.Spec.Config)

		if err != nil {
			helmRelease.logger.Info(err.Error())
			return helmRelease, err
		}

		helmRelease.logger.Info("parsed config", "name", instance.Spec.Name, "config", config)

		if err := helmRelease.setOptions(config, instance.Spec.Namespace); err != nil {
			helmRelease.logger.Error(err, "set options", "name", instance.Spec.Name)
			return helmRelease, err
		}
	}

	helmRelease.logger.Info("set options", "name", instance.Spec.Name)

	shouldBeDeleted := instance.GetDeletionTimestamp() != nil
	if shouldBeDeleted {
		return helmRelease, nil
	}

	helmRelease.ValuesTemplate = values.New(instance, helmRelease.logger, helmRelease.K8sClient)

	if len(instance.Spec.Values) != 0 {
		if specValues, err = helmRelease.getValues(); err != nil {
			return helmRelease, err
		}
	}

	helmRelease.ValuesTemplate.Values = specValues
	helmRelease.logger.Info("specified values: %v", "values", helmRelease.ValuesTemplate.Values)
	// TODO: rework getting index configmap
	cm, err := chart.GetChartIndexConfigMap(instance.Spec.Chart, helmRelease.Repo, watchNamespace, helmRelease.K8sClient)

	if err != nil {
		return helmRelease, err
	}

	cv, err := chart.GetChartVersionFromIndexConfigmap(helmRelease.Version, cm)

	if err != nil {
		return helmRelease, err
	}

	hc := &helmchart.Chart{}

	options := &action.ChartPathOptions{
		Version:               cv.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}

	defaultValues := chartversion.GetDefaultValuesFromConfigMap(helmRelease.Name, helmRelease.Repo, cv.Version, watchNamespace, helmRelease.K8sClient, helmRelease.logger)

	parsedValues := utils.MergeMaps(defaultValues, specValues)

	if err := chart.LoadChartByResources(helmRelease.K8sClient, helmRelease.logger, hc, cv, instance.Spec.Chart, instance.Spec.Repo, watchNamespace, options, parsedValues); err != nil {
		return helmRelease, err
	}

	if err := chart.LoadDependencies(hc, watchNamespace, utils.GetEnvSettings(map[string]string{}), scheme, helmRelease.logger, helmRelease.K8sClient); err != nil {
		return helmRelease, err
	}

	helmRelease.Chart = hc

	// func should be with pointer struct as input
	if err := helmRelease.validateChartSpecs(); err != nil {
		return helmRelease, err
	}

	return helmRelease, nil
}

// Update represents update or installation process of a release
func (hc *Release) Update() error {

	if hc.Chart == nil || hc.Chart.Metadata == nil {
		return errors.NewBadRequest("chart not loaded on action update")
	}

	hc.logger.Info("config install: "+fmt.Sprint(hc.Config), "name", hc.Name, "repo", hc.Repo)

	var release *release.Release
	var err error
	var ok bool

	installConfig := hc.Config

	hc.logger.Info("configupdate: "+fmt.Sprint(hc.Config), "name", hc.Name, "repo", hc.Repo)

	release, _ = hc.getRelease()

	// Check if something changed regarding the existing release
	if release != nil {
		if ok, err = hc.valuesChanged(); err != nil {
			return err
		}

		hc.Revision = release.Version

		if ok {
			if err := hc.upgrade(hc.Chart); err != nil {
				return err
			}
			hc.logger.Info("release updated.", "name", release.Name, "namespace", release.Namespace, "chart", hc.Chart.Name(), "repo", hc.Repo)
			return nil
		}

		hc.logger.Info("nothing changed for release.", "name", release.Name, "namespace", release.Namespace, "chart", hc.Chart.Name(), "repo", hc.Repo)
		return nil
	}

	client := action.NewInstall(installConfig)
	client.ReleaseName = hc.Name
	client.Namespace = hc.releaseNamespace
	client.CreateNamespace = false
	hc.setInstallFlags(client)

	if release, err = client.Run(hc.Chart, hc.ValuesTemplate.Values); err != nil {
		hc.logger.Error(err, "error on installing release", "release", hc.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
		return err
	}

	hc.Revision = release.Version

	hc.logger.Info("release successfully installed.", "name", release.Name, "namespace", release.Namespace, "chart", hc.Chart.Name(), "repo", hc.Repo)
	return nil
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

func (hc *Release) upgrade(helmChart *helmchart.Chart) error {
	var rel *release.Release
	var err error

	vals := hc.ValuesTemplate.Values

	client := action.NewUpgrade(hc.Config)
	client.Namespace = hc.releaseNamespace
	hc.setUpgradeFlags(client)

	if rel, err = client.Run(hc.Name, helmChart, vals); err != nil {
		hc.logger.Info(err.Error())
		return err
	}

	hc.Revision = rel.Version

	hc.logger.Info("successfully upgraded.", "name", rel.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
	return nil
}
