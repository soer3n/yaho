package chart

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func setFiles(mu *sync.Mutex, helmChart *chart.Chart, chartName string, chartPathOptions *action.ChartPathOptions, logger logr.Logger, c client.WithWatch) {
	defer mu.Unlock()
	mu.Lock()

	d := make(chan *chart.File)
	t := make(chan *chart.File)
	quit := make(chan bool)
	counter := 0

	helmChart.Files = []*chart.File{}
	helmChart.Templates = []*chart.File{}

	go getFiles(chartPathOptions.Version, chartName, c, logger, quit, d, t)

	for {
		select {
		case i := <-d:
			helmChart.Files = append(helmChart.Files, i)
		case j := <-t:
			helmChart.Templates = append(helmChart.Templates, j)
		case <-quit:
			counter++
			if counter == 3 {
				close(d)
				close(t)
				return
			}
		}
	}
}

func getFiles(chartVersion, chartName string, c client.Client, logger logr.Logger, quit chan bool, f chan *chart.File, t chan *chart.File) {

	appendFilesFromConfigMap(chartName, chartVersion, "tmpl", c, logger, quit, f)
	appendFilesFromConfigMap(chartName, chartVersion, "tmpl", c, logger, quit, t)
	appendFilesFromConfigMap(chartName, chartVersion, "crds", c, logger, quit, f)
}

func appendFilesFromConfigMap(chartName, version, suffix string, c client.Client, logger logr.Logger, quit chan bool, channels ...chan *chart.File) {
	var err error

	configmapList := v1.ConfigMapList{}

	selector := labels.NewSelector()
	requirement, _ := labels.NewRequirement(configMapLabelKey, selection.Equals, []string{chartName + "-" + version})
	selector = selector.Add(*requirement)
	requirement, _ = labels.NewRequirement(configMapLabelType, selection.Equals, []string{suffix})
	selector = selector.Add(*requirement)

	if err = c.List(context.Background(), &configmapList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		logger.Info("error on listing files configmaps", "chart", chartName)
		quit <- true
		return
	}

	for _, configmap := range configmapList.Items {
		for key, data := range configmap.BinaryData {
			baseName := "templates/"
			if suffix == "crds" {
				baseName = "crds/"
			}

			if configmap.ObjectMeta.Labels[configMapLabelSubName] != "" {
				baseName = baseName + configmap.ObjectMeta.Labels[configMapLabelSubName] + "/"
			}

			for _, f := range channels {
				f <- &chart.File{
					Name: baseName + key,
					Data: data,
				}
			}
		}
	}
	quit <- true
}
