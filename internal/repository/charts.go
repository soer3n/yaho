package repository

import (
	"context"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (hr *Repo) deployCharts(instance helmv1alpha1.Repository, selector map[string]string, scheme *runtime.Scheme) error {

	var err error

	mapChannel := make(chan helmv1alpha1.Chart)
	hr.wg.Add(2)

	go func() {
		defer hr.wg.Done()
		for chartObj := range mapChannel {
			if err := hr.updateChart(chartObj, instance, scheme); err != nil {
				hr.logger.Error(err, "error on managing chart", "chart", chartObj.Name)
			}
		}
	}()

	go func() {
		defer hr.wg.Done()
		// request charts from kubernetes api or if not present download it
		if err = hr.getCharts(selector, instance, mapChannel); err != nil {
			hr.logger.Info("Error on getting charts", "repo", hr.Name)
		}
		close(mapChannel)
	}()

	hr.wg.Wait()

	return nil
}

// GetCharts represents returning list of internal chart structs for a given repo
func (hr *Repo) getCharts(selectors map[string]string, instance helmv1alpha1.Repository, mapChannel chan helmv1alpha1.Chart) error {

	var chartList []*chart.Chart
	var chartAPIList helmv1alpha1.ChartList
	var err error

	selectorObj := client.MatchingLabels{}

	for k, selector := range selectors {
		selectorObj[k] = selector
	}

	if err = hr.K8sClient.List(hr.ctx, &chartAPIList, client.InNamespace(hr.Namespace.Name), selectorObj); err != nil {
		return err
	}

	if len(chartAPIList.Items) > 0 {
		for _, v := range chartAPIList.Items {
			mapChannel <- v
		}
		return nil
	}

	if chartList == nil {

		for k, chartMetadata := range hr.loadRepositoryIndex().Entries {
			obj := chart.New(k, hr.URL, chartMetadata, hr.Settings, hr.logger, hr.Name, hr.K8sClient, hr.getter, kube.Client{
				Factory: cmdutil.NewFactory(hr.Settings.RESTClientGetter()),
				Log:     nopLogger,
			})

			hr.logger.Info("initializing chart struct by metadata", "repo", hr.Name, "chart", k)

			if err = hr.transformChart(obj, instance, mapChannel); err != nil {
				hr.logger.Error(err, "error on updating chart version", "repo", hr.Name)
			}
		}
	}

	return nil
}

func (hr *Repo) loadRepositoryIndex() *repo.IndexFile {
	indexFile, err := hr.getIndexByURL()

	if err != nil {
		hr.logger.Error(err, "error on getting repo index file")
	}

	return indexFile
}

func (hr *Repo) updateChart(helmChart helmv1alpha1.Chart, instance helmv1alpha1.Repository, scheme *runtime.Scheme) error {

	repo := instance.DeepCopy()
	owned := helmChart.DeepCopy()
	if err := controllerutil.SetControllerReference(repo, owned, scheme); err != nil {
		hr.logger.Error(err, "failed to set owner ref for chart", "chart", owned.Name)
	}

	installedChart := &helmv1alpha1.Chart{}
	err := hr.K8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: owned.ObjectMeta.Namespace,
		Name:      owned.Spec.Name,
	}, installedChart)
	if err != nil {
		if errors.IsNotFound(err) {

			if err = hr.K8sClient.Create(hr.ctx, owned); err != nil {
				hr.logger.Error(err, "failed to create chart resource")
			}
		}

		return nil
	}

	installedChart.Spec = owned.Spec

	err = hr.K8sClient.Update(hr.ctx, installedChart)

	if err != nil {
		hr.logger.Error(err, "failed to update chart resource")
	}

	return nil
}

// AddOrUpdateChartMap represents update of a map of chart structs if needed
func (hr *Repo) transformChart(instance *chart.Chart, repo helmv1alpha1.Repository, mapChannel chan helmv1alpha1.Chart) error {

	apiObj := &helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: hr.Namespace.Name,
			Labels: map[string]string{
				"chart":     instance.Name,
				"repo":      hr.Name,
				"repoGroup": repo.ObjectMeta.Labels["repoGroup"],
			},
		},
	}

	for _, version := range instance.Versions {
		if err := version.AddOrUpdateChartMap(instance.URL, apiObj); err != nil {
			return err
		}
	}

	mapChannel <- *apiObj.DeepCopy()
	return nil
}
