package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/mocks"
	inttypes "github.com/soer3n/apps-operator/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReleaseConfigMaps(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()
	apiObjList := getTestReleaseSpecs()

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "repo", Namespace: ""}, &helmv1alpha1.Repo{}).Return(nil)
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "notfound", Namespace: ""}, &helmv1alpha1.Repo{}).Return(errors.New("repo not found"))
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "chart", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "notfound", Namespace: ""}, &helmv1alpha1.Chart{}).Return(errors.New("chart not found"))

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(httpResponse, nil)

	httpMock.On("Get",
		"").Return(&http.Response{}, errors.New("no valid url"))

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		current := apiObj.Input.(*helmv1alpha1.Release)
		testObj := NewHelmRelease(current, settings, &clientMock, &httpMock, kube.Client{})
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range current.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		testObj.Version = current.Spec.Version
		configList, _ := testObj.GetParsedConfigMaps("")
		expect, _ := apiObj.ReturnValue.([]v1.ConfigMap)

		assert.Equal(expect, configList)
	}
}

func TestReleaseUpdate(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()
	apiObjList := getTestReleaseSpecs()

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.MatchingLabels{"repoGroup": "group"}, client.InNamespace("")}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(1).(*helmv1alpha1.ChartList)
		spec := helmv1alpha1.ChartList{
			Items: []helmv1alpha1.Chart{
				{
					Spec: helmv1alpha1.ChartSpec{
						Name: "dep",
					},
				},
				getTestChartSpec(),
			},
		}
		c.Items = spec.Items

	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "chart", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "dep", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartDepSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "release", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "notfound", Namespace: ""}, &helmv1alpha1.Chart{}).Return(errors.New("chart not found"))
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-default-notfound-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestReleaseDefaultValueConfigMap()
		c.Data = spec.Data
		c.ObjectMeta = spec.ObjectMeta
	})

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-tmpl-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestReleaseTemplateConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-crds-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestReleaseCRDConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-default-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestReleaseDefaultValueConfigMap()
		c.Data = spec.Data
		c.ObjectMeta = spec.ObjectMeta
	})

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-tmpl-dep-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestReleaseTemplateConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-crds-dep-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestReleaseCRDConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-default-dep-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestReleaseDefaultValueConfigMap()
		c.Data = spec.Data
		c.ObjectMeta = spec.ObjectMeta
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(httpResponse, nil)

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		current := apiObj.Input.(*helmv1alpha1.Release)
		testObj := NewHelmRelease(current, settings, &clientMock, &httpMock, kube.Client{})
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range current.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		testObj.Version = current.Spec.Version
		testObj.ValuesTemplate.ValuesMap = map[string]string{
			"bar": "foo",
		}
		testObj.Config = getTestReleaseFakeActionConfig(t)

		if err := testObj.Config.Releases.Create(getTestReleaseDeployedReleaseObj()); err != nil {
			log.Print(err)
		}

		err := testObj.Update(helmv1alpha1.Namespace{
			Name:    "",
			Install: false,
		}, map[string]helmv1alpha1.DependencyConfig{
			"dep": {Enabled: true},
		})
		assert.Equal(err, apiObj.ReturnError)
	}
}

func TestReleaseInitValuesTemplate(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()
	apiObjList := getTestReleaseValueRefListSpec()
	testRelease := &helmv1alpha1.Release{
		Spec: helmv1alpha1.ReleaseSpec{
			Name:    "test",
			Chart:   "chart",
			Repo:    "repo",
			Version: "0.0.1",
			ValuesTemplate: &helmv1alpha1.ValueTemplate{
				ValueRefs: []string{"notpresent"},
			},
		},
	}

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		current := apiObj.Input.([]*ValuesRef)
		testObj := NewHelmRelease(testRelease, settings, &clientMock, &httpMock, kube.Client{})
		testObj.InitValuesTemplate(current, "namespace", "v0.0.1")
		expect := apiObj.ReturnValue.(map[string]interface{})
		assert.Equal(testObj.Values, expect)
	}
}

