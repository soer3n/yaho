package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/mocks"
	inttypes "github.com/soer3n/apps-operator/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRepoGetCharts(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()
	apiObjList := getTestRepoSpecs()

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "selector",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "notpresent",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	indexFile := getTestRepoIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/index.yaml").Return(httpResponse, nil)

	req, _ := http.NewRequest(http.MethodGet, "https://foo.bar/charts/index.yaml", nil)

	httpMock.On("Do",
		req).Return(httpResponse, nil)

	reqAuth, _ := http.NewRequest(http.MethodGet, "https://bar.foo/charts/index.yaml", nil)
	reqAuth.SetBasicAuth("foo", "encrypted")

	httpMock.On("Do",
		reqAuth).Return(httpResponse, nil)

	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		val := apiObj.Input.(helmv1alpha1.Repo)
		testObj := NewHelmRepo(&val, settings, &clientMock, &httpMock, kube.Client{})
		selectors := make(map[string]string, 0)

		// parse selectors string from api object meta data
		for k, v := range val.ObjectMeta.Labels {
			selectors[k] = v
		}

		_, err := testObj.GetCharts(settings, selectors)
		assert.Equal(err, apiObj.ReturnError)
	}
}

func getTestRepoChartListSpec() *helmv1alpha1.ChartList {

	chartSpec := helmv1alpha1.ChartSpec{
		Name:        "chart.Name",
		Home:        "chart.Spec.Home",
		Sources:     []string{"chart.Spec.Sources"},
		Description: "chart.Spec.Description",
		Keywords:    []string{"chart.Spec.Keywords"},
		Versions: []helmv1alpha1.ChartVersion{
			{
				Name: "0.1.0",
				URL:  "https://foo.bar/repo/foo.tar.gz",
			},
		},
		Maintainers: []*chart.Maintainer{
			{
				Name: "chart.Spec.Maintainers",
			},
		},
		Icon:        "chart.Spec.Icon",
		APIVersion:  "chart.Spec.APIVersion",
		Condition:   "chart.Spec.Condition",
		Tags:        "chart.Spec.Tags",
		AppVersion:  "chart.Spec.AppVersion",
		Deprecated:  false,
		Annotations: map[string]string{},
		KubeVersion: "chart.Spec.KubeVersion",
		Type:        "chart.Spec.Type",
	}

	return &helmv1alpha1.ChartList{
		Items: []helmv1alpha1.Chart{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: chartSpec,
			},
		},
	}
}

func getTestRepoSpecs() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			Input: helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "selector",
					},
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "test",
					URL:  "https://foo.bar/charts",
				},
			},
			ReturnValue: "",
			ReturnError: nil,
		},
		{
			Input: helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "selector",
					},
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "test",
					URL:  "https://bar.foo/charts",
					Auth: &helmv1alpha1.Auth{
						User:     "foo",
						Password: "encrypted",
						Cert:     "certContent",
						Key:      "keyContent",
						Ca:       "certCa",
					},
				},
			},
			ReturnValue: "",
			ReturnError: nil,
		},
		{
			Input: helmv1alpha1.Repo{
				Spec: helmv1alpha1.RepoSpec{
					Name: "notpresent",
					URL:  "https://foo.bar/charts",
				},
			},
			ReturnValue: "",
			ReturnError: nil,
			//},
			/*{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"label": "notpresent",
					},
				},
				Spec: helmv1alpha1.RepoSpec{
					Name: "notpresent",
					Url:  "https://foo.bar/charts",
					Auth: &helmv1alpha1.Auth{
						User:     "foo",
						Password: "encrypted",
						Cert:     "certContent",
						Key:      "keyContent",
						Ca:       "certCa",
					},
				},
			},*/
		},
	}
}

func getTestRepoIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}

func getExpectedTestRepoCharts(c client.Client) []*Chart {
	return []*Chart{
		{
			Repo:      "testrepo",
			Settings:  cli.New(),
			k8sClient: c,
			Versions: ChartVersions{
				{},
			},
		},
	}
}
