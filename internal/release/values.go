package release

import (
	"reflect"

	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
)

func (hc *Release) getValues() (map[string]interface{}, error) {
	templateObj := hc.ValuesTemplate

	returnValues, err := templateObj.ManageValues()
	if err != nil {
		return templateObj.Values, err
	}

	hc.ValuesTemplate.Values = templateObj.Values
	hc.ValuesTemplate.ValuesMap = templateObj.ValuesMap

	return returnValues, nil
}

func (hc *Release) setValues(chartName string, chartPathOptions *action.ChartPathOptions, helmChart *helmchart.Chart, vals map[string]interface{}) {
	defer hc.mu.Unlock()
	hc.mu.Lock()
	defaultValues := hc.getDefaultValuesFromConfigMap("helm-default-" + chartName + "-" + chartPathOptions.Version)
	hc.Chart.Values = defaultValues
	cv := values.MergeValues(vals, hc.Chart)
	helmChart.Values = cv
}

func (hc *Release) getInstalledValues() (map[string]interface{}, error) {
	client := action.NewGetValues(hc.Config)
	return client.Run(hc.Name)
}

func (hc *Release) valuesChanged(vals map[string]interface{}) (bool, error) {
	var installedValues map[string]interface{}
	var err error

	if installedValues, err = hc.getInstalledValues(); err != nil {
		return false, err
	}

	hc.logger.Info("values parsed", "name", hc.Name, "chart", hc.Chart.Name(), "repo", hc.Repo, "values length", len(installedValues))

	for key := range installedValues {
		if _, ok := vals[key]; !ok {
			hc.logger.Error(err, "missing key", "key", key)
		}
	}

	if len(vals) < 1 && len(installedValues) < 1 {
		return false, nil
	}

	if reflect.DeepEqual(installedValues, vals) {
		return false, nil
	}

	return true, nil
}
