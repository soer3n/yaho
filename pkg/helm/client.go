package helm

import (
	"github.com/prometheus/common/log"
	oputils "github.com/soer3n/apps-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetHelmClient(instance interface{}) (*HelmClient, error) {

	var metaObj *metav1.ObjectMeta
	metaObj, ok := instance.(*metav1.ObjectMeta)

	hc := &HelmClient{
		Repos: &HelmRepos{},
		Env:   map[string]string{},
	}

	settings := hc.GetEnvSettings()
	hc.Env["RepositoryConfig"] = settings.RepositoryConfig
	hc.Env["RepositoryCache"] = settings.RepositoryCache

	var repoList []*HelmRepo
	var helmRepo *HelmRepo

	hc.Env["RepositoryConfig"], hc.Env["RepositoryCache"] = oputils.GetLabelsByInstance(metaObj, hc.Env)

	log.Infof("Trying HelmRepo %v", instance.Spec.Name)

	helmRepo = &HelmRepo{
		Name:     instance.Spec.Name,
		Url:      instance.Spec.Url,
		Settings: hc.GetEnvSettings(),
	}

	if instance.Spec.Auth != nil {
		helmRepo.Auth = HelmAuth{
			User:     instance.Spec.Auth.User,
			Password: instance.Spec.Auth.Password,
			Cert:     instance.Spec.Auth.Cert,
			Key:      instance.Spec.Auth.Key,
			Ca:       instance.Spec.Auth.Ca,
		}
	}

	repoList = append(repoList, helmRepo)

	hc.Repos.Entries = repoList
	hc.Repos.Settings = hc.GetEnvSettings()
	return hc, nil
}
