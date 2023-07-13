package chartversion

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const configMapLabelKey = "yaho.soer3n.dev/chart"
const configMapRepoLabelKey = "yaho.soer3n.dev/repo"
const configMapLabelType = "yaho.soer3n.dev/type"

// CreateConfigMaps represents func for parsing configmaps and sending them to receive method
func createConfigMaps(cm chan v1.ConfigMap, hc *chart.Chart, v *repo.ChartVersion, repository, namespace string, logger logr.Logger) error {

	wg := &sync.WaitGroup{}

	wg.Add(3)

	go func() {
		createTemplateConfigMap(cm, "tmpl", namespace, repository, hc, v, logger)
		wg.Done()
	}()

	go func() {
		createTemplateConfigMap(cm, "crds", namespace, repository, hc, v, logger)
		wg.Done()
	}()

	go func() {
		createDefaultValueConfigMap(cm, namespace, repository, v, hc.Values, logger)
		wg.Done()
	}()

	wg.Wait()
	return nil
}

func GetDefaultValuesFromConfigMap(name, repository, version, namespace string, c client.WithWatch, logger logr.Logger) map[string]interface{} {
	var err error
	values := make(map[string]interface{})
	configmap := &v1.ConfigMap{}

	configMapName := "helm-default-" + repository + "-" + name + "-" + version

	if err = c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: configMapName}, configmap); err != nil {
		logger.Info("error on getting default values", "msg", err.Error())
		return values
	}

	jsonMap := make(map[string]interface{})

	if err = json.Unmarshal([]byte(configmap.Data["values"]), &jsonMap); err != nil {
		panic(err)
	}

	return jsonMap
}

func ParseConfigMaps(cm chan v1.ConfigMap, hc *chart.Chart, v *repo.ChartVersion, repository, namespace string, logger logr.Logger) error {

	if hc == nil || hc.Metadata == nil {
		return k8serrors.NewBadRequest("chart not loaded")
	}

	go func() {
		if err := createConfigMaps(cm, hc, v, repository, namespace, logger); err != nil {
			logger.Error(err, "error on creating or updating related resources")
		}
		close(cm)
	}()

	return nil
}

// TODO: move this to chart model!
func DeployConfigMap(configmap v1.ConfigMap, hc *chart.Chart, v *repo.ChartVersion, repository, namespace string, localClient, remoteClient client.WithWatch, scheme *runtime.Scheme, logger logr.Logger) error {
	//mu := &sync.Mutex{}
	//defer mu.Unlock()
	//mu.Lock()

	if reflect.DeepEqual(localClient, remoteClient) {
		chartList := &yahov1alpha2.ChartList{}
		ls := labels.Set{}

		// filter repositories by group selector if set
		ls = labels.Merge(ls, labels.Set{configMapLabelKey: hc.Name()})

		if err := localClient.List(context.Background(), chartList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(ls),
		}); err != nil {
			return err
		}

		if len(chartList.Items) != 1 {
			return errors.New("multiple charts found")
		}
		chartObj := chartList.Items[0]
		if err := controllerutil.SetControllerReference(&chartObj, &configmap, scheme); err != nil {
			return err
		}
	}

	current := &v1.ConfigMap{}
	err := remoteClient.Get(context.Background(), client.ObjectKey{
		Namespace: configmap.ObjectMeta.Namespace,
		Name:      configmap.ObjectMeta.Name,
	}, current)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			if err = remoteClient.Create(context.TODO(), &configmap); err != nil {
				return err
			}
		}
		return err
	}

	if err := remoteClient.Update(context.Background(), &configmap, &client.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func createTemplateConfigMap(cm chan v1.ConfigMap, name, namespace, repository string, hc *chart.Chart, v *repo.ChartVersion, logger logr.Logger) {
	immutable := new(bool)
	*immutable = true

	var list []*chart.File

	switch name {
	case "crds":
		list = hc.CRDs()
	default:
		list = hc.Templates
	}

	objectMeta := metav1.ObjectMeta{
		Name:      "helm-" + name + "-" + repository + "-" + v.Metadata.Name + "-" + v.Metadata.Version,
		Namespace: namespace,
		Labels: map[string]string{
			configMapLabelKey:     v.Metadata.Name + "-" + v.Metadata.Version,
			configMapRepoLabelKey: repository,
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

		// skip directory for tests if present
		if key == "tests" {
			continue
		}

		fileName := strings.Replace(path[2], "/", "", 1)
		if _, ok := configMapMap[key]; !ok {
			configMapMap[key] = v1.ConfigMap{
				Immutable: immutable,
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-" + name + "-" + repository + "-" + v.Metadata.Name + "-" + key + "-" + v.Metadata.Version,
					Namespace: namespace,
					Labels: map[string]string{
						configMapLabelKey:     v.Metadata.Name + "-" + v.Metadata.Version,
						configMapRepoLabelKey: repository,
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

	baseConfigmap.BinaryData = binaryData
	cm <- baseConfigmap

	for _, configmap := range configMapMap {
		cm <- configmap
	}

}

func createDefaultValueConfigMap(cm chan v1.ConfigMap, namespace, repository string, v *repo.ChartVersion, values map[string]interface{}, logger logr.Logger) {
	immutable := new(bool)
	*immutable = true
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-default-" + repository + "-" + v.Metadata.Name + "-" + v.Metadata.Version,
		Namespace: namespace,
		Labels: map[string]string{
			configMapLabelKey:     v.Metadata.Name + "-" + v.Metadata.Version,
			configMapRepoLabelKey: repository,
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

	logger.Info("parsed default values configmap", "chart", v.Name, "repository", repository, "configmap", configmap)

	cm <- configmap
}
