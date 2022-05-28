package chartversion

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"sync"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const configMapLabelKey = "helm.soer3n.info/chart"
const configMapRepoLabelKey = "helm.soer3n.info/repo"
const configMapLabelType = "helm.soer3n.info/type"
const configMapLabelSubName = "helm.soer3n.info/subname"

func New(version, namespace string, chartObj *helmv1alpha1.Chart, vals chartutil.Values, index repo.ChartVersions, scheme *runtime.Scheme, logger logr.Logger, k8sclient client.WithWatch, g utils.HTTPClientInterface) (*ChartVersion, error) {

	obj := &ChartVersion{
		mu:        sync.Mutex{},
		wg:        sync.WaitGroup{},
		owner:     chartObj,
		namespace: namespace,
		scheme:    scheme,
		k8sClient: k8sclient,
		logger:    logger,
		getter:    g,
	}

	parsedVersion, err := obj.getParsedVersion(version, index)

	if err != nil {
		obj.logger.Info("could not parse semver version", "version", version)
		return nil, err
	}

	for _, cv := range index {
		if cv.Version == parsedVersion {
			obj.Version = cv
			obj.Version.Version = parsedVersion
			break
		}
	}

	if obj.Version == nil {
		return obj, errors.New("chart version is not valid")
	}

	repo, err := obj.getControllerRepo()

	if err != nil {
		logger.Info(err.Error())
		return obj, err
	}

	obj.repo = repo

	if err := obj.setChartURL(index); err != nil {
		return obj, err
	}

	options := &action.ChartPathOptions{
		Version:               version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}

	if vals == nil {
		vals = obj.getDefaultValuesFromConfigMap(chartObj.Name, parsedVersion)
	}

	c, err := obj.getChart(chartObj.Spec.Name, options, vals)

	if err != nil {
		obj.logger.Info(err.Error())
	}

	obj.Obj = c

	if err := obj.addDependencies(); err != nil {
		obj.logger.Info(err.Error())
	}

	return obj, nil
}

func (chartVersion *ChartVersion) Prepare(config *action.Configuration) error {

	releaseClient := action.NewInstall(config)

	if chartVersion.Obj == nil {
		chartVersion.logger.Info("load chart obj")
		err := chartVersion.loadChartByURL(releaseClient)

		if err != nil {
			return err
		}
	}

	if err := chartVersion.addDependencies(); err != nil {
		return err
	}

	return nil
}

func (chartVersion *ChartVersion) ManageSubResources() error {
	cmChannel := make(chan v1.ConfigMap)

	chartVersion.wg.Add(2)
	chartVersion.logger.Info("parse and deploy configmaps")

	go func() {
		if err := chartVersion.parseConfigMaps(cmChannel); err != nil {
			close(cmChannel)
			chartVersion.logger.Error(err, "error on parsing affected resources")
		}
		chartVersion.wg.Done()
	}()

	go func() {
		for configmap := range cmChannel {
			if err := chartVersion.deployConfigMap(configmap); err != nil {
				chartVersion.logger.Error(err, "error on creating configmap", "configmap", configmap.ObjectMeta.Name)
			}
		}
		chartVersion.wg.Done()
	}()

	chartVersion.wg.Wait()

	return nil
}

func (chartVersion *ChartVersion) CreateOrUpdateSubCharts() error {

	for _, e := range chartVersion.deps {
		chartVersion.logger.Info("create or update child chart", "child", e.Name, "version", e.Version)
		if err := chartVersion.createOrUpdateSubChart(e); err != nil {
			chartVersion.logger.Info("failed to manage subchart", "chart", e.Name, "error", err.Error())
			return err
		}
	}

	return nil
}

func (chartVersion *ChartVersion) getControllerRepo() (*helmv1alpha1.Repository, error) {
	instance := &helmv1alpha1.Repository{}

	if chartVersion.owner == nil {
		return instance, errors.New("chart api resource not present")
	}

	err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{
		Name: chartVersion.owner.Spec.Repository,
	}, instance)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			chartVersion.logger.Info("HelmRepo resource not found.", "name", chartVersion.owner.Spec.Repository)
			return instance, err
		}
		// Error reading the object - requeue the request.
		chartVersion.logger.Error(err, "Failed to get ControllerRepo")
		return instance, err
	}

	return instance, nil
}

func (chartVersion *ChartVersion) setValues(helmChart *chart.Chart, apiObj *helmv1alpha1.Chart, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) {
	defer chartVersion.mu.Unlock()
	chartVersion.mu.Lock()

	if helmChart == nil {
		helmChart = &chart.Chart{}
	}

	obj := &chart.Chart{}
	obj.Metadata = &chart.Metadata{
		Name: apiObj.Spec.Name,
	}
	defaultValues := chartVersion.getDefaultValuesFromConfigMap(apiObj.Spec.Name, chartPathOptions.Version)
	obj.Values = defaultValues
	cv := values.MergeValues(vals, obj)
	helmChart.Values = cv
}

