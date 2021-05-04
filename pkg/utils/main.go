package utils

import (
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

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
