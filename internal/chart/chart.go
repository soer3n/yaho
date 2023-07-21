package chart

import (
	"errors"
	"net/http"
	"strings"
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
	"k8s.io/klog/v2"
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
	currentStatus := status.DeepCopy()

	chart := &Chart{
		Name:      name,
		Repo:      repository,
		Namespace: namespace,
		Status: &ChartStatus{
			Conditions:    &currentStatus.Conditions,
			ChartVersions: currentStatus.ChartVersions,
		},
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
			Reason:             "chartInitialization",
			Message:            err.Error(),
		}
		meta.SetStatusCondition(chart.Status.Conditions, condition)
		return nil, err
	}

	condition := metav1.Condition{
		Type:               "indexLoaded",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "chartInitialization",
		Message:            "successfully loaded",
	}
	meta.SetStatusCondition(chart.Status.Conditions, condition)

	chart.helm.index = *ix
	chart.kubernetes.scheme = scheme

	return chart, nil
}

func (c *Chart) Update(instance *yahov1alpha2.Chart) error {

	c.logger.Info("set versions")
	if err := c.setVersions(instance); err != nil {
		c.logger.Info(err.Error())
		c.setConditions(instance, &chart.Chart{}, false)
		return err
	}

	c.setConditions(instance, &chart.Chart{}, false)

	for name := range c.Status.ChartVersions {

		hc := &chart.Chart{}
		var currentVersion *repo.ChartVersion

		chartUrl, err := c.getChartURL(name, instance.Spec.Repository)

		if err != nil {
			c.setConditions(instance, &chart.Chart{}, false)
			return err
		}

		for _, ix := range c.helm.index {
			if ix.Version == name {
				currentVersion = ix
			}
		}

		if err := c.prepareVersion(hc, currentVersion, chartUrl); err != nil {
			c.setConditions(instance, hc, false)
			return err
		}

		// compare and manage charts
		c.logger.Info("create or update configmaps", "version", currentVersion.Version)
		if err := ManageSubResources(hc, currentVersion, c.Repo, c.Namespace, c.kubernetes.client, c.kubernetes.client, c.kubernetes.scheme, c.logger); err != nil {
			c.setConditions(instance, hc, false)
			return err
		}

		c.Status.ChartVersions[currentVersion.Version] = yahov1alpha2.ChartVersion{
			Loaded:    true,
			Specified: c.Status.ChartVersions[currentVersion.Version].Specified,
		}
		c.setConditions(instance, hc, true)
	}
	return nil
}

func (c *Chart) setConditions(instance *yahov1alpha2.Chart, hc *chart.Chart, configmapsCreated bool) {

	// TODO: set initial expected "good" status
	meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
		Type:               "prepareChart",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "chartUpdate",
		Message:            "success",
	})
	meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
		Type:               "configmapCreate",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "chartUpdate",
		Message:            "success",
	})
	meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
		Type:               "remoteSync",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "chartUpdate",
		Message:            "success",
	})

	// set versions func....
	if len(c.Status.ChartVersions) < 1 {
		if len(instance.Spec.Versions) < 1 {
			message := "no versions set."
			meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
				Type:               "prepareChart",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "chartUpdate",
				Message:            message,
			})
		} else {
			message := "no versions from spec are synced."
			meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
				Type:               "prepareChart",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "chartUpdate",
				Message:            message,
			})
		}
	} else {
		for k, item := range c.Status.ChartVersions {
			if !item.Loaded {
				message := "version " + k + " currently not loaded."
				meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
					Type:               "prepareChart",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             "chartUpdate",
					Message:            message,
				})
			}
		}

		for _, item := range instance.Spec.Versions {
			if _, ok := c.Status.ChartVersions[item]; !ok {
				message := "specified version " + item + " currently not loaded."
				meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
					Type:               "prepareChart",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             "chartUpdate",
					Message:            message,
				})
			}
		}
	}

	// ManageSubResources func...
	if !configmapsCreated {
		message := "could not create resources."
		meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
			Type:               "configmapCreate",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "chartUpdate",
			Message:            message,
		})
	}

	remoteSync := metav1.ConditionTrue
	remoteSyncMessage := "success"

	for _, versionStatus := range c.Status.ChartVersions {
		if !versionStatus.Loaded {
			remoteSync = metav1.ConditionFalse
			remoteSyncMessage = "not all is synced."
		}
	}

	meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
		Type:               "remoteSync",
		Status:             remoteSync,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "chartUpdate",
		Message:            remoteSyncMessage,
	})

	// prepare version func....
	if hc.Metadata == nil {
		message := "could not load chart metadata."
		meta.SetStatusCondition(c.Status.Conditions, metav1.Condition{
			Type:               "configmapCreate",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "chartUpdate",
			Message:            message,
		})

		return
	}

	linkedCharts := []string{}

	for _, chart := range hc.Metadata.Dependencies {
		repo, err := GetRepositoryNameByUrl(chart.Repository, c.kubernetes.client)

		if err != nil {
			c.logger.Info("could get repo name by url for linked chart", "chart", c.Name, "linked_chart", chart.Name, "repository_url", chart.Repository)
		}
		linkedCharts = append(linkedCharts, repo+"/"+chart.Name)
	}

	c.Status.LinkedCharts = linkedCharts
	c.Status.Deprecated = hc.Metadata.Deprecated

}

