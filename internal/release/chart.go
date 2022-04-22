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

func (hc *Release) getChart(chartName, watchNamespace string, index repo.ChartVersions, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) (*helmchart.Chart, error) {

	chartObj := &helmv1alpha1.Chart{}

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Name: chartName,
	}, chartObj); err != nil {
		return nil, err
	}

	c, err := chartversion.New(hc.Version, watchNamespace, chartObj, vals, index, hc.scheme, hc.logger, hc.K8sClient, hc.getter)

	if err != nil {
		return nil, err
	}

	if c.Obj == nil {
		return nil, errors.NewBadRequest("could not load chart")
	}

	if len(c.Obj.Files) < 1 {
		return nil, errors.NewBadRequest("no files detected in chart struct")
	}

	return c.Obj, nil
}

func (hc *Release) getChartIndexConfigMap(chartName string) (*v1.ConfigMap, error) {
	indexMap := &v1.ConfigMap{}

	if err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: hc.Namespace.Name,
		Name:      "helm-" + hc.Repo + "-" + chartName + "-index",
	}, indexMap); err != nil {
		return indexMap, err
	}

	return indexMap, nil
}

func (hc *Release) getChartIndex(indexMap *v1.ConfigMap) (repo.ChartVersions, error) {

	var index repo.ChartVersions

	rawData := indexMap.BinaryData["versions"]

	if err := json.Unmarshal(rawData, &index); err != nil {
		hc.logger.Error(err, "error on marshaling chart index")
		return index, err
	}

	return index, nil
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
