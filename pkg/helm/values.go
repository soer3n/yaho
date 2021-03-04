package helm

import helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"

func NewValueTemplate(valuesList []*ValuesRef) *HelmValueTemplate {
	return &HelmValueTemplate{
		valuesRef: valuesList,
	}
}

func (hv *HelmValueTemplate) ManageValues() error {
	var base []*ValuesRef
	var values, subValues map[string]interface{}

	base = NewOptions(
		map[string]string{
			"parent": "base",
		}).
		Filter(hv.valuesRef)

	for _, ref := range base {
		if values, err := hv.manageStruct(ref); err != nil {
			return err

		}

		if err := hv.mergeMaps(values); err != nil {
			return err
		}
	}

	return nil
}

func (hv *HelmValueTemplate) manageStruct(valueMap *ValuesRef) (map[string]interface{}, error) {
	var valMap, merged map[string]interface{}
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

			merged = mergeMaps(hv.transformToMap(v.Ref), merged)

		}
	}

	valMap = hv.transformToMap(valueMap.Ref)

	return mergeMaps(valMap, merged), nil
}

func (hv *HelmValueTemplate) mergeMaps(valueMap map[string]interface{}) error {
	for k, v := range valueMap {
		hv.Values[k] = v
	}
	return nil

}

func (hv *HelmValueTemplate) transformToMap(values *helmv1alpha1.Values) map[string]interface{} {
	var valMap map[string]interface{}
	for k, v := range values.Spec.Values {
		valMap[k] = v
	}
	return valMap
}