func (c *Chart) setVersions(instance *yahov1alpha2.Chart) error {

	// var chartVersions ChartVersions

	if c.Status.ChartVersions == nil {
		c.Status.ChartVersions = make(map[string]yahov1alpha2.ChartVersion)
	}

	for value, version := range instance.Status.ChartVersions {
		requested := false
		for _, requestedVersion := range instance.Spec.Versions {
			if version.Specified && value == requestedVersion {
				requested = true
			}
		}

		if !requested {
			c.logger.Info("remove version from status map because it is no longer requested...", "version", version, "chart", c.Name)
			delete(c.Status.ChartVersions, value)
		}
	}

	for _, version := range instance.Spec.Versions {
		c.logger.Info("init rendering version ...", "version", version, "chart", c.Name)

		parsedVersion := version
		var err error

		if strings.Contains(version, "*") || strings.Contains(version, "x") {
			c.logger.Info("rendering placeholder in version ...", "version", version, "chart", c.Name)
			parsedVersion, err = getParsedVersion(version, c.helm.index)

			if err != nil {
				return err
			}

			c.logger.Info("successfully found version ...", "version", parsedVersion, "chart", c.Name)
		}

		if _, ok := c.Status.ChartVersions[parsedVersion]; !ok {
			c.Status.ChartVersions[parsedVersion] = yahov1alpha2.ChartVersion{
				Loaded:    false,
				Specified: true,
			}
		}

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

	if hc == nil {
		hc = &chart.Chart{}
	}

	if err := LoadChartByResources(c.kubernetes.client, c.logger, hc, v, c.Name, c.Repo, c.Namespace, options, map[string]interface{}{}); err != nil {
		return err
	}

	if len(hc.Files) < 1 {
		if err := LoadChartByURL(c.Name, chartUrl, releaseClient, c.getter, hc); err != nil {
			return err
		}

	}

	for _, dep := range v.Dependencies {
		tempChart := &chart.Chart{}

		repositoryName, err := GetRepositoryNameByUrl(dep.Repository, c.kubernetes.client)

		if err != nil {
			c.logger.Error(err, "error getting repository name by url", "name", dep.Name)
			continue
		}

		cvi, err := utils.LoadChartIndex(dep.Name, repositoryName, c.Namespace, c.kubernetes.client)

		if err != nil {
			c.logger.Error(err, "error getting repository chart version for dependency", "name", c.Name, "dependency", dep.Name)
			continue
		}

		var cv *repo.ChartVersion

		currentVersion, _ := getParsedVersion(dep.Version, *cvi)

		for _, i := range *cvi {
			if i.Version == currentVersion {
				cv = i
				break
			}
		}

		if err := LoadChartByResources(c.kubernetes.client, c.logger, tempChart, cv, dep.Name, repositoryName, c.Namespace, options, map[string]interface{}{}); err != nil {
			return err
		}

		if len(tempChart.Files) > 0 {
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

func ManageSubResources(hc *chart.Chart, v *repo.ChartVersion, repository, namespace string, localClient, remoteClient client.WithWatch, scheme *runtime.Scheme, logger logr.Logger) error {
	cmChannel := make(chan v1.ConfigMap)
	wg := &sync.WaitGroup{}
	errList := []error{}

	wg.Add(2)
	logger.Info("parse and deploy configmaps")

	go func() {
		if err := chartversion.ParseConfigMaps(cmChannel, hc, v, repository, namespace, logger); err != nil {
			close(cmChannel)
			logger.Error(err, "error on parsing affected resources")
		}
		wg.Done()
	}()

	go func() {
		for configmap := range cmChannel {
			if err := chartversion.DeployConfigMap(configmap, hc, v, repository, namespace, localClient, remoteClient, scheme, logger); err != nil {
				errList = append(errList, err)
				logger.Error(err, "error on configmap deployment", "configmap", configmap.ObjectMeta.Name)
			}
		}
		wg.Done()
	}()

	wg.Wait()

	if err := errors.Join(errList...); err != nil {
		return err
	}

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

	hc.Files = c.Files
	hc.Templates = c.Templates
	hc.Values = c.Values

	return nil
}

func LoadChartByResources(c client.WithWatch, logger logr.Logger, helmChart *chart.Chart, v *repo.ChartVersion, chartName, repository, namespace string, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) error {

	mu := &sync.Mutex{}
	wg := sync.WaitGroup{}
	klog.V(0).Infof("load chart %s by cluster configmaps.\n Options: %v \nChart Version: %v", chartName, chartPathOptions, *v)
	wg.Add(3)

	go func() {
		defer wg.Done()
		setVersion(mu, helmChart, *v, chartName, chartPathOptions)
	}()

	go func() {
		defer wg.Done()
		setValues(mu, helmChart, chartName, repository, namespace, chartPathOptions, logger, c, vals)
		klog.V(0).Infof("parsed values by cluster configmaps for chart %s:\n %v", chartName, len(helmChart.Values))
	}()

	go func() {
		defer wg.Done()
		setFiles(mu, helmChart, chartName, chartPathOptions, logger, c)
		klog.V(0).Infof("parsed files by cluster configmaps for chart %s:\n %v", chartName, len(helmChart.Files))
		klog.V(0).Infof("parsed templates by cluster configmaps for chart %s:\n %v", chartName, len(helmChart.Templates))
	}()

	wg.Wait()

	if len(helmChart.Files) < 1 {
		logger.Info("no files detected for chart resource")
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
