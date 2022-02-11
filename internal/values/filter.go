package values

import (
	"reflect"
	"strings"
)

// NewOptions for init struct by map
func NewOptions(filterMap map[string]string) *ListOptions {
	return &ListOptions{
		filter: filterMap,
	}
}

// Filter and every other func adapted from: https://play.golang.org/p/o5JmVAL9RXL
func (options *ListOptions) Filter(list []*ValuesRef) []*ValuesRef {
	cs := []*ValuesRef{}
	for _, c := range list {
		if options.filterValues(c) {
			cs = append(cs, c)
		}
	}

	return cs
}

func (options *ListOptions) filterValues(c *ValuesRef) bool {
	filterLists := options.filter

	spec := options.getFilterSpec(c)
	for k, v := range filterLists {
		if spec[k] != v {
			return false
		}
	}

	return true
}

func (options *ListOptions) convertV(values interface{}) reflect.Value {
	conV, ok := values.(reflect.Value)
	if !ok {
		conV = reflect.Indirect(reflect.ValueOf(values))
	}
	return conV
}

func (options *ListOptions) getFilterName(valueT reflect.Type, confV reflect.Value, i int, prefix string) (reflect.Value, string, string) {
	field := confV.Field(i)
	structName := valueT.Field(i).Name
	tag := valueT.Field(i).Tag

	filterName := ""
	if prefix != "" {
		filterName = prefix + "." + structName
	}
	if f := tag.Get("filter"); f != "" {
		filterName = f
	}

	return field, structName, filterName
}

func (options *ListOptions) getFilterSpec(f *ValuesRef) map[string]string {
	list := options.listValueSpecs(f, "")
	m := map[string]string{}

	for _, l := range list {
		kv := strings.Split(l, "=")
		m[kv[0]] = kv[1]
	}

	return m
}

func (options *ListOptions) listValueSpecs(values interface{}, prefix string) []string {
	list := []string{}

	conV := options.convertV(values)
	conT := conV.Type()
	for i := 0; i < conV.NumField(); i++ {
		field, structName, filterName := options.getFilterName(conT, conV, i, prefix)
		switch field.Kind() {
		case reflect.Struct:
			if prefix != "" {
				structName = prefix + "." + structName
			}
			list = append(list, options.listValueSpecs(field, structName)...)
		default:
			if v := conV.Field(i).String(); v != "" {
				list = append(list, filterName+"="+v)
			}
		}
	}

	return list
}
