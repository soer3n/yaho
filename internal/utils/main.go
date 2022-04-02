package utils

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"path/filepath"

	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Contains represents func for checking if a string is in a list of strings
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func LoadChartIndex(chart, repository, namespace string, c client.Client) (*repo.ChartVersions, error) {

	var rawData []byte

	obj := &v1.ConfigMap{}
	var versions *repo.ChartVersions

	if err := c.Get(context.Background(), types.NamespacedName{
		Name:      "helm-" + repository + "-" + chart + "-index",
		Namespace: namespace,
	}, obj); err != nil {
		return nil, err
	}

	rawData = obj.BinaryData["versions"]

	if err := json.Unmarshal(rawData, &versions); err != nil {
		return nil, err
	}

	return versions, nil
}

// GetLabelsByInstance represents func for parsing labels by k8s objectMeta and env map
func GetLabelsByInstance(metaObj metav1.ObjectMeta, env map[string]string) (string, string) {
	var repoPath, repoCache string

	repoPath = filepath.Dir(env["RepositoryConfig"])
	repoCache = env["RepositoryCache"]

	repoLabel, repoLabelOk := metaObj.Labels["repo"]
	repoGroupLabel, repoGroupLabelOk := metaObj.Labels["repoGroup"]

	if repoLabelOk {
		if repoGroupLabelOk {
			repoPath = repoPath + "/" + metaObj.Namespace + "/" + repoGroupLabel
			repoCache = repoCache + "/" + metaObj.Namespace + "/" + repoGroupLabel
		} else {
			repoPath = repoPath + "/" + metaObj.Namespace + "/" + repoLabel
			repoCache = repoCache + "/" + metaObj.Namespace + "/" + repoLabel
		}
	}

	return repoPath + "/repositories.yaml", repoCache
}

// RandomString return a string with random chars of length n
func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		n, _ := rand.Int(rand.Reader, (big.NewInt(30)))
		b[i] = letters[n.Uint64()]
	}
	return string(b)
}
