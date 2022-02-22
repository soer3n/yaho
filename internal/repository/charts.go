package repository

import (
	"context"
	"fmt"

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

func (hr *Repo) deployCharts(instance helmv1alpha1.Repo, selector map[string]string, scheme *runtime.Scheme) error {

	var err error

	chartChannel := make(chan *chart.Chart)
	mapChannel := make(chan *helmv1alpha1.Chart)
	c := make(chan error, 1)

	hr.wg.Add(4)

	// request charts from kubernetes api or if not present download it
	go func() {
		defer hr.mu.Unlock()
		defer hr.wg.Done()
		hr.mu.Lock()
		if err = hr.getCharts(selector, chartChannel, mapChannel); err != nil {
			hr.logger.Info("Error on getting charts", "repo", hr.Name)
		}
		close(chartChannel)
	}()

	// transform internal struct to custom resource
	go func() {
		defer hr.wg.Done()

		for chart := range chartChannel {
			hr.mu.Lock()
			if err = hr.transformChart(chart, instance, mapChannel); err != nil {
				hr.logger.Error(err, "error on updating chart version", "repo", hr.Name)
			}
			hr.mu.Unlock()
		}
		close(mapChannel)
	}()

	// goroutine 3
	go func() {
		defer hr.wg.Done()
		for chartObj := range mapChannel {
			hr.wg.Add(1)
			// deploy send object
			go func(chartObj *helmv1alpha1.Chart, instance helmv1alpha1.Repo, c chan<- error) {
				defer hr.wg.Done()
				if err := hr.updateChart(chartObj, instance, c, scheme); err != nil {
					hr.logger.Error(err, "error on managing chart", "chart", chartObj.Name)
				}
			}(chartObj, instance, c)
		}
		close(c)
	}()

	// goroutine 4 for printing errors
	go func() {
		defer hr.wg.Done()
		for i := range c {
			hr.logger.Error(i, "failed chart loading")
		}
	}()

	hr.wg.Wait()

	return nil
}

// GetCharts represents returning list of internal chart structs for a given repo
func (hr *Repo) getCharts(selectors map[string]string, chartChannel chan *chart.Chart, mapChannel chan *helmv1alpha1.Chart) error {

	var chartList []*chart.Chart
	var indexFile *repo.IndexFile
	var chartAPIList helmv1alpha1.ChartList
	var err error

	selectorObj := client.MatchingLabels{}

	for k, selector := range selectors {
		selectorObj[k] = selector
	}

	if err = hr.K8sClient.List(context.Background(), &chartAPIList, client.InNamespace(hr.Namespace.Name), selectorObj); err != nil {
		return err
	}

	if len(chartAPIList.Items) > 0 {
		for _, v := range chartAPIList.Items {
			mapChannel <- &v
		}
		close(mapChannel)
		close(chartChannel)
		return nil
	}

	if chartList == nil {

		if indexFile, err = hr.getIndexByURL(); err != nil {
			hr.logger.Error(err, "error on getting repo index file")
			return err
		}

		if indexFile == nil {
			return nil
		}

		for k, chartMetadata := range indexFile.Entries {
			chartChannel <- chart.New(k, hr.URL, chartMetadata, hr.Settings, hr.logger, hr.Name, hr.K8sClient, hr.getter, kube.Client{
				Factory: cmdutil.NewFactory(hr.Settings.RESTClientGetter()),
				Log:     nopLogger,
			})
			hr.logger.Info("initializing chart struct by metadata", "repo", hr.Name, "chart", k)
		}
		close(chartChannel)
	}

	return nil
}

func (hr *Repo) updateChart(helmChart *helmv1alpha1.Chart, instance helmv1alpha1.Repo, c chan<- error, scheme *runtime.Scheme) error {

	repo := instance.DeepCopy()
	if err := controllerutil.SetControllerReference(repo, helmChart, scheme); err != nil {
		fmt.Println(err)
	}

	installedChart := &helmv1alpha1.Chart{}
	err := hr.K8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: helmChart.ObjectMeta.Namespace,
		Name:      helmChart.Spec.Name,
	}, installedChart)
	if err != nil {
		if errors.IsNotFound(err) {

			if err = hr.K8sClient.Create(context.TODO(), helmChart); err != nil {
				c <- err
			}
		}

		return nil
	}

	installedChart.Spec = helmChart.Spec

	err = hr.K8sClient.Update(context.TODO(), installedChart)

	if err != nil {
		c <- err
	}

	return nil
}

// AddOrUpdateChartMap represents update of a map of chart structs if needed
func (hr *Repo) transformChart(instance *chart.Chart, repo helmv1alpha1.Repo, mapChannel chan *helmv1alpha1.Chart) error {

	apiObj := &helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hr.Name,
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

	mapChannel <- apiObj
	return nil
}
