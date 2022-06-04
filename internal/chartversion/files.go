package chartversion

import (
	"context"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (chartVersion *ChartVersion) setFiles(helmChart *chart.Chart, apiObj *helmv1alpha1.Chart, chartPathOptions *action.ChartPathOptions) {
	defer chartVersion.mu.Unlock()
	chartVersion.mu.Lock()

	c := make(chan *chart.File)
	t := make(chan *chart.File)
	quit := make(chan bool)
	counter := 0

	files := []*chart.File{}
	templates := []*chart.File{}

	go getFiles(chartPathOptions.Version, apiObj, chartVersion.k8sClient, chartVersion.logger, quit, c, t)

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

func getFiles(chartVersion string, helmChart *helmv1alpha1.Chart, c client.Client, logger logr.Logger, quit chan bool, f chan *chart.File, t chan *chart.File) {

	appendFilesFromConfigMap(helmChart.Spec.Name, chartVersion, "tmpl", c, logger, quit, f)
	appendFilesFromConfigMap(helmChart.Spec.Name, chartVersion, "tmpl", c, logger, quit, t)
	appendFilesFromConfigMap(helmChart.Spec.Name, chartVersion, "crds", c, logger, quit, f)
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
