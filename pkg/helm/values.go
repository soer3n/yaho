package helm

import (
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

func NewValueTemplate(valuesList []*helmv1alpha1.Values) *HelmValueTemplate {
	return &HelmValueTemplate{
		refList: valuesList,
	}
}

func (hv *HelmValueTemplate) ManageValues() error {

	return nil
}
