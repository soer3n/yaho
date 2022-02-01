package helm

import (
	"encoding/json"
	"sync"

	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
)

// NewValueTemplate represents initialization of internal struct for managing helm values
func NewValueTemplate(valuesList []*ValuesRef) *ValueTemplate {
	return &ValueTemplate{
		valuesRef: valuesList,
	}
}

// ManageValues represents parsing of a map with interfaces into HelmValueTemplate struct
func (hv *ValueTemplate) ManageValues() (map[string]interface{}, error) {
	var base []*ValuesRef
	var values, merged map[string]interface{}
	var err error

	base = NewOptions(
		map[string]string{
			"parent": "base",
		}).
		Filter(hv.valuesRef)

	merged = make(map[string]interface{})
	var wg sync.WaitGroup
	c := make(chan map[string]interface{}, 1)

	for _, ref := range base {
		if values, err = hv.manageStruct(ref); err != nil {
			return merged, err
		}

		refValues := ref.Ref
		wg.Add(1)

		go func(refValues *helmv1alpha1.Values, values map[string]interface{}, c chan<- map[string]interface{}) {
			defer wg.Done()
			c <- hv.transformToMap(refValues, values, true)
		}(refValues, values, c)
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	d := make(chan map[string]interface{}, 1)

	for i := range c {
		go func(d chan map[string]interface{}, i, merged map[string]interface{}) {
			d <- mergeMaps(i, merged)
		}(d, i, merged)
		merged = <-d
	}

	return merged, nil
}

func (hv *ValueTemplate) manageStruct(valueMap *ValuesRef) (map[string]interface{}, error) {
	valMap := make(map[string]interface{})
	var merged map[string]interface{}
	c := make(chan map[string]interface{}, 1)

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

			refKey := hv.getRefKeyByValue(v.Ref.Name, valueMap.Ref.Spec.Refs)

			go func(v *ValuesRef, merged, valMap map[string]interface{}, refKey string, c chan<- map[string]interface{}) {
				merged = hv.transformToMap(v.Ref, merged, true, refKey)
				c <- mergeMaps(merged, valMap)
			}(v, merged, valMap, refKey, c)
			valMap = <-c
		}
	}

	return valMap, nil
}

func (hv ValueTemplate) getRefKeyByValue(value string, refMap map[string]string) string {
	for k, v := range refMap {
		if value == v {
			return k
		}
	}

	return ""
}

func (hv ValueTemplate) transformToMap(values *helmv1alpha1.Values, childMap map[string]interface{}, unstructed bool, parents ...string) map[string]interface{} {
	valMap := make(map[string]interface{})
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

		mapKey := ""

		if len(parents) > 0 {
			mapKey = parents[0]
		}

		if unstructed {
			valMap = mergeUntypedMaps(convertedMap, valMap, mapKey)
		}
	}

	return mergeUntypedMaps(childMap, valMap, parentKey)
}
