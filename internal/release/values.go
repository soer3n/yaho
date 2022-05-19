package release

import (
	"reflect"

	"helm.sh/helm/v3/pkg/action"
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

func (hc *Release) getInstalledValues() (map[string]interface{}, error) {
	client := action.NewGetValues(hc.Config)
	return client.Run(hc.Name)
}

func (hc *Release) valuesChanged() (bool, error) {
	var installedValues map[string]interface{}
	var err error

	vals := hc.ValuesTemplate.Values

	hc.logger.Info("new values", "object", vals)

	if installedValues, err = hc.getInstalledValues(); err != nil {
		return false, err
	}

	hc.logger.Info("installed values", "object", installedValues)
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
