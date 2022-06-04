package values

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
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
		Values:    map[string]interface{}{},
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
	var merged map[string]interface{}

	base = NewOptions(
		map[string]string{
			"parent": "base",
		}).
		Filter(hv.ValuesRef)

	if len(base) == 0 {
		return merged, errors.New("no references for parent resource")
	}

	merged = make(map[string]interface{})
	// var wg sync.WaitGroup
	c := make(chan map[string]interface{}, 1)
	d := make(chan map[string]interface{}, 1)
	counter := 0

	for _, ref := range base {
		hv.logger.Info("manage values ref", "struct", ref)

		refValues := ref.Ref.DeepCopy()

		go hv.parse(ref, refValues, c)
	}

	for {
		select {
		case i := <-c:
			merged = utils.MergeMaps(i, merged)
			counter++
			if counter == len(base) {
				return merged, nil
			}

		case <-time.After(1000 * time.Second):
			close(d)
			return map[string]interface{}{}, errors.New("timeout on value parsing")
		}
	}
}

func (hv *ValueTemplate) parse(ref *ValuesRef, refValues *helmv1alpha1.Values, c chan map[string]interface{}) {

	if err := hv.manageStruct(ref); err != nil {
		hv.logger.Info("parsing values reference failed", "error", err.Error())
		return
	}

	values, _ := hv.transformToMap(refValues, true)
	c <- values
}

func (hv *ValueTemplate) manageStruct(valueMap *ValuesRef, parents ...string) error {

	var merged map[string]interface{}
	c := make(chan map[string]interface{}, 1)
	counter := 0

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
			return nil
		}

		for _, v := range temp {
			merged = make(map[string]interface{})
			parents = append(parents, valueMap.Key)
			if v.Ref.Spec.Refs != nil {
				if err := hv.manageStruct(v, parents...); err != nil {
					return err
				}
			}

			refKey := hv.getRefKeyByValue(parents, v.Ref.Name, valueMap.Ref.Spec.Refs)

			go func(v *ValuesRef) {
				merged, _ = hv.transformToMap(v.Ref, true, refKey...)
				c <- merged
			}(v)

		}

		for {
			select {
			case t := <-c:
				// merge child map directly to struct field is better !!!
				hv.Values = utils.MergeMaps(hv.Values, t)
				counter++
				if counter == len(temp) {
					return nil
				}
				return nil
			case <-time.After(1000 * time.Second):
				hv.logger.Info("timeout on managing struct reference", "struct", valueMap)
				return errors.New("time out on value parsing")
			}
		}
	}

	return nil
}

func (hv ValueTemplate) getRefKeyByValue(parents []string, value string, refMap map[string]string) []string {
	for k, v := range refMap {
		if value == v {
			parents = append(parents, k)
		}
	}

	return parents
}

func (hv ValueTemplate) transformToMap(values *helmv1alpha1.Values, unstructed bool, parents ...string) (map[string]interface{}, error) {
	valMap := make(map[string]interface{})
	var parentKey string

	hv.logger.Info("parent key", "values", values.GetName(), "parent", parentKey)

	rawVals := values.Spec.ValuesMap
	var convertedMap map[string]interface{}

	if rawVals != nil && rawVals.Raw != nil {
		if err := json.Unmarshal(rawVals.Raw, &convertedMap); err != nil {
			hv.logger.Error(err, "error on parsing map", "map", rawVals)
			return valMap, err
		}

		hv.logger.Info("converting map succeeded", "parent", parentKey, "map name", values.GetName(), "map length", len(convertedMap))
		hv.logger.Info("converting map succeeded", "parent", parentKey, "map name", values.GetName(), "map", convertedMap)

	}

	valMap = utils.MergeUntypedMaps(hv.Values, convertedMap, parents...)
	return valMap, nil
}