func (chartVersion *ChartVersion) setVersion(helmChart *chart.Chart, apiObj *helmv1alpha1.Chart, chartPathOptions *action.ChartPathOptions) {
	defer chartVersion.mu.Unlock()
	chartVersion.mu.Lock()

	if helmChart == nil {
		helmChart = &chart.Chart{}
	}

	if helmChart.Metadata == nil {
		helmChart.Metadata = &chart.Metadata{}
	}

	helmChart.Metadata.Name = apiObj.Spec.Name
	helmChart.Metadata = chartVersion.Version.Metadata
	helmChart.Metadata.Version = chartPathOptions.Version
	helmChart.Metadata.APIVersion = chartVersion.Version.Metadata.APIVersion
}

func (chartVersion *ChartVersion) getCredentials() *Auth {
	secret := chartVersion.repo.Spec.AuthSecret
	namespace := chartVersion.namespace
	secretObj := &v1.Secret{}
	creds := &Auth{}

	if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: secret}, secretObj); err != nil {
		return nil
	}

	if _, ok := secretObj.Data["user"]; !ok {
		chartVersion.logger.Info("Username empty for repo auth")
	}

	if _, ok := secretObj.Data["password"]; !ok {
		chartVersion.logger.Info("Password empty for repo auth")
	}

	username, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["user"]))
	pw, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["password"]))
	creds.User = string(username)
	creds.Password = string(pw)

	return creds
}

func (chartVersion *ChartVersion) createOrUpdateSubChart(dep *helmv1alpha1.ChartDep) error {

	chartVersion.logger.Info("fetching chart related to release resource")

	charts := &helmv1alpha1.ChartList{}
	labelSetRepo, _ := labels.ConvertSelectorToLabelsMap("repo=" + dep.Repo)
	labelSetChart, _ := labels.ConvertSelectorToLabelsMap("chart=" + dep.Name)
	ls := labels.Merge(labelSetRepo, labelSetChart)

	chartVersion.logger.Info("selector", "labelset", ls)

	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(ls),
	}

	if err := chartVersion.k8sClient.List(context.Background(), charts, opts); err != nil {
		return err
	}

	var group *string

	if len(charts.Items) == 0 {
		chartVersion.logger.Info("chart not found")

		obj := &helmv1alpha1.Chart{
			ObjectMeta: metav1.ObjectMeta{
				Name: dep.Name + "-" + dep.Repo,
			},
			Spec: helmv1alpha1.ChartSpec{
				Name:       dep.Name,
				Repository: dep.Repo,
				CreateDeps: true,
				Versions:   []string{dep.Version},
			},
		}

		if obj.ObjectMeta.Labels == nil {
			obj.ObjectMeta.Labels = map[string]string{}
		}

		if v, ok := chartVersion.owner.ObjectMeta.Labels["repoGroup"]; ok {
			group = &v
		}

		if group != nil {
			obj.ObjectMeta.Labels["repoGroup"] = *group
		}

		obj.ObjectMeta.Labels["repo"] = dep.Repo
		obj.ObjectMeta.Labels["unmanaged"] = "true"

		if err := controllerutil.SetControllerReference(chartVersion.repo, obj, chartVersion.scheme); err != nil {
			return err
		}

		if err := chartVersion.k8sClient.Create(context.TODO(), obj); err != nil {
			return err
		}

		if !chartVersion.watchForSubResourceSync(obj) {
			return errors.New("subresource" + obj.ObjectMeta.Name + "not synced")

		}

		return nil
	}

	current := &charts.Items[0]
	// group = nil

	if utils.Contains(current.Spec.Versions, dep.Version) {
		return nil
	}

	current.Spec.Versions = append(current.Spec.Versions, dep.Version)

	if err := chartVersion.k8sClient.Update(context.TODO(), current); err != nil {
		return err
	}

	if !chartVersion.watchForSubResourceSync(current) {
		return errors.New("subresource" + current.ObjectMeta.Name + "not synced")
	}

	return nil
}

func (chartVersion *ChartVersion) watchForSubResourceSync(subResource *helmv1alpha1.Chart) bool {

	r := &helmv1alpha1.ChartList{
		Items: []helmv1alpha1.Chart{
			*subResource,
		},
	}

	watcher, err := chartVersion.k8sClient.Watch(context.Background(), r)

	if err != nil {
		chartVersion.logger.Info("cannot get watcher for subresource")
		return false
	}

	defer watcher.Stop()

	select {
	case res := <-watcher.ResultChan():
		ch := res.Object.(*helmv1alpha1.Chart)

		if res.Type == watch.Modified {

			synced := "synced"
			if *ch.Status.Dependencies == synced && *ch.Status.Versions == synced {
				return true
			}
		}
	case <-time.After(10 * time.Second):
		return false
	}

	return false
}

func (chartVersion *ChartVersion) getParsedVersion(version string, index repo.ChartVersions) (string, error) {

	var constraint *semver.Constraints
	var v *semver.Version
	var err error

	current, _ := semver.NewVersion("0.0.0")

	if constraint, err = semver.NewConstraint(version); err != nil {
		return "", err
	}

	for _, e := range index {
		if v, err = semver.NewVersion(e.Version); err != nil {
			return "", err
		}

		if constraint.Check(v) && v.GreaterThan(current) {
			current = v
			continue
		}
	}

	return current.String(), nil
}
