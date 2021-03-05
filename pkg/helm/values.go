package helm

import (
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

func NewValueTemplate(valuesList []*ValuesRef) *HelmValueTemplate {
	return &HelmValueTemplate{
		valuesRef: valuesList,
	}
}

func (hv *HelmValueTemplate) ManageValues() error {
	var base []*ValuesRef
	var values, merged map[string]interface{}
	var err error

	base = NewOptions(
		map[string]string{
			"parent": "base",
		}).
		Filter(hv.valuesRef)

	for _, ref := range base {
		if values, err = hv.manageStruct(ref); err != nil {
			return err

		}

		if hv.Values == nil {
			hv.Values = make(map[string]interface{})
		}

		merged = hv.transformToMap(ref.Ref, values)

		if err = hv.mergeMaps(merged); err != nil {
			return err
		}
	}

	return nil
}

func (hv *HelmValueTemplate) manageStruct(valueMap *ValuesRef) (map[string]interface{}, error) {
	valMap := make(map[string]interface{})
	merged := make(map[string]interface{})

	if valueMap.Ref.Spec.Refs != nil {
		temp := NewOptions(
			map[string]string{
				"parent": valueMap.Ref.ObjectMeta.Name,
			}).
			Filter(hv.valuesRef)

		for _, v := range temp {
			if v.Ref.Spec.Refs != nil {
				if merged, err := hv.manageStruct(v); err != nil {
					return merged, err
				}
			}

			merged = hv.transformToMap(v.Ref, merged, hv.getValuesAsList(valueMap.Ref.Spec.Refs))

		}

	}

	valMap = hv.transformToMap(valueMap.Ref, merged)

	return mergeMaps(valMap, merged), nil
}

func (hv HelmValueTemplate) getValuesAsList(values map[string]string) []string {

	valueList := []string{}

	for k, _ := range values {
		valueList = append(valueList, k)
	}

	return valueList
}

func (hv *HelmValueTemplate) mergeMaps(valueMap map[string]interface{}) error {
	temp := mergeMaps(hv.Values, valueMap)
	hv.Values = make(map[string]interface{})

	for k, _ := range temp {
		hv.Values[k] = temp[k].(string)
	}

	return nil

}

func (hv *HelmValueTemplate) transformToMap(values *helmv1alpha1.Values, childMap map[string]interface{}, parents ...[]string) map[string]interface{} {
	valMap := make(map[string]interface{})
	var parentKey string

	for _, parent := range parents {
		for _, child := range parent {
			parentKey = parentKey + child + "."
		}
	}

	for k, v := range values.Spec.Values {
		valMap[parentKey+k] = v
	}

	for ck, cv := range childMap {
		valMap[parentKey+ck] = cv
	}
	return valMap
}
