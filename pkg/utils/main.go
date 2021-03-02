package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
)

func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func GetLabelsByInstance(instance metav1.ObjectMeta, env map[string]string) (string, string) {

	var repoPath, repoCache string

	repoPath = filepath.Dir(env["RepositoryConfig"])
	repoCache = env["RepositoryCache"]

	repoLabel, repoLabelOk := instance.Labels["repo"]
	repoGroupLabel, repoGroupLabelOk := instance.Labels["repoGroup"]

	if repoLabelOk {
		if repoGroupLabelOk {
			repoPath = repoPath + "/" + instance.Namespace + "/" + repoGroupLabel + "/repositories.yaml"
			repoCache = repoCache + "/" + instance.Namespace + "/" + repoGroupLabel
		} else {
			repoPath = repoPath + "/" + instance.Namespace + "/" + repoLabel + "/repositories.yaml"
			repoCache = repoCache + "/" + instance.Namespace + "/" + repoLabel
		}
	}

	if _, ok := instance.Labels["release"]; !ok {

		instance.Labels = map[string]string{
			"release": instance.Name,
		}
	}

	return repoPath, repoCache
}