func getTestReleaseValueRefListSpec() []inttypes.TestCase {
	values := map[string]interface{}{
		"values": "foo",
		"key":    map[string]string{"bar": "fuz"},
	}
	castedValues, _ := json.Marshal(values)
	var template map[string]interface{}

	return []inttypes.TestCase{
		{
			ReturnValue: template,
			ReturnError: nil,
			Input: []*ValuesRef{
				{
					Ref: &helmv1alpha1.Values{
						ObjectMeta: metav1.ObjectMeta{
							Name: "values",
						},
						Spec: helmv1alpha1.ValuesSpec{
							Refs: map[string]string{},
							ValuesMap: &runtime.RawExtension{
								Raw: castedValues,
							},
						},
					},
				},
			},
		},
	}
}

func getTestReleaseSpecs() []inttypes.TestCase {

	return []inttypes.TestCase{
		{
			ReturnValue: getTestReleaseChartConfigMapsValid(),
			ReturnError: nil,
			Input: &helmv1alpha1.Release{
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "test",
					Chart:   "chart",
					Repo:    "repo",
					Version: "0.0.1",
					ValuesTemplate: &helmv1alpha1.ValueTemplate{
						ValueRefs: []string{"notpresent"},
					},
				},
			},
		},
		{
			ReturnValue: []v1.ConfigMap{},
			ReturnError: nil,
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "selector",
					},
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "release",
					Chart:   "chart",
					Repo:    "repo",
					Version: "0.0.1",
					ValuesTemplate: &helmv1alpha1.ValueTemplate{
						ValueRefs: []string{"notpresent"},
					},
				},
			},
		},
		{
			ReturnValue: []v1.ConfigMap{},
			ReturnError: errors.New("chart not found"),
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "selector",
					},
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "test",
					Chart:   "notfound",
					Repo:    "repo",
					Version: "0.0.1",
					ValuesTemplate: &helmv1alpha1.ValueTemplate{
						ValueRefs: []string{"notpresent"},
						DependenciesConfig: map[string]helmv1alpha1.DependencyConfig{
							"subMeta": {
								Enabled: true,
							},
						},
					},
				},
			},
		},
		{
			ReturnValue: []v1.ConfigMap{},
			ReturnError: errors.New("chart not found"),
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "selector",
					},
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "test",
					Chart:   "notfound",
					Repo:    "notfound",
					Version: "0.0.1",
					ValuesTemplate: &helmv1alpha1.ValueTemplate{
						ValueRefs: []string{"notpresent"},
					},
				},
			},
		},
	}
}

func getTestChartSpec() helmv1alpha1.Chart {
	return helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "chart",
			Labels: map[string]string{
				"repoGroup": "group",
			},
		},
		Spec: helmv1alpha1.ChartSpec{
			Name:       "chart",
			APIVersion: "0.0.1",
			Versions: []helmv1alpha1.ChartVersion{
				{
					Name: "0.0.1",
					URL:  "https://foo.bar/charts/foo-0.0.1.tgz",
					Dependencies: []*helmv1alpha1.ChartDep{
						{
							Name:    "dep",
							Version: "0.0.1",
							Repo:    "repo",
						},
					},
				},
			},
		},
	}
}

func getTestChartDepSpec() helmv1alpha1.Chart {
	return helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dep",
			Labels: map[string]string{
				"repoGroup": "group",
			},
		},
		Spec: helmv1alpha1.ChartSpec{
			Name:       "chart",
			APIVersion: "0.0.1",
			Versions: []helmv1alpha1.ChartVersion{
				{
					Name: "0.0.1",
					URL:  "https://foo.bar/charts/foo-0.0.1.tgz",
				},
			},
		},
	}
}

