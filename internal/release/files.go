package release

import (
	"context"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (hc *Release) setFiles(chartName string, chartPathOptions *action.ChartPathOptions, helmChart *helmchart.Chart, chartObj *helmv1alpha1.Chart) {
	defer hc.mu.Unlock()
	hc.mu.Lock()

	c := make(chan *helmchart.File)
	t := make(chan *helmchart.File)
	quit := make(chan bool)
	counter := 0

	files := []*helmchart.File{}
	templates := []*helmchart.File{}

	go getFiles(chartName, chartPathOptions.Version, chartObj, hc.K8sClient, hc.logger, quit, c, t)

	for {
		select {
		case i := <-c:
			files = append(files, i)
		case j := <-t:
			templates = append(templates, j)
		case <-quit:
			counter++
			if counter == 3 {
				close(c)
				close(t)
				helmChart.Files = files
				helmChart.Templates = templates
				return
			}
		}
	}
}

func getFiles(chartName, chartVersion string, helmChart *helmv1alpha1.Chart, c client.Client, logger logr.Logger, quit chan bool, f chan *helmchart.File, t chan *helmchart.File) {

	appendFilesFromConfigMap(chartName, chartVersion, "tmpl", c, logger, quit, f)
	appendFilesFromConfigMap(chartName, chartVersion, "tmpl", c, logger, quit, t)
	appendFilesFromConfigMap(chartName, chartVersion, "crds", c, logger, quit, f)
}

func appendFilesFromConfigMap(chart, version, suffix string, c client.Client, logger logr.Logger, quit chan bool, channels ...chan *helmchart.File) {
	var err error

	configmapList := v1.ConfigMapList{}

	selector := labels.NewSelector()
	requirement, _ := labels.NewRequirement(configMapLabelKey, selection.Equals, []string{chart + "-" + version + "-" + suffix})
	selector = selector.Add(*requirement)

	if err = c.List(context.Background(), &configmapList, &client.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		logger.Info("error on listing files configmaps", "chart", chart)
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
				f <- &helmchart.File{
					Name: baseName + key,
					Data: data,
				}
			}
		}
	}
	quit <- true
}
