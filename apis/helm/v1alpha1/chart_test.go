package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetChartVersion(t *testing.T) {

	testChartSpec := &Chart{
		Spec: ChartSpec{
			Versions: []ChartVersion{
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

	cv := testChartSpec.GetChartVersion("0.0.1")
	assert.NotNil(cv)

	cv = testChartSpec.GetChartVersion("0.0.2")
	assert.Equal(&ChartVersion{}, cv)

	cv = testChartSpec.GetChartVersion("0.0.3")
	assert.NotNil(cv)

	cv = testChartSpec.GetChartVersion("fasfhoiqef")
	assert.Equal(&ChartVersion{}, cv)

	testChartSpec.Spec.Versions = make([]ChartVersion, 0)
	testChartSpec.Spec.Versions = append(testChartSpec.Spec.Versions, ChartVersion{
		Name: "fdsifhdsogb",
	})

	cv = testChartSpec.GetChartVersion("0.0.1")
	assert.Equal(&ChartVersion{}, cv)
}

func TestConvertChartVersions(t *testing.T) {

	testChartSpec := &Chart{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
		Spec: ChartSpec{
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
			Versions: []ChartVersion{
				{
					Name: "0.0.1",
				},
				{
					Name: "0.0.3",
					Dependencies: []ChartDep{
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

	cvs := testChartSpec.ConvertChartVersions()
	assert.Len(cvs, 2)
}
