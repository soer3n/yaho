package repository

import (
	"context"

	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (hr *Repo) deployChart(instance *yahov1alpha2.Repository, chart yahov1alpha2.Entry, scheme *runtime.Scheme) error {

	hr.logger.Info("fetching chart related to release resource")

	c := &yahov1alpha2.Chart{}
	charts := &yahov1alpha2.ChartList{}
	labelSetRepo, _ := labels.ConvertSelectorToLabelsMap(configMapRepoLabelKey + "=" + hr.Name)
	labelSetChart, _ := labels.ConvertSelectorToLabelsMap(configMapLabelKey + "=" + chart.Name)
	ls := labels.Merge(labelSetRepo, labelSetChart)

	hr.logger.Info("selector", "labelset", ls)

	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(ls),
	}

	if err := hr.K8sClient.List(context.Background(), charts, opts); err != nil {
		return err
	}

	if len(charts.Items) == 0 {
		hr.logger.Info("chart resource not present", "chart", chart.Name)
		c.ObjectMeta = metav1.ObjectMeta{
			Name:   chart.Name + "-" + hr.Name,
			Labels: ls,
		}

		c.Spec = yahov1alpha2.ChartSpec{
			Name:       chart.Name,
			Versions:   chart.Versions,
			Repository: instance.ObjectMeta.Name,
			CreateDeps: true,
		}

		if err := controllerutil.SetControllerReference(instance, c, scheme); err != nil {
			hr.logger.Error(err, "failed to set owner ref for chart", "chart", chart.Name)
		}

		if err := hr.K8sClient.Create(context.Background(), c); err != nil {
			hr.logger.Info("error on chart create", "error", err.Error())
		}

		hr.logger.Info("chart resource created", "chart", chart.Name)
		return nil
	}

	c = &charts.Items[0]
	c.Spec = yahov1alpha2.ChartSpec{
		Name:       chart.Name,
		Versions:   chart.Versions,
		Repository: instance.ObjectMeta.Name,
		CreateDeps: true,
	}

	if err := hr.K8sClient.Update(context.Background(), c); err != nil {
		hr.logger.Info("could not update chart resource", "chart", chart.Name, "error", err.Error())
		return err
	}
	hr.logger.Info("chart resource updated", "chart", chart.Name)

	return nil
}
