package chart

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/chartversion"
	"github.com/soer3n/yaho/internal/utils"
	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
)

const configMapLabelKey = "yaho.soer3n.dev/chart"
const configMapRepoLabelKey = "yaho.soer3n.dev/repo"
const configMapRepoGroupLabelKey = "yaho.soer3n.dev/repoGroup"
const configMapLabelType = "yaho.soer3n.dev/type"
const configMapLabelSubName = "yaho.soer3n.dev/subname"
const configMapLabelUnmanaged = "yaho.soer3n.dev/unmanaged"

// New represents initialization of internal chart struct
func New(name, repository, namespace string, status *yahov1alpha2.ChartStatus, settings *cli.EnvSettings, scheme *runtime.Scheme, logger logr.Logger, k8sclient client.WithWatch, g utils.HTTPClientInterface, getter genericclioptions.RESTClientGetter, kubeconfig []byte) (*Chart, error) {

	var err error

	chart := &Chart{
		Name:       name,
		Repo:       repository,
		Namespace:  namespace,
		Status:     status.DeepCopy(),
		helm:       helm{},
		kubernetes: kubernetes{},
		logger:     logger,
		getter:     g,
	}

	logger.Info("init chart")
	config, err := utils.InitActionConfig(getter, kubeconfig, logger)

	if err != nil {
		logger.Info("Error on getting action config for chart")
		return nil, err
	}

	logger.Info("init metadata")
	chart.setMetadata(name, chart.Repo, namespace, config, settings, logger, k8sclient, g)

	chart.logger.Info("load chart struct")
	ix, err := utils.LoadChartIndex(chart.Name, chart.Repo, namespace, k8sclient)

	if err != nil {
		condition := metav1.Condition{
			Type:               "indexLoaded",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "chart initialization",
			Message:            err.Error(),
		}
		meta.SetStatusCondition(&chart.Status.Conditions, condition)
		return nil, err
	}

	condition := metav1.Condition{
		Type:               "indexLoaded",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "chart initialization",
		Message:            "successfully loaded",
	}
	meta.SetStatusCondition(&chart.Status.Conditions, condition)

	chart.helm.index = *ix
	chart.kubernetes.scheme = scheme

	return chart, nil
}

func (c *Chart) Update(instance *yahov1alpha2.Chart) error {

	c.logger.Info("set versions")
	if err := c.setVersions(instance, c.Namespace, c.kubernetes.scheme); err != nil {
		c.logger.Info(err.Error())
		return err
	}
	for name := range c.Status.ChartVersions {

		var hc *chart.Chart
		var currentVersion *repo.ChartVersion

		chartUrl, err := c.getChartURL(name, instance.Spec.Repository)

		if err != nil {
			return err
		}

		for _, ix := range c.helm.index {
			if ix.Version == name {
				currentVersion = ix
			}
		}

		if err := c.prepareVersion(hc, currentVersion, chartUrl); err != nil {
			condition := metav1.Condition{
				Type:               "prepareChart",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "chart update",
				Message:            err.Error(),
			}
			meta.SetStatusCondition(&c.Status.Conditions, condition)
			return err
		}

		condition := metav1.Condition{
			Type:               "prepareChart",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "chart update",
			Message:            "successful update",
		}
		meta.SetStatusCondition(&c.Status.Conditions, condition)

		// compare and manage charts
		c.logger.Info("create or update configmaps", "version", currentVersion.Version)
		if err := c.manageSubResources(hc, currentVersion, c.Repo); err != nil {
			condition := metav1.Condition{
				Type:               "configmapCreate",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "chart update",
				Message:            err.Error(),
			}
			meta.SetStatusCondition(&c.Status.Conditions, condition)
			return err
		}

		condition = metav1.Condition{
			Type:               "configmapCreate",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "chart update",
			Message:            "successful update",
		}
		meta.SetStatusCondition(&c.Status.Conditions, condition)
	}

	return nil
}

func (c *Chart) setVersions(instance *yahov1alpha2.Chart, namespace string, scheme *runtime.Scheme) error {

	// var chartVersions ChartVersions

	for _, version := range instance.Spec.Versions {
		c.logger.Info("init rendering version ...", "version", version, "chart", c.Name)
		if _, ok := c.Status.ChartVersions[version]; !ok {
			c.Status.ChartVersions[version] = yahov1alpha2.ChartVersion{
				Loaded:    false,
				Specified: true,
			}
			// set new version entry in status
		}
		/*
				c.logger.Info("init version struct", "version", version)
				obj, err := chartversion.New(version, namespace, instance.ObjectMeta.Name, instance.Spec.Name, instance.Spec.Repository, nil, c.index, scheme, c.logger, c.K8sClient, c.getter)

				if err != nil {
					c.logger.Info(err.Error(), "version", version)
					return err
				}

				if c.Status.Deprecated == nil {
					instance.Status.Deprecated = &obj.Version.Deprecated
				}

				chartVersions = append(chartVersions, obj)
			}

			c.Versions = chartVersions
		*/
	}
	return nil
}

