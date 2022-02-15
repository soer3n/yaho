package repository

import (
	"context"
	"fmt"
	"strings"
	"sync"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (hr *Repo) deployCharts(instance *helmv1alpha1.Repo, selector map[string]string, scheme *runtime.Scheme) error {

	var chartList []*chart.Chart
	var err error

	if chartList, err = hr.GetCharts(selector); err != nil {
		hr.logger.Info("Error on getting charts", "repo", instance.Spec.Name)
	}

	chartObjMap := make(map[string]*helmv1alpha1.Chart)

	for _, chart := range chartList {
		chartObjMap = chart.AddOrUpdateChartMap(chartObjMap, instance)
	}

	// this is use just for using channels, goroutines and waitGroup
	// could be senseful here if we have to deal with big repositories
	var wg sync.WaitGroup
	c := make(chan string, 1)

	for _, chartObj := range chartObjMap {
		wg.Add(1)

		go func(helmChart *helmv1alpha1.Chart, instance *helmv1alpha1.Repo, c chan<- string) {
			defer wg.Done()

			if err := controllerutil.SetControllerReference(instance, helmChart, scheme); err != nil {
				hr.logger.Error(err, "failed to set controller ref")
			}

			installedChart := &helmv1alpha1.Chart{}
			err := hr.K8sClient.Get(context.Background(), client.ObjectKey{
				Namespace: helmChart.ObjectMeta.Namespace,
				Name:      helmChart.Spec.Name,
			}, installedChart)
			if err != nil {
				if errors.IsNotFound(err) {
					hr.logger.Info("Trying to install HelmChart " + helmChart.Name)

					if err = hr.K8sClient.Create(context.TODO(), helmChart); err != nil {
						hr.logger.Info(err.Error())
						c <- installedChart.Spec.Name
					}
				}

				c <- ""
				return
			}

			installedChart.Spec = helmChart.Spec

			err = hr.K8sClient.Update(context.TODO(), installedChart)

			if err != nil {
				c <- installedChart.Spec.Name
			} else {
				hr.logger.Info("chart is up to date", "chart", installedChart.Spec.Name)
				c <- ""
			}
		}(chartObj, instance, c)
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	failedChartList := []string{}

	for i := range c {
		if i != "" {
			failedChartList = append(failedChartList, i)
		}
	}

	if len(failedChartList) > 0 {
		return fmt.Errorf("problems with charts: %v", strings.Join(failedChartList, ","))
	}

	return nil
}
