package helm

import (
	"github.com/pkg/errors"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	oputils "github.com/soer3n/apps-operator/pkg/utils"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
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

func (hc *HelmClient) RemoveByName(name string) error {

	if ok := hc.Repos.installed.Remove(name); !ok {
		return errors.Errorf("Error removing repository %q.", name)
	}

	if err := hc.RemoveRepoCache(name); err != nil {
		return errors.Errorf("Error removing repository cache for %q.", name)
	}

	return hc.Repos.installed.WriteFile(hc.GetEnvSettings().RepositoryConfig, 0644)
}

func (hc *HelmClient) RemoveRepoCache(name string) error {

	if err := removeFile(hc.GetEnvSettings().RepositoryCache, helmpath.CacheIndexFile(name)); err != nil {
		return err
	}

	if err := removeFile(hc.GetEnvSettings().RepositoryCache, helmpath.CacheChartsFile(name)); err != nil {
		return err
	}

	return nil
}

func (hc *HelmClient) setInstalledRepos() error {

	var f *repo.File
	var err error

	if f, err = repo.LoadFile(hc.GetEnvSettings().RepositoryConfig); err != nil {
		if err = f.WriteFile(hc.GetEnvSettings().RepositoryConfig, 0644); err != nil {
			return err
		}
	}

	hc.Repos.installed = f
	return nil
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
