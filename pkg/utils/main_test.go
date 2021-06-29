package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
