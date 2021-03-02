package helm

import (
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	oputils "github.com/soer3n/apps-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetHelmClient(instance interface{}) (*HelmClient, error) {

	hc := &HelmClient{
		Repos:    &HelmRepos{},
		Releases: &HelmReleases{},
		Env:      map[string]string{},
	}

	var repoObj *helmv1alpha1.Repo
	var metaStruct metav1.ObjectMeta
	repoObj, ok := instance.(*helmv1alpha1.Repo)

	if ok {
		metaStruct = repoObj.ObjectMeta
	}

	var releaseObj *helmv1alpha1.Release
	releaseObj, ok = instance.(*helmv1alpha1.Release)

	if ok {
		metaStruct = releaseObj.ObjectMeta
	}

	settings := hc.GetEnvSettings()
	hc.Env["RepositoryConfig"] = settings.RepositoryConfig
	hc.Env["RepositoryCache"] = settings.RepositoryCache
	hc.Env["RepositoryConfig"], hc.Env["RepositoryCache"] = oputils.GetLabelsByInstance(metaStruct, hc.Env)

	if err := hc.manageEntries(instance); err != nil {
		return hc, err
	}

	hc.Repos.Settings = hc.GetEnvSettings()
	return hc, nil
}

func (hc *HelmClient) manageEntries(instance interface{}) error {

	repoObj, ok := instance.(helmv1alpha1.Repo)

	if ok {
		_ = hc.setRepo(repoObj)
	}

	releaseObj, ok := instance.(helmv1alpha1.Release)

	if ok {
		if err := hc.setRelease(releaseObj); err != nil {
			return err
		}
	}

	return nil
}

func (hc *HelmClient) setRepo(instance helmv1alpha1.Repo) error {

	var repoList []*HelmRepo
	var helmRepo *HelmRepo

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
	return nil
}

func (hc *HelmClient) setRelease(instance helmv1alpha1.Release) error {
	var releaseList []*HelmRelease
	var helmRelease *HelmRelease

	log.Infof("Trying HelmRepo %v", instance.Spec.Name)

	helmRelease = &HelmRelease{
		Name:     instance.Spec.Name,
		Repo:     instance.Spec.Repo,
		Chart:    instance.Spec.Chart,
		Settings: hc.GetEnvSettings(),
	}

	actionConfig, err := helmRelease.GetActionConfig(helmRelease.Settings)

	if err != nil {
		return err
	}

	helmRelease.Config = actionConfig

	log.Infof("HelmRelease config path: %v", helmRelease.Settings.RepositoryCache)

	if instance.Spec.ValuesTemplate != nil {
		if instance.Spec.ValuesTemplate.Values != nil {
			helmRelease.ValuesTemplate.Values = instance.Spec.ValuesTemplate.Values
		}
		if instance.Spec.ValuesTemplate.ValueFiles != nil {
			helmRelease.ValuesTemplate.ValueFiles = instance.Spec.ValuesTemplate.ValueFiles
		}
	}

	releaseList = append(releaseList, helmRelease)
	hc.Releases.Entries = releaseList
	return nil
}
