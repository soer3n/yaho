package helm

func NewValueTemplate(valuesList []*ValuesRef) *HelmValueTemplate {
	return &HelmValueTemplate{
		valuesRef: valuesList,
	}
}

func (hv *HelmValueTemplate) ManageValues() error {
	var values map[string]interface{}
	var base, temp []*ValuesRef
	var optStruct *ListOptions
	var options map[string]string

	options = map[string]string{
		"parent": "base",
	}

	optStruct = NewOptions(options)
	base = optStruct.Filter(hv.valuesRef)

	for _, ref := range base {
		options = map[string]string{
			"parent": ref.Ref.ObjectMeta.Name,
		}

		optStruct = NewOptions(options)
		temp = optStruct.Filter(hv.valuesRef)
	}

	return nil
}

func (hv *HelmValueTemplate) mergeMaps(valueMap map[string]interface{}) error {
	for k, v := range valueMap {
		hv.Values[k] = v
	}
	return nil

}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	for k, v := range a {
		b[k] = v
	}
	return b
}
