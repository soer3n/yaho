package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClient(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

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

	indexFile := getTestClientIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/index.yaml").Return(httpResponse, nil)

	assert := assert.New(t)

	testObj := NewHelmClient(getTestClientRepo(), &clientMock, &httpMock)
	repoObj := testObj.GetRepo("")

	assert.Nil(repoObj)

	testObj = NewHelmClient(getTestClientRelease(), &clientMock, &httpMock)
	releaseObj := testObj.GetRelease("", "")

	assert.Nil(releaseObj)
}

func getTestClientRepo() *Repo {
	return &Repo{
		Name: "boo",
		URL:  "https://unknown.domain/charts",
	}
}

func getTestClientRelease() *Release {
	return &Release{
		Name:    "far",
		Repo:    "boo",
		Chart:   "foo",
		Version: "0.0.1",
	}
}

func getTestClientIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}
