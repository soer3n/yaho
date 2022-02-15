package values

import (
	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
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
	Ref    *helmv1alpha1.Values `json:"Ref" filter:"ref"`
	Parent string               `json:"Parent" filter:"parent"`
}

// ListOptions represents struct for filters for searching in values kubernetes resources
type ListOptions struct {
	filter map[string]string
}
