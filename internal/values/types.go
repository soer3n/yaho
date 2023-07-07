package values

import (
	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValueTemplate represents struct for possible value inputs
type ValueTemplate struct {
	ValuesRef  []*ValuesRef
	Values     map[string]interface{}
	ValuesMap  map[string]string
	ValueFiles []string
	logger     logr.Logger
	k8sClient  client.Client
}

// ValuesRef represents struct for filtering values kubernetes resources by json tag
type ValuesRef struct {
	Ref    *yahov1alpha2.Values `json:"Ref" filter:"ref"`
	Parent string               `json:"Parent" filter:"parent"`
	Key    string               `json:"Key" filter:"key"`
}

// ListOptions represents struct for filters for searching in values kubernetes resources
type ListOptions struct {
	filter map[string]string
}
