package release

import (
	"context"
	"encoding/json"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chartversion"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (hc *Release) setChart(chartName string, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) error {

	hc.Chart = &helmchart.Chart{}
	chartObj := &helmv1alpha1.Chart{}
	indexMap := &v1.ConfigMap{}
	var index repo.ChartVersions

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: hc.Namespace.Name,
		Name:      chartName,
	}, chartObj); err != nil {
		return err
	}

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: hc.Namespace.Name,
		Name:      "helm-" + hc.Repo + "-" + chartName + "-index",
	}, indexMap); err != nil {
		return err
	}

	rawData := indexMap.BinaryData["versions"]

	if err := json.Unmarshal(rawData, &index); err != nil {
		hc.logger.Error(err, "error on marshaling chart index")
		return err
	}

	c, err := chartversion.New(hc.Version, chartObj, vals, index, hc.scheme, hc.logger, hc.K8sClient, hc.getter)

	if err != nil {
		return err
	}

	if c.Obj == nil {
		return errors.NewBadRequest("could not load chart")
	}

	if len(c.Obj.Files) < 1 {
		return errors.NewBadRequest("no files detected in chart struct")
	}

	hc.Chart = c.Obj

	if err := hc.validateChartSpecs(); err != nil {
		return err
	}

	return nil
}

func (hc *Release) validateChartSpecs() error {

	if err := hc.Chart.Validate(); err != nil {
		return err
	}

	for _, d := range hc.Chart.Dependencies() {

		if err := d.Validate(); err != nil {
			return err
		}
	}

	return nil
}
