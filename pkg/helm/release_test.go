package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReleaseConfigMaps(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()
	apiObjList := getTestReleaseSpecs()

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "repo", Namespace: ""}, &helmv1alpha1.Repo{}).Return(nil)
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "chart", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
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

		testObj := NewHelmRelease(apiObj, settings, &clientMock, &httpMock)
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range apiObj.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		testObj.Version = apiObj.Spec.Version
		configList := testObj.GetParsedConfigMaps()

		// assert.Equal(expected, charts, "Structs should be equal.")
		assert.NotNil(configList)
	}
}

func TestReleaseUpdate(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()
	apiObjList := getTestReleaseSpecs()

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.MatchingLabels{"repo": "repo"}, client.InNamespace("")}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "test", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "release", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := getTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-tmpl-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(2).(*v1.ConfigMap)
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-crds-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(2).(*v1.ConfigMap)
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-default-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := getTestDefaultValueConfigMap()
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

		testObj := NewHelmRelease(apiObj, settings, &clientMock, &httpMock)
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range apiObj.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		testObj.Version = apiObj.Spec.Version
		testObj.ValuesTemplate.ValuesMap = map[string]string{
			"bar": "foo",
		}
		testObj.Config = getFakeActionConfig(t)

		if err := testObj.Config.Releases.Create(getTestDeployedReleaseObj()); err != nil {
			log.Print(err)
		}

		configList := testObj.Update()

		// assert.Equal(expected, charts, "Structs should be equal.")
		assert.Nil(configList)
	}
}

func getTestReleaseSpecs() []*helmv1alpha1.Release {
	return []*helmv1alpha1.Release{
		{
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
		{
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
	}
}

func getTestChartSpec() helmv1alpha1.Chart {
	return helmv1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "chart",
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
	return &chart.Chart{
		Templates: []*chart.File{},
		Values:    map[string]interface{}{},
		Metadata: &chart.Metadata{
			Name:       "meta",
			Version:    "0.0.1",
			APIVersion: "0.0.1",
		},
	}
}

var verbose = flag.Bool("test.log", false, "enable test logging")

func getFakeActionConfig(t *testing.T) *action.Configuration {
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

func getTestDeployedReleaseObj() *release.Release {
	return &release.Release{
		Name:  "release",
		Chart: getTestHelmChart(),
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
	}
}

func getTestDefaultValueConfigMap() v1.ConfigMap {

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
