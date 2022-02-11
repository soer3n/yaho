package helm

import (
	"log"
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/release"
	"github.com/soer3n/yaho/internal/values"
	"github.com/soer3n/yaho/tests/mocks"
	helmmocks "github.com/soer3n/yaho/tests/mocks/helm"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestReleaseConfigMaps(t *testing.T) {
	clientMock, httpMock := helmmocks.GetReleaseMock()
	settings := cli.New()
	apiObjList := testcases.GetTestReleaseSpecsForConfigMaps()
	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		current := apiObj.Input.(*helmv1alpha1.Release)
		testObj := release.New(current, settings, logf.Log, clientMock, httpMock, kube.Client{})
		selectors := ""

		// parse selectors string from api object meta data
		for k, v := range current.ObjectMeta.Labels {
			if selectors != "" {
				selectors = selectors + ","
			}
			selectors = selectors + k + "=" + v
		}

		testObj.Version = current.Spec.Version
		cmList, chartUpdateList := testObj.GetParsedConfigMaps("")
		// TODO: why is dependency chart not correctly parsed
		expect, _ := apiObj.ReturnValue.(map[string]int)

		assert.Len(cmList, expect["configmap"])
		assert.Len(chartUpdateList, expect["chart"])
	}
}

func TestReleaseUpdate(t *testing.T) {
	clientMock, httpMock := helmmocks.GetReleaseMock()
	settings := cli.New()
	apiObjList := testcases.GetTestReleaseSpecs()
	assert := assert.New(t)

	for _, apiObj := range apiObjList {

		current := apiObj.Input.(*helmv1alpha1.Release)
		testObj := release.New(current, settings, logf.Log, clientMock, httpMock, kube.Client{})
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
		testObj.Config = testcases.GetTestReleaseFakeActionConfig(t)

		if err := testObj.Config.Releases.Create(testcases.GetTestReleaseDeployedReleaseObj()); err != nil {
			log.Print(err)
		}

		err := testObj.Update(helmv1alpha1.Namespace{
			Name:    "",
			Install: false,
		})
		assert.Equal(apiObj.ReturnError, err)
	}
}

func TestReleaseInitValuesTemplate(t *testing.T) {
	clientMock := unstructuredmocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}
	settings := cli.New()
	apiObjList := testcases.GetTestReleaseValueRefListSpec()
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

		current := apiObj.Input.([]*values.ValuesRef)
		testObj := release.New(testRelease, settings, logf.Log, &clientMock, &httpMock, kube.Client{})
		testObj.InitValuesTemplate(current, "namespace", "v0.0.1")
		expect := apiObj.ReturnValue.(map[string]interface{})
		assert.Equal(expect, testObj.Values)
	}
}
