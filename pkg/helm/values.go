package helm

import (
	"encoding/json"
	"fmt"

	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"sigs.k8s.io/yaml"
)

func NewValueTemplate(valuesList []*ValuesRef) *HelmValueTemplate {
	return &HelmValueTemplate{
		valuesRef: valuesList,
	}
}

func (hv *HelmValueTemplate) ManageValues() (map[string]interface{}, error) {
	var base []*ValuesRef
	var values, merged map[string]interface{}
	var err error

	base = NewOptions(
		map[string]string{
			"parent": "base",
		}).
		Filter(hv.valuesRef)

	merged = make(map[string]interface{})

	for _, ref := range base {
		if values, err = hv.manageStruct(ref); err != nil {
			return merged, err

		}

		if hv.Values == nil {
			hv.Values = make(map[string]interface{})
		}

		merged = hv.transformToMap(ref.Ref, values)

		if err = hv.mergeMaps(merged); err != nil {
			return merged, err
		}
	}

	for k, merge := range merged {
		hv.ValuesMap[k] = merge.(string)
	}

	return merged, nil
}

func (hv *HelmValueTemplate) manageStruct(valueMap *ValuesRef) (map[string]interface{}, error) {
	valMap := make(map[string]interface{})
	var merged map[string]interface{}

	if valueMap.Ref.Spec.Refs != nil {
		temp := NewOptions(
			map[string]string{
				"parent": valueMap.Ref.ObjectMeta.Name,
			}).
			Filter(hv.valuesRef)

		for _, v := range temp {
			merged = make(map[string]interface{})
			if v.Ref.Spec.Refs != nil {
				if merged, err := hv.manageStruct(v); err != nil {
					return merged, err
				}
			}

			merged = hv.transformToMap(v.Ref, merged, hv.getRefKeyByValue(v.Ref.Name, valueMap.Ref.Spec.Refs))
			valMap = mergeMaps(merged, valMap)
		}

	}

	return valMap, nil
}

func (hv HelmValueTemplate) getRefKeyByValue(value string, refMap map[string]string) string {
	for k, v := range refMap {
		if value == v {
			return k
		}
	}

	return ""
}

func (hv HelmValueTemplate) getValuesAsList(values map[string]string) []string {

	valueList := []string{}

	for k := range values {
		valueList = append(valueList, k)
	}

	return valueList
}

func (hv *HelmValueTemplate) mergeMaps(valueMap map[string]interface{}) error {
	temp := mergeMaps(hv.Values, valueMap)
	hv.ValuesMap = make(map[string]string)

	for k, v := range temp {
		hv.ValuesMap[k] = v.(string)
	}

	return nil

}

func (hv HelmValueTemplate) transformToMap(values *helmv1alpha1.Values, childMap map[string]interface{}, parents ...string) map[string]interface{} {
	valMap := make(map[string]interface{})
	subMap := make(map[string]string)
	var parentKey string

	for _, parent := range parents {
		if parentKey != "" {
			parentKey = parentKey + "."
		}

		parentKey = parentKey + parent
	}

	rawVals := values.Spec.ValuesMap
	var convertedMap map[string]interface{}

	if rawVals != nil && rawVals.Raw != nil {
		if err := json.Unmarshal(rawVals.Raw, &convertedMap); err != nil {
			log.Debugf("error on parsing:%v", err)
			return valMap
		}

		log.Debugf("ConvertedMap: %v", convertedMap)

		if err := json.Unmarshal(rawVals.Raw, &convertedMap); err != nil {
			log.Debugf("error on parsing:%v", err)
			return valMap
		}

		log.Debugf("ConvertedMap: %v", convertedMap)

		for key, val := range hv.parseFromUntypedMap(parentKey, convertedMap) {
			valMap[key] = val
			log.Debugf("Parsed key: %v; Parsed value: %v", key, val)
		}
	}

	for ck, cv := range childMap {
		bytes := []byte(cv.(string))
		subMap = hv.parseMap(parentKey+ck, bytes)
		for key, value := range subMap {
			valMap[key] = value
		}
	}

	return valMap
}

func (hv HelmValueTemplate) parseMap(key string, payload []byte) map[string]string {

	valMap := make(map[string]string)
	subMap := make(map[string]string)
	returnKey := key

	if err := yaml.Unmarshal([]byte(payload), &subMap); err != nil {
		log.Debugf("Error: %v", err)
		valMap[key] = string(payload[:])
		return valMap
	}

	for ix, entry := range subMap {
		returnKey = returnKey + "." + ix
		if err := yaml.Unmarshal([]byte(entry), &subMap); err != nil {
			valMap[returnKey] = entry
		} else {
			return hv.parseMap(ix, []byte(entry))
		}
	}

	return valMap

}

func (hv HelmValueTemplate) parseFromUntypedMap(parentKey string, convertedMap map[string]interface{}) map[string]string {

	var targetMap map[string]string
	valMap := make(map[string]string)
	returnKey := parentKey

	for ix, entry := range convertedMap {

		if parentKey != "" {
			returnKey = returnKey + "."
		}
		returnKey = returnKey + ix
		stringVal, ok := entry.(string)

		if ok {
			valMap[returnKey] = stringVal
			returnKey = parentKey
			continue
		}

		stringVal = fmt.Sprintf("%v", entry)

		if err := yaml.Unmarshal([]byte(stringVal), &targetMap); err != nil {
			for k, v := range hv.parseFromUntypedMap(returnKey, entry.(map[string]interface{})) {
				valMap[k] = v
			}
		} else {
			valMap[returnKey] = entry.(string)
		}

		returnKey = parentKey
	}

	return valMap
}