func getTestHelmChart() *chart.Chart {
	c := &chart.Chart{
		Templates: []*chart.File{},
		Values:    map[string]interface{}{},
		Metadata: &chart.Metadata{
			Name:       "meta",
			Version:    "0.0.1",
			APIVersion: "0.0.1",
			Dependencies: []*chart.Dependency{
				{
					Name:    "subMeta",
					Version: "0.0.1",
				},
			},
		},
	}

	c.AddDependency(&chart.Chart{
		Templates: []*chart.File{},
		Values:    map[string]interface{}{},
		Metadata: &chart.Metadata{
			Name:       "subMeta",
			Version:    "0.0.1",
			APIVersion: "0.0.1",
		},
	})
	return c
}

var verbose = flag.Bool("test.log", false, "enable test logging")

func getTestReleaseFakeActionConfig(t *testing.T) *action.Configuration {
	return &action.Configuration{
		Releases:     storage.Init(driver.NewMemory()),
		KubeClient:   &kubefake.FailingKubeClient{PrintingKubeClient: kubefake.PrintingKubeClient{Out: ioutil.Discard}},
		Capabilities: chartutil.DefaultCapabilities,
		Log: func(format string, v ...interface{}) {
			t.Helper()
			if *verbose {
				t.Logf(format, v...)
			}
		},
	}
}

func getTestReleaseDeployedReleaseObj() *release.Release {
	return &release.Release{
		Name:  "release",
		Chart: getTestHelmChart(),
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
	}
}

func getTestReleaseDefaultValueConfigMap() v1.ConfigMap {

	immutable := new(bool)
	*immutable = true
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-default-chart-0.0.1",
		Namespace: "",
	}
	configmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
		Data:       make(map[string]string),
	}

	values := map[string]interface{}{
		"values": "foo",
		"key":    map[string]string{"bar": "fuz"},
	}
	castedValues, _ := json.Marshal(values)
	configmap.Data["values"] = string(castedValues)

	return configmap
}

func getTestReleaseTemplateConfigMap() v1.ConfigMap {

	immutable := new(bool)
	*immutable = true
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-tmpl-chart-0.0.1",
		Namespace: "",
	}
	configmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
		BinaryData: make(map[string][]byte),
	}

	values := map[string]interface{}{
		"values": "foo",
		"key":    map[string]string{"bar": "fuz"},
	}
	castedValues, _ := json.Marshal(values)
	configmap.BinaryData["values"] = castedValues

	return configmap
}

func getTestReleaseCRDConfigMap() v1.ConfigMap {

	immutable := new(bool)
	*immutable = true
	objectMeta := metav1.ObjectMeta{
		Name:      "helm-crd-chart-0.0.1",
		Namespace: "",
	}
	configmap := v1.ConfigMap{
		Immutable:  immutable,
		ObjectMeta: objectMeta,
		BinaryData: make(map[string][]byte),
	}

	values := map[string]interface{}{
		"values": "foo",
		"key":    map[string]string{"bar": "fuz"},
	}
	castedValues, _ := json.Marshal(values)
	configmap.BinaryData["values"] = castedValues

	return configmap
}

func getTestReleaseChartConfigMapsValid() []v1.ConfigMap {

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	chart, _ := loader.LoadArchive(raw)

	immutable := new(bool)
	*immutable = true

	l := []v1.ConfigMap{}
	t := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "helm-tmpl-chart-0.0.1",
		},
		Immutable:  immutable,
		BinaryData: map[string][]byte{},
	}

	for _, tpl := range chart.Templates {
		path := strings.SplitAfter(tpl.Name, "/")
		t.BinaryData[path[1]] = tpl.Data
	}

	l = append(l, t)
	t = v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "helm-crds-chart-0.0.1",
		},
		Immutable:  immutable,
		BinaryData: map[string][]byte{},
	}

	for _, tpl := range chart.CRDs() {
		path := strings.SplitAfter(tpl.Name, "/")
		t.BinaryData[path[1]] = tpl.Data
	}

	l = append(l, t)
	t = v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "helm-default-chart-0.0.1",
		},
		Immutable: immutable,
		Data:      map[string]string{},
	}

	castedValues, _ := json.Marshal(chart.Values)
	t.Data["values"] = string(castedValues)

	l = append(l, t)
	return l
}
