package chartversion

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"helm.sh/helm/v3/pkg/chart"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CreateConfigMaps represents func for parsing configmaps and sending them to receive method
func (chartVersion *ChartVersion) CreateConfigMaps(cm chan v1.ConfigMap, deps []*chart.Chart) error {

	wg := &sync.WaitGroup{}

	wg.Add(3)

	go func() {
		chartVersion.createTemplateConfigMap(cm, "tmpl")
		wg.Done()
	}()

	go func() {
		chartVersion.createTemplateConfigMap(cm, "crds")
		wg.Done()
	}()

	go func() {
		chartVersion.createDefaultValueConfigMap(cm, chartVersion.DefaultValues)
		wg.Done()
	}()

	wg.Wait()
	return nil
}

func (chartVersion *ChartVersion) getDefaultValuesFromConfigMap(name, version string) map[string]interface{} {
	var err error
	values := make(map[string]interface{})
	configmap := &v1.ConfigMap{}

	configMapName := "helm-default-" + name + "-" + version

	if err = chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{Namespace: chartVersion.owner.Namespace, Name: configMapName}, configmap); err != nil {
		return values
	}

	jsonMap := make(map[string]interface{})

	if err = json.Unmarshal([]byte(configmap.Data["values"]), &jsonMap); err != nil {
		panic(err)
	}

	return jsonMap
}

func (chartVersion *ChartVersion) parseConfigMaps(cm chan v1.ConfigMap) error {

	if chartVersion.Obj == nil || chartVersion.Obj.Metadata == nil {
		return k8serrors.NewBadRequest("chart not loaded")
	}

	chartVersion.Templates = chartVersion.Obj.Templates
	chartVersion.CRDs = chartVersion.Obj.CRDs()
	chartVersion.DefaultValues = chartVersion.Obj.Values
	// actually not needed
	deps := chartVersion.Obj.Dependencies()

	go func() {
		if err := chartVersion.CreateConfigMaps(cm, deps); err != nil {
			chartVersion.logger.Error(err, "error on creating or updating related resources")
		}
		close(cm)
	}()

	return nil
}

func (chartVersion *ChartVersion) deployConfigMap(configmap v1.ConfigMap) error {
	defer chartVersion.mu.Unlock()
	chartVersion.mu.Lock()
	if err := controllerutil.SetControllerReference(chartVersion.owner, &configmap, chartVersion.scheme); err != nil {
		return err
	}

	current := &v1.ConfigMap{}
	err := chartVersion.k8sClient.Get(context.Background(), client.ObjectKey{
		Namespace: configmap.ObjectMeta.Namespace,
		Name:      configmap.ObjectMeta.Name,
	}, current)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			if err = chartVersion.k8sClient.Create(context.TODO(), &configmap); err != nil {
				return err
			}
		}
		return err
	}

	if err := chartVersion.k8sClient.Update(context.Background(), &configmap, &client.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func (chartVersion *ChartVersion) createTemplateConfigMap(cm chan v1.ConfigMap, name string) {
	immutable := new(bool)
	*immutable = true

	var list []*chart.File

	switch name {
	case "crds":
		list = chartVersion.Obj.CRDs()
	default:
		list = chartVersion.Obj.Templates
	}

	objectMeta := metav1.ObjectMeta{
		Name:      "helm-" + name + "-" + chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
		Namespace: chartVersion.owner.Namespace,
		Labels: map[string]string{
			configMapLabelKey:     chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
			configMapRepoLabelKey: chartVersion.owner.Spec.Repository,
			configMapLabelType:    name,
		},
	}
	baseConfigmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
	}

	configMapMap := make(map[string]v1.ConfigMap)
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
					Namespace: chartVersion.owner.Namespace,
					Labels: map[string]string{
						configMapLabelKey:     chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
						configMapRepoLabelKey: chartVersion.owner.Spec.Repository,
						configMapLabelType:    name,
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

	if len(binaryData) > 0 {
		baseConfigmap.BinaryData = binaryData
		cm <- baseConfigmap
	}

	for _, configmap := range configMapMap {
		cm <- configmap
	}

}

func (chartVersion *ChartVersion) createDefaultValueConfigMap(cm chan v1.ConfigMap, values map[string]interface{}) {
	immutable := new(bool)
	*immutable = true
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-default-" + chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
		Namespace: chartVersion.owner.Namespace,
		Labels: map[string]string{
			configMapLabelKey:     chartVersion.Version.Metadata.Name + "-" + chartVersion.Version.Metadata.Version,
			configMapRepoLabelKey: chartVersion.owner.Spec.Repository,
			configMapLabelType:    "default",
		},
	}
	configmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
		Data:       make(map[string]string),
	}

	castedValues, _ := json.Marshal(values)
	configmap.Data["values"] = string(castedValues)

	cm <- configmap
}
