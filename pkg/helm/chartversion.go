package helm

import (
	"strings"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/chart"
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

func (chartVersion *HelmChartVersion) createConfigMaps(namespace string) []v1.ConfigMap {
	returnList := []v1.ConfigMap{}

	returnList = append(returnList, chartVersion.createConfigMap("tmpl", namespace, chartVersion.Templates))
	returnList = append(returnList, chartVersion.createConfigMap("crds", namespace, chartVersion.CRDs))

	return returnList
}

func (chartVersion *HelmChartVersion) createConfigMap(name string, namespace string, list []*chart.File) v1.ConfigMap {

	immutable := new(bool)
	*immutable = false
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-" + name + "-" + chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
		Namespace: namespace,
	}
	configmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
	}

	binaryData := make(map[string][]byte)

	for _, entry := range list {
		path := strings.SplitAfter(entry.Name, "/")
		binaryData[path[1]] = entry.Data
	}

	configmap.BinaryData = binaryData

	return configmap
}
