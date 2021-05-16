package helm

import (
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (chartVersion *HelmChartVersion) AddOrUpdateChartMap(chartObjMap map[string]*helmv1alpha1.Chart, instance *helmv1alpha1.Repo) (map[string]*helmv1alpha1.Chart, error) {

	chartMeta := chartVersion.Version.Metadata
	_, ok := chartObjMap[chartMeta.Name]

	if ok {
		chartObjMap[chartMeta.Name].Spec.Versions = append(chartObjMap[chartMeta.Name].Spec.Versions, chartMeta.Version)
		return chartObjMap, nil
	}

	helmChart := &helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chartMeta.Name,
			Namespace: instance.ObjectMeta.Namespace,
			Labels: map[string]string{
				"chart":     chartMeta.Name,
				"repo":      instance.Spec.Name,
				"repoGroup": instance.ObjectMeta.Labels["repoGroup"],
			},
		},
		Spec: helmv1alpha1.ChartSpec{
			Name:        chartMeta.Name,
			Home:        chartMeta.Home,
			Sources:     chartMeta.Sources,
			Versions:    []string{chartMeta.Version},
			Description: chartMeta.Description,
			Keywords:    chartMeta.Keywords,
			Maintainers: chartMeta.Maintainers,
			Icon:        chartMeta.Icon,
			APIVersion:  chartMeta.APIVersion,
			Condition:   chartMeta.Condition,
			Tags:        chartMeta.Tags,
			AppVersion:  chartMeta.AppVersion,
			Deprecated:  chartMeta.Deprecated,
			Annotations: chartMeta.Annotations,
			KubeVersion: chartMeta.KubeVersion,
			Type:        chartMeta.Type,
		},
	}

	chartObjMap[chartMeta.Name] = helmChart
	return chartObjMap, nil
}

func (chartVersion *HelmChartVersion) createTemplateConfigMap() v1.ConfigMap {
	return v1.ConfigMap{}
}

func (chartVersion *HelmChartVersion) createCRDConfigMap() v1.ConfigMap {
	return v1.ConfigMap{}
}