func (c *Chart) prepareVersion(hc *chart.Chart, v *repo.ChartVersion, chartUrl string) error {
	c.logger.Info("prepare object", "version", v.Version)
	releaseClient := action.NewInstall(c.helm.config)

	// preparing helm chart struct...

	options := &action.ChartPathOptions{
		Version:               v.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}

	if err := LoadChartByResources(c.kubernetes.client, c.logger, hc, v, c.Name, c.Repo, c.Namespace, options, map[string]interface{}{}); err != nil {
		return err
	}

	if hc == nil {
		if err := LoadChartByURL(c.Name, chartUrl, releaseClient, c.getter, hc); err != nil {
			return err
		}

	}

	for _, dep := range v.Dependencies {
		var tempChart *chart.Chart

		repositoryName, err := getRepositoryNameByUrl(dep.Repository, c.kubernetes.client)

		if err != nil {
			c.logger.Error(err, "name %s", dep.Name)
			continue
		}

		if err := LoadChartByResources(c.kubernetes.client, c.logger, tempChart, v, dep.Name, repositoryName, c.Namespace, options, map[string]interface{}{}); err != nil {
			return err
		}

		if tempChart != nil {
			hc.AddDependency(tempChart)
			continue
		}
		if err := LoadChartByURL(c.Name, chartUrl, releaseClient, c.getter, tempChart); err != nil {
			return err
		}
		hc.AddDependency(tempChart)

	}

	return nil
}

func (c *Chart) manageSubResources(hc *chart.Chart, v *repo.ChartVersion, repository string) error {
	cmChannel := make(chan v1.ConfigMap)
	wg := &sync.WaitGroup{}

	wg.Add(2)
	c.logger.Info("parse and deploy configmaps")

	go func() {
		if err := chartversion.ParseConfigMaps(cmChannel, hc, v, repository, c.Namespace, c.logger); err != nil {
			close(cmChannel)
			c.logger.Error(err, "error on parsing affected resources")
		}
		wg.Done()
	}()

	go func() {
		for configmap := range cmChannel {
			if err := chartversion.DeployConfigMap(configmap, hc, v, repository, c.Namespace, c.kubernetes.client, c.kubernetes.scheme, c.logger); err != nil {
				c.logger.Error(err, "error on creating configmap", "configmap", configmap.ObjectMeta.Name)
			}
		}
		wg.Done()
	}()

	wg.Wait()

	return nil
}

func (c *Chart) setMetadata(name, repository, namespace string, config *action.Configuration, settings *cli.EnvSettings, logger logr.Logger, k8sclient client.WithWatch, g utils.HTTPClientInterface) {
	c.Name = name
	c.Namespace = namespace
	c.helm.config = config
	c.helm.client = action.NewInstall(config)
	c.helm.settings = settings
	c.Repo = repository
	c.kubernetes.client = k8sclient
	// URL:       chartURL,
	c.getter = g
	c.logger = logger.WithValues("repo", repository, "chart", name)
	// c.mu = &sync.Mutex{}
}

func LoadChartByURL(chartName, chartUrl string, releaseClient *action.Install, getter utils.HTTPClientInterface, hc *chart.Chart) error {

	releaseClient.ReleaseName = chartName

	if hc.Metadata.Version == "" {
		return errors.New("no chartversion loaded")
	}

	releaseClient.Version = hc.Metadata.Version

	releaseClient.ChartPathOptions.RepoURL = releaseClient.RepoURL
	credentials := &Auth{
		User:     releaseClient.Username,
		Password: releaseClient.Password,
		Ca:       releaseClient.CaFile,
		Cert:     releaseClient.CertFile,
		Key:      releaseClient.KeyFile,
	}

	c, err := downloadChart(credentials, chartUrl, getter)

	if err != nil {
		return err
	}

	hc = c

	return nil
}

func LoadChartByResources(c client.WithWatch, logger logr.Logger, helmChart *chart.Chart, v *repo.ChartVersion, chartName, repository, namespace string, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) error {

	mu := &sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		setVersion(mu, helmChart, *v, chartName, chartPathOptions)
	}()

	go func() {
		defer wg.Done()
		setValues(mu, helmChart, chartName, repository, namespace, chartPathOptions, logger, c, vals)
	}()

	go func() {
		defer wg.Done()
		setFiles(mu, helmChart, chartName, chartPathOptions, logger, c)
	}()

	wg.Wait()

	if len(helmChart.Files) < 1 {
		return errors.New("no files detected for chart resource")
	}

	// validate after channels are closed
	if err := helmChart.Validate(); err != nil {
		return err
	}

	return nil
}

// TODO: this func should be in the chart model
func downloadChart(opts *Auth, url string, getter utils.HTTPClientInterface) (*chart.Chart, error) {
	var resp *http.Response
	var err error

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if opts != nil {
		if opts.User != "" && opts.Password != "" {
			req.SetBasicAuth(opts.User, opts.Password)
		}
	}

	if resp, err = getter.Do(req); err != nil {
		return nil, err
	}

	c, err := loader.LoadArchive(resp.Body)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func setValues(mu *sync.Mutex, helmChart *chart.Chart, chartName, repository, namespace string, chartPathOptions *action.ChartPathOptions, logger logr.Logger, c client.WithWatch, vals map[string]interface{}) {
	defer mu.Unlock()
	mu.Lock()

	if helmChart == nil {
		helmChart = &chart.Chart{}
	}

	obj := &chart.Chart{}
	obj.Metadata = &chart.Metadata{
		Name: chartName,
	}
	defaultValues := chartversion.GetDefaultValuesFromConfigMap(chartName, repository, chartPathOptions.Version, namespace, c, logger)
	obj.Values = defaultValues
	cv := values.MergeValues(vals, obj)
	helmChart.Values = cv
}

func setVersion(mu *sync.Mutex, helmChart *chart.Chart, v repo.ChartVersion, chartName string, chartPathOptions *action.ChartPathOptions) {
	defer mu.Unlock()
	mu.Lock()

	if helmChart == nil {
		helmChart = &chart.Chart{}
	}

	if helmChart.Metadata == nil {
		helmChart.Metadata = &chart.Metadata{}
	}

	helmChart.Metadata = v.Metadata
	helmChart.Metadata.Version = chartPathOptions.Version
}
