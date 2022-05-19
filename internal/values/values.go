package values

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New represents initialization of internal struct for managing helm values
func New(instance *helmv1alpha1.Release, logger logr.Logger, k8sClient client.Client) *ValueTemplate {

	valuesList := []*helmv1alpha1.Values{}

	hv := &ValueTemplate{
		// ValuesRef: valuesList,
		logger:    logger,
		k8sClient: k8sClient,
	}

	if instance.Spec.Values != nil {
		valuesList = hv.getValuesByReference(instance.Spec.Values, instance.ObjectMeta.Namespace)
	}

	refList, err := hv.getRefList(valuesList, instance)

	if err != nil {
		hv.logger.Error(err, "error on parsing values list")
	}

	hv.ValuesRef = refList
	return hv
}

// MergeValues returns map of input values and input chart default values
func MergeValues(specValues map[string]interface{}, helmChart *helmchart.Chart) map[string]interface{} {
	// parsing values; goroutines are nessecarry due to tail recursion in called funcs
	// init buffered channel for coalesce values
	c := make(chan map[string]interface{}, 1)

	// run coalesce values in separate goroutine to avoid memory leak in main goroutine
	go func() {
		cv, _ := chartutil.CoalesceValues(helmChart, specValues)
		c <- cv
	}()

	select {
	case t := <-c:
		return t

	case <-time.After(10 * time.Second):
		return map[string]interface{}{}
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
		Filter(hv.ValuesRef)

	if len(base) == 0 {
		return map[string]interface{}{}, errors.New("no references for parent resource")
	}

	merged = make(map[string]interface{})
	//var wg sync.WaitGroup
	c := make(chan map[string]interface{}, 1)
	d := make(chan map[string]interface{}, 1)

	for _, ref := range base {
		hv.logger.Info("manage values ref", "struct", ref)
		if values, err = hv.manageStruct(ref); err != nil {
			return merged, err
		}

		refValues := ref.Ref.DeepCopy()
		//wg.Add(1)

		go func() {
			//defer wg.Done()
			c <- hv.transformToMap(refValues, values, true)
		}()
	}

	// wg.Wait()
	// close(c)

	go func() {
		for i := range c {
			m := utils.MergeMaps(i, merged)
			d <- m
		}
		// close(d)
	}()

	select {
	case t := <-d:
		merged = t
		return merged, nil
	case <-time.After(10 * time.Second):
		close(d)
		return map[string]interface{}{}, errors.New("timeout on value parsing")
	}
}

func (hv *ValueTemplate) manageStruct(valueMap *ValuesRef) (map[string]interface{}, error) {
	valMap := make(map[string]interface{})
	var merged map[string]interface{}
	c := make(chan map[string]interface{}, 1)

	hv.logger.Info("manage struct", "ref", valueMap)

	if valueMap.Ref.Spec.Refs != nil {
		temp := NewOptions(
			map[string]string{
				"parent": valueMap.Ref.ObjectMeta.Name,
			}).
			Filter(hv.ValuesRef)

		hv.logger.Info("internal query result", "filter", "parent:"+valueMap.Ref.ObjectMeta.Name, "result", temp)

		if len(temp) == 0 {
			hv.logger.Info("skip due to empty list", "filter", "parent:"+valueMap.Ref.ObjectMeta.Name)
			return valMap, nil
		}

		for _, v := range temp {
			merged = make(map[string]interface{})
			if v.Ref.Spec.Refs != nil {
				if merged, err := hv.manageStruct(v); err != nil {
					return merged, err
				}
			}

			refKey := hv.getRefKeyByValue(valueMap.Key, valueMap.Ref.Name, valueMap.Ref.Spec.Refs)

			go func(v *ValuesRef) {
				merged = hv.transformToMap(v.Ref, merged, true, refKey)
				c <- utils.MergeMaps(merged, valMap)
			}(v)

		}

		select {
		case t := <-c:
			valMap = t
			return valMap, nil
		case <-time.After(10 * time.Second):
			hv.logger.Info("timeout on managing struct reference", "struct", valueMap)
			return nil, errors.New("time out on value parsing")
		}
	}

	return valMap, nil
}

func (hv ValueTemplate) getRefKeyByValue(parent, value string, refMap map[string]string) string {
	for k, v := range refMap {
		if value == v {
			return parent + "." + k
		}
	}

	return parent
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

	hv.logger.Info("parent key", "values", values.GetName(), "child", childMap, "parent", parentKey)

	rawVals := values.Spec.ValuesMap
	var convertedMap map[string]interface{}

	if rawVals != nil && rawVals.Raw != nil {
		if err := json.Unmarshal(rawVals.Raw, &convertedMap); err != nil {
			hv.logger.Error(err, "error on parsing map", "map", rawVals)
			return valMap
		}

		hv.logger.Info("converting map succeeded", "map name", values.GetName(), "map length", len(convertedMap))
		hv.logger.Info("converting map succeeded", "map name", values.GetName(), "map", convertedMap)

		mapKey := ""

		if len(parents) > 0 {
			mapKey = parents[0]
		}

		if unstructed {
			valMap = utils.MergeUntypedMaps(convertedMap, valMap, mapKey)
		}
	}

	return utils.MergeUntypedMaps(childMap, valMap, parentKey)
}
