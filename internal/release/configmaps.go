package release

import (
	"context"
	"encoding/json"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	"github.com/soer3n/yaho/internal/utils"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// parsedConfigMaps represents parsing and returning of chart related data for a release
func (hc *Release) parseConfigMaps(c chan helmv1alpha1.Chart, cm chan v1.ConfigMap) error {

	if hc.Chart.Metadata == nil {
		return errors.NewBadRequest("chart not loaded")
	}

	var chartObj helmv1alpha1.Chart
	var err error

	chartVersion := &chart.ChartVersion{}

	if err = hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: hc.Namespace.Name,
		Name:      hc.Chart.Name(),
	}, &chartObj); err != nil {
		return err
	}

	chartVersion.Version = &repo.ChartVersion{
		Metadata: &helmchart.Metadata{
			Name:    hc.Chart.Name(),
			Version: hc.Version,
		},
	}

	chartVersion.Templates = hc.Chart.Templates
	chartVersion.CRDs = hc.Chart.CRDs()
	chartVersion.DefaultValues = hc.Chart.Values
	deps := hc.Chart.Dependencies()
	version := utils.GetChartVersion(hc.Version, &chartObj)
	chartVersion.Version.Metadata.Version = version.Name

	go func() {
		for _, v := range version.Dependencies {
			if err := hc.validateChartSpec(c, deps, v); err != nil {
				hc.logger.Error(err, "error on validating dep chart")
			}
		}
		close(c)
	}()

	go func() {
		if err := chartVersion.CreateConfigMaps(cm, hc.Namespace.Name, deps); err != nil {
			hc.logger.Error(err, "error on creating or updating related resources")
		}
		close(cm)
	}()

	return nil
}

func (hc *Release) deployConfigMap(configmap v1.ConfigMap, instance *helmv1alpha1.Repo, scheme *runtime.Scheme) error {
	defer hc.mu.Unlock()
	hc.mu.Lock()
	if err := controllerutil.SetControllerReference(instance, &configmap, scheme); err != nil {
		return err
	}

	current := &v1.ConfigMap{}
	err := hc.K8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: configmap.ObjectMeta.Namespace,
		Name:      configmap.ObjectMeta.Name,
	}, current)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = hc.K8sClient.Create(context.TODO(), &configmap); err != nil {
				return err
			}
		}
		return err
	}

	return nil
}

func (hc *Release) updateChart(chart helmv1alpha1.Chart, instance *helmv1alpha1.Repo) error {
	current := &helmv1alpha1.Chart{}
	err := hc.K8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: chart.ObjectMeta.Namespace,
		Name:      chart.ObjectMeta.Name,
	}, current)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = hc.K8sClient.Create(context.TODO(), &chart); err != nil {
				return err
			}
		}
		return err
	}

	if err = hc.K8sClient.Update(context.TODO(), &chart); err != nil {
		return err
	}

	return nil
}

// UpdateAffectedResources represents parsing and installing subresources
func (hc *Release) UpdateAffectedResources(scheme *runtime.Scheme) error {

	var controller *helmv1alpha1.Repo

	if hc.Chart == nil {
		return errors.NewBadRequest("chart not loaded on action update affected resources")
	}

	controller, err := hc.getControllerRepo(hc.Repo, hc.Namespace.Name)

	if err != nil {
		return err
	}

	chartChannel := make(chan helmv1alpha1.Chart)
	cmChannel := make(chan v1.ConfigMap)
	hc.wg.Add(3)

	go func() {
		if err := hc.parseConfigMaps(chartChannel, cmChannel); err != nil {
			close(chartChannel)
			close(cmChannel)
			hc.logger.Error(err, "error on parsing affected resources")
		}
		hc.wg.Done()
	}()

	go func() {
		for chart := range chartChannel {
			if err := hc.updateChart(chart, controller); err != nil {
				hc.logger.Error(err, "error on updating chart", "chart", chart.Name)
			}
		}
		hc.wg.Done()
	}()

	go func() {
		for configmap := range cmChannel {
			if err := hc.deployConfigMap(configmap, controller, scheme); err != nil {
				hc.logger.Error(err, "error on creating configmap", "configmap", configmap.ObjectMeta.Name)
			}
		}
		hc.wg.Done()
	}()

	hc.wg.Wait()

	return nil
}

func (hc *Release) getDefaultValuesFromConfigMap(name string) map[string]interface{} {
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
