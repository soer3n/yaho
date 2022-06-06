package utils

import (
	"testing"

	"github.com/soer3n/yaho/internal/utils"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContains(t *testing.T) {
	testList := []string{"foo", "bar"}

	assert := assert.New(t)

	ok := utils.Contains(testList, "foo")
	assert.True(ok)

	ok = utils.Contains(testList, "fuz")
	assert.False(ok)
}

func TestGetLabelsByInstance(t *testing.T) {
	metaObj := metav1.ObjectMeta{
		Name:      "foo",
		Namespace: "bar",
		Labels: map[string]string{
			"yaho.soer3n.dev/repo": "repo",
		},
	}

	envMap := map[string]string{
		"RepositoryConfig": "/tmp/.config/helm",
		"RepositoryCache":  "/tmp/.cache/helm",
	}

	assert := assert.New(t)

	config, cache := utils.GetLabelsByInstance(metaObj, envMap)
	assert.NotNil(config)
	assert.NotNil(cache)

	metaObj.Labels["yaho.soer3n.dev/repoGroup"] = "group"
	config, cache = utils.GetLabelsByInstance(metaObj, envMap)
	assert.NotNil(config)
	assert.NotNil(cache)
}

/*
func TestGetChartVersion(t *testing.T) {
	testChartSpec := &v1alpha1.Chart{
		Spec: v1alpha1.ChartSpec{
			Versions: []string{"0.0.1", "0.0.3"},
		},
	}

	assert := assert.New(t)

	cv := utils.GetChartVersion("0.0.1", testChartSpec)
	assert.NotNil(cv)

	cv = utils.GetChartVersion("0.0.2", testChartSpec)
	assert.Equal(&v1alpha1.ChartVersion{Name: "0.0.1"}, cv)

	cv = utils.GetChartVersion("0.0.3", testChartSpec)
	assert.NotNil(cv)

	cv = utils.GetChartVersion("fasfhoiqef", testChartSpec)
	assert.Equal(&v1alpha1.ChartVersion{}, cv)

	testChartSpec.Spec.Versions = make([]string, 0)
	testChartSpec.Spec.Versions = append(testChartSpec.Spec.Versions, "0.0.4")

	cv = utils.GetChartVersion("0.0.1", testChartSpec)
	assert.Equal(&v1alpha1.ChartVersion{Name: "fdsifhdsogb"}, cv)
}

func TestConvertChartVersions(t *testing.T) {
	testChartSpec := &v1alpha1.Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		Spec: v1alpha1.ChartSpec{
			Name:       "foo",
			Versions:   []string{"0.0.1", "0.0.3"},
			Repository: "repo",
			CreateDeps: true,
		},
	}
	assert := assert.New(t)

	cvs := utils.ConvertChartVersions(testChartSpec)
	assert.Len(cvs, 2)
}
*/
