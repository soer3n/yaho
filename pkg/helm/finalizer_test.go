package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/mocks"
	inttypes "github.com/soer3n/apps-operator/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestFinalizerHandle(t *testing.T) {

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

	indexFile := getTestFinalizerIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/index.yaml").Return(httpResponse, nil)

	assert := assert.New(t)

	testObj := NewHelmClient(getTestFinalizerRelease(), &clientMock, &httpMock)
	testObj.Releases.Entries[0].Config = getTestFinalizerFakeActionConfig(t)

	if err := testObj.Releases.Entries[0].Config.Releases.Create(getTestFinalizerDeployedReleaseObj()); err != nil {
		log.
			Print(err)
	}

	for _, v := range getTestFinalizerSpecs() {

		ok, err := HandleFinalizer(testObj, v)

		//ok, _ := HandleFinalizer(testObj, getTestClientRepo())

		// assert.Equal(expected, charts, "Structs should be equal.")
		//assert.True(ok)

		// assert.Equal(expected, charts, "Structs should be equal.")
		assert.Equal(v.ReturnValue, ok)
		assert.Equal(v.ReturnError, err)
	}

}

func getTestFinalizerRepo() *helmv1alpha1.Repo {
	return &helmv1alpha1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "",
		},
		Spec: helmv1alpha1.RepoSpec{
			Name: "repo",
			URL:  "https://foo.bar/charts",
		},
	}
}

func getTestFinalizerRelease() *helmv1alpha1.Release {
	return &helmv1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "release",
			Namespace: "",
		},
		Spec: helmv1alpha1.ReleaseSpec{
			Name:  "release",
			Repo:  "repo",
			Chart: "chart",
		},
	}
}

func getTestFinalizerFakeActionConfig(t *testing.T) *action.Configuration {
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

func getTestFinalizerDeployedReleaseObj() *release.Release {
	return &release.Release{
		Name:  "release",
		Chart: getTestHelmChart(),
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
	}
}

func getTestFinalizerIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}

func getTestFinalizerSpecs() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnError: nil,
			ReturnValue: true,
			Input:       getTestClientRelease(),
		},
	}
}
