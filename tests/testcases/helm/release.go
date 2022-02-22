package helm

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/values"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GetTestReleaseValueRefListSpec returns testcases for testing release cr with values
func GetTestReleaseValueRefListSpec() []inttypes.TestCase {
	testValues := map[string]interface{}{
		"values": "foo",
		"key":    map[string]string{"bar": "fuz"},
	}
	castedValues, _ := json.Marshal(testValues)
	var template map[string]interface{}

	return []inttypes.TestCase{
		{
			ReturnValue: template,
			ReturnError: nil,
			Input: []*values.ValuesRef{
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

// GetTestReleaseSpecs returns testcases for testing release cr
func GetTestReleaseSpecs() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnValue: GetTestReleaseChartConfigMapsValid(),
			ReturnError: nil,
			Input: &helmv1alpha1.Release{
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "test",
					Chart:   "chart",
					Repo:    "repo",
					Version: "0.0.1",
					Values:  []string{"notpresent"},
				},
			},
		},
		{
			ReturnValue: GetTestReleaseChartConfigMapsValid(),
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
					Values:  []string{"notpresent"},
				},
			},
		},
		{
			ReturnValue: []v1.ConfigMap{},
			ReturnError: k8serrors.NewBadRequest("chart not loaded on action update"),
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
					Values:  []string{"notpresent"},
				},
			},
		},
		{
			ReturnValue: []v1.ConfigMap{},
			ReturnError: k8serrors.NewBadRequest("chart not loaded on action update"),
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
					Values:  []string{"notpresent"},
				},
			},
		},
	}
}

// GetTestReleaseSpecsForConfigMaps returns testcases for testing release cr
func GetTestReleaseSpecsForConfigMaps() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnValue: map[string]int{
				"configmap": 6,
				"chart":     1,
			},
			ReturnError: nil,
			Input: &helmv1alpha1.Release{
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    "test",
					Chart:   "chart",
					Repo:    "repo",
					Version: "0.0.1",
					Values:  []string{"notpresent"},
				},
			},
		},
		{
			ReturnValue: map[string]int{
				"configmap": 6,
				"chart":     1,
			},
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
					Values:  []string{"notpresent"},
				},
			},
		},
		{
			ReturnValue: map[string]int{
				"configmap": 0,
				"chart":     0,
			},
			ReturnError: k8serrors.NewBadRequest("chart not loaded on action update affected resources"),
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
					Values:  []string{"notpresent"},
				},
			},
		},
		{
			ReturnValue: map[string]int{
				"configmap": 0,
				"chart":     0,
			},
			ReturnError: k8serrors.NewBadRequest("chart not loaded on action update affected resources"),
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
					Values:  []string{"notpresent"},
				},
			},
		},
	}
}

// GetTestChartSpec returns chart spec for testing release cr with dependencies
func GetTestChartSpec() helmv1alpha1.Chart {
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

// GetTestChartDepSpec returns chart spec for testing release cr
func GetTestChartDepSpec() helmv1alpha1.Chart {
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

// GetTestHelmChart returns chart struct for testing release cr
func GetTestHelmChart() *chart.Chart {
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

// GetTestReleaseFakeActionConfig returns helm action configuration for testing release cr
func GetTestReleaseFakeActionConfig(t *testing.T) *action.Configuration {
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

// GetTestReleaseDeployedReleaseObj returns helm release for testing release cr
func GetTestReleaseDeployedReleaseObj() *release.Release {
	return &release.Release{
		Name:  "release",
		Chart: GetTestHelmChart(),
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
	}
}

// GetTestReleaseDefaultValueConfigMap returns configmap with default values for testing release cr
func GetTestReleaseDefaultValueConfigMap() v1.ConfigMap {
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

// GetTestReleaseTemplateConfigMap returns configmap with templates for testing release cr
func GetTestReleaseTemplateConfigMap() v1.ConfigMap {
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

// GetTestReleaseCRDConfigMap returns configmap with crds for testing release cr
func GetTestReleaseCRDConfigMap() v1.ConfigMap {
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

// GetTestReleaseChartConfigMapsValid returns configmaps for testing release cr
func GetTestReleaseChartConfigMapsValid() []v1.ConfigMap {
	raw, _ := os.Open("../../../testutils/busybox-0.1.0.tgz")

	defer func() {
		if err := raw.Close(); err != nil {
			log.Printf("Error closing file: %s\n", err)
		}
	}()

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
