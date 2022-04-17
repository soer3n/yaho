package repository

import (
	"context"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (hr *Repo) deployChart(instance *helmv1alpha1.Repository, chart helmv1alpha1.Entry, selectors map[string]string, scheme *runtime.Scheme) error {

	c := &helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chart.Name,
			Namespace: hr.Namespace.Name,
			Labels:    selectors,
		},
		Spec: helmv1alpha1.ChartSpec{
			Name:       chart.Name,
			Versions:   chart.Versions,
			Repository: instance.ObjectMeta.Name,
			CreateDeps: true,
		},
	}

	if err := controllerutil.SetControllerReference(instance, c, scheme); err != nil {
		hr.logger.Error(err, "failed to set owner ref for chart", "chart", chart)
	}

	if err := hr.K8sClient.Create(context.Background(), c); err != nil {
		hr.logger.Info("error on chart create", "error", err.Error())
		if k8serrors.IsAlreadyExists(err) {
			if err := hr.K8sClient.Update(context.Background(), c); err != nil {
				hr.logger.Info("could not update chart resource", "chart", chart.Name)
				return err
			}
			hr.logger.Info("chart resource updated", "chart", chart.Name)
			return nil
		}
		hr.logger.Info("chart resource created", "chart", chart.Name)
	}

	return nil
}
