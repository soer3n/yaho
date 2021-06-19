package helm

import (
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	oputils "github.com/soer3n/apps-operator/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewHelmClient(instance interface{}, k8sClient *client.Client) *HelmClient {

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
		return hc
	}

	hc.Repos.Settings = hc.GetEnvSettings()
	return hc
}

func (hc *HelmClient) GetRepo(name string) *HelmRepo {

	for _, repo := range hc.Repos.Entries {
		if name == repo.Name {
			return repo
		}
	}

	return &HelmRepo{}
}

func (hc *HelmClient) GetRelease(name, repo string) *HelmRelease {

	for _, release := range hc.Releases.Entries {
		if release.Name == name && release.Repo == repo {
			return release
		}
	}
	return &HelmRelease{}
}

func (hc *HelmClient) manageEntries(instance interface{}) error {

	var releaseObj *helmv1alpha1.Release
	repoObj, ok := instance.(*helmv1alpha1.Repo)
	settings := hc.GetEnvSettings()
	actionConfig, _ := initActionConfig(settings)

	if ok {
		hc.Repos.Entries = append(hc.Repos.Entries, NewHelmRepo(repoObj, settings))
	}

	if releaseObj, ok = instance.(*helmv1alpha1.Release); ok {
		hc.Releases.Entries = append(hc.Releases.Entries, NewHelmRelease(releaseObj, settings, actionConfig))
	}

	return nil
}
