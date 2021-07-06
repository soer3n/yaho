package helm

import (
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/apps-operator/internal/mocks"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetEnv(t *testing.T) {

	clientMock := mocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

	assert := assert.New(t)

	testObj := NewHelmClient(getTestMainRelease(), &clientMock, &httpMock)
	testObj.Env = map[string]string{
		"KubeConfig":       "a",
		"KubeContext":      "b",
		"KubeAsUser":       "c",
		"KubeAsGroups":     "d",
		"KubeAPIServer":    "e",
		"RegistryConfig":   "f",
		"RepositoryConfig": "g",
		"RepositoryCache":  "h",
		"PluginsDirectory": "i",
		"KubeToken":        "j",
	}

	settings := testObj.GetEnvSettings()

	// assert.Equal(expected, charts, "Structs should be equal.")
	assert.NotNil(settings)
}

func getTestMainRelease() *helmv1alpha1.Release {
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
