package helm

import (
	"encoding/json"
	"strings"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const configMapLabelKey = "helm.soer3n.info/chart"

// const configMapRepoLabelKey = "helm.soer3n.info/repo"
const configMapLabelSubName = "helm.soer3n.info/subname"

// AddOrUpdateChartMap represents update of version specific data of a map of chart structs if needed
func (chartVersion ChartVersion) AddOrUpdateChartMap(chartObjMap map[string]*helmv1alpha1.Chart, instance *helmv1alpha1.Repo) (map[string]*helmv1alpha1.Chart, error) {
	chartMeta := chartVersion.Version.Metadata
	_, ok := chartObjMap[chartMeta.Name]
	chartURL, err := repo.ResolveReferenceURL(instance.Spec.URL, chartVersion.Version.URLs[0])
	if err != nil {
		return chartObjMap, err
	}

	version := helmv1alpha1.ChartVersion{
		Name:         chartMeta.Version,
		Templates:    "helm-tmpl-" + chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
		CRDs:         "helm-crds-" + chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
		Dependencies: chartVersion.createDependenciesList(chartMeta),
		URL:          chartURL,
	}

	if ok {
		chartObjMap[chartMeta.Name].Spec.Versions = append(chartObjMap[chartMeta.Name].Spec.Versions, version)
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
			Name:    chartMeta.Name,
			Home:    chartMeta.Home,
			Sources: chartVersion.Version.Sources,
			Versions: []helmv1alpha1.ChartVersion{
				version,
			},
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

func (chartVersion ChartVersion) createDependenciesList(chartMeta *chart.Metadata) []*helmv1alpha1.ChartDep {
	deps := make([]*helmv1alpha1.ChartDep, 0)

	for _, dep := range chartMeta.Dependencies {
		deps = append(deps, &helmv1alpha1.ChartDep{
			Name:      dep.Name,
			Version:   dep.Version,
			Repo:      dep.Repository,
			Condition: dep.Condition,
		})
	}

	return deps
}

func (chartVersion ChartVersion) createConfigMaps(namespace string, deps []*chart.Chart) []v1.ConfigMap {
	returnList := []v1.ConfigMap{}

	returnList = append(returnList, chartVersion.createTemplateConfigMap("tmpl", namespace, chartVersion.Templates)...)
	returnList = append(returnList, chartVersion.createTemplateConfigMap("crds", namespace, chartVersion.CRDs)...)
	returnList = append(returnList, chartVersion.createDefaultValueConfigMap(namespace, chartVersion.DefaultValues))

	cms := chartVersion.createDependenciesConfigMaps(namespace, deps)
	returnList = append(returnList, cms...)

	return returnList
}

func (chartVersion ChartVersion) createDependenciesConfigMaps(namespace string, deps []*chart.Chart) []v1.ConfigMap {
	cmList := []v1.ConfigMap{}
	immutable := new(bool)
	*immutable = true

	for _, dep := range deps {
		binaryData := make(map[string][]byte)

		for _, entry := range dep.Templates {
			path := strings.SplitAfter(entry.Name, "/")
			binaryData[path[len(path)-1]] = entry.Data
		}

		cmList = append(cmList, v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helm-tmpl-" + dep.Name() + "-" + dep.Metadata.Version,
				Namespace: namespace,
				Labels: map[string]string{
					configMapLabelKey: dep.Metadata.Name + "-" + dep.Metadata.Version + "-tmpl",
				},
			},
			Immutable:  immutable,
			BinaryData: binaryData,
		})

		binaryData = make(map[string][]byte)

		for _, entry := range dep.CRDs() {
			path := strings.SplitAfter(entry.Name, "/")
			binaryData[path[len(path)-1]] = entry.Data
		}

		cmList = append(cmList, v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helm-crds-" + dep.Name() + "-" + dep.Metadata.Version,
				Namespace: namespace,
			},
			Immutable:  immutable,
			BinaryData: binaryData,
		})

		castedValues, _ := json.Marshal(dep.Values)

		cmList = append(cmList, v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helm-default-" + dep.Name() + "-" + dep.Metadata.Version,
				Namespace: namespace,
			},
			Immutable: immutable,
			Data: map[string]string{
				"values": string(castedValues),
			},
		})

		subConfigMaps := chartVersion.createDependenciesConfigMaps(namespace, dep.Dependencies())
		cmList = append(cmList, subConfigMaps...)
	}

	return cmList
}

func (chartVersion ChartVersion) createTemplateConfigMap(name string, namespace string, list []*chart.File) []v1.ConfigMap {
	immutable := new(bool)
	*immutable = true
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-" + name + "-" + chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
		Namespace: namespace,
		Labels: map[string]string{
			configMapLabelKey: chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version + "-" + name,
		},
	}
	baseConfigmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
	}

	configMapMap := make(map[string]v1.ConfigMap)
	configMapList := []v1.ConfigMap{}

	binaryData := make(map[string][]byte)

	for _, entry := range list {
		path := strings.SplitAfter(entry.Name, "/")

		if len(path) > 3 {
			continue
		}

		if len(path) == 2 {
			binaryData[path[len(path)-1]] = entry.Data
			continue
		}
		key := strings.Replace(path[1], "/", "", 1)
		fileName := strings.Replace(path[2], "/", "", 1)
		if _, ok := configMapMap[key]; !ok {
			configMapMap[key] = v1.ConfigMap{
				Immutable: immutable,
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-" + name + "-" + chartVersion.Version.Metadata.Name + "-" + key + "-" + chartVersion.Version.Metadata.Version,
					Namespace: namespace,
					Labels: map[string]string{
						configMapLabelKey:     chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version + "-" + name,
						configMapLabelSubName: key,
					},
				},
				BinaryData: map[string][]byte{
					fileName: entry.Data,
				},
			}
			continue
		}

		configMapMap[key].BinaryData[fileName] = entry.Data
	}

	baseConfigmap.BinaryData = binaryData
	configMapList = append(configMapList, baseConfigmap)

	for _, c := range configMapMap {
		configMapList = append(configMapList, c)
	}

	return configMapList
}

func (chartVersion ChartVersion) createDefaultValueConfigMap(namespace string, values map[string]interface{}) v1.ConfigMap {
	immutable := new(bool)
	*immutable = true
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-default-" + chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
		Namespace: namespace,
	}
	configmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
		Data:       make(map[string]string),
	}

	castedValues, _ := json.Marshal(values)
	configmap.Data["values"] = string(castedValues)

	return configmap
}
