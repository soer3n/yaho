package utils

import (
	"testing"

	"github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContains(t *testing.T) {

	testList := []string{"foo", "bar"}

	assert := assert.New(t)

	ok := Contains(testList, "foo")
	assert.True(ok)

	ok = Contains(testList, "fuz")
	assert.False(ok)
}

func TestGetLabelsByInstance(t *testing.T) {

	metaObj := metav1.ObjectMeta{
		Name:      "foo",
		Namespace: "bar",
		Labels: map[string]string{
			"repo": "repo",
		},
	}

	envMap := map[string]string{
		"RepositoryConfig": "/tmp/.config/helm",
		"RepositoryCache":  "/tmp/.cache/helm",
	}

	assert := assert.New(t)

	config, cache := GetLabelsByInstance(metaObj, envMap)
	assert.NotNil(config)
	assert.NotNil(cache)

	metaObj.Labels["repoGroup"] = "group"
	config, cache = GetLabelsByInstance(metaObj, envMap)
	assert.NotNil(config)
	assert.NotNil(cache)
}

func TestGetChartVersion(t *testing.T) {

	testChartSpec := &v1alpha1.Chart{
		Spec: v1alpha1.ChartSpec{
			Versions: []v1alpha1.ChartVersion{
				{
					Name: "0.0.1",
				},
				{
					Name: "0.0.3",
				},
			},
		},
	}

	assert := assert.New(t)

	cv := GetChartVersion("0.0.1", testChartSpec)
	assert.NotNil(cv)

	cv = GetChartVersion("0.0.2", testChartSpec)
	assert.Equal(&v1alpha1.ChartVersion{}, cv)

	cv = GetChartVersion("0.0.3", testChartSpec)
	assert.NotNil(cv)

	cv = GetChartVersion("fasfhoiqef", testChartSpec)
	assert.Equal(&v1alpha1.ChartVersion{}, cv)

	testChartSpec.Spec.Versions = make([]v1alpha1.ChartVersion, 0)
	testChartSpec.Spec.Versions = append(testChartSpec.Spec.Versions, v1alpha1.ChartVersion{
		Name: "fdsifhdsogb",
	})

	cv = GetChartVersion("0.0.1", testChartSpec)
	assert.Equal(&v1alpha1.ChartVersion{}, cv)
}

func TestConvertChartVersions(t *testing.T) {

	testChartSpec := &v1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		Spec: v1alpha1.ChartSpec{
			Name:        "foo",
			Home:        "home",
			Sources:     []string{"source"},
			Description: "desc",
			Keywords:    []string{"key", "word"},
			Maintainers: []*chart.Maintainer{},
			Icon:        "icon",
			APIVersion:  "apiversion",
			Condition:   "cond",
			Tags:        "tags",
			AppVersion:  "appversion",
			Deprecated:  false,
			Annotations: map[string]string{},
			KubeVersion: "new",
			Type:        "type",
			Versions: []v1alpha1.ChartVersion{
				{
					Name: "0.0.1",
				},
				{
					Name: "0.0.3",
					Dependencies: []v1alpha1.ChartDep{
						{
							Name:    "dep",
							Repo:    "dep",
							Version: "0.0.1",
						},
					},
				},
			},
		},
	}
	assert := assert.New(t)

	cvs := ConvertChartVersions(testChartSpec)
	assert.Len(cvs, 2)
}
