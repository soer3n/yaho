package helm

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	client "github.com/soer3n/apps-operator/pkg/client"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func (hr *HelmRepo) Update() error {

	err, entry := hr.GetEntryObj()

	if err != nil {
		return errors.Wrapf(err, "error on initializing object for %q.", hr.Url)
	}

	cr, err := repo.NewChartRepository(entry, getter.All(hr.Settings))

	if err != nil {
		return errors.Wrapf(err, "error on initializing repo %q ", hr.Url)
	}

	cr.CachePath = hr.Settings.RepositoryCache
	_, err = cr.DownloadIndexFile()

	if err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", hr.Url)
	}

	return nil
}

func (hr *HelmRepos) Remove() error {

	err := hr.SetInstalledRepos()

	if err != nil {
		return err
	}

	for key, repository := range hr.installed.Repositories {

		if ok := hr.shouldBeInstalled(repository.Name, repository.URL); ok == false {
			log.Debugf("Removing repo: index: %v name: %v", key, repository.Name)
			ok := hr.installed.Remove(repository.Name)

			if !ok {
				return errors.Errorf("Error removing repository %q.", repository.Name)
			}

			err := hr.RemoveRepoCache(repository.Name)

			if err != nil {
				return errors.Errorf("Error removing repository cache for %q.", repository.Name)
			}
		}
	}

	return hr.installed.WriteFile(hr.Settings.RepositoryConfig, 0644)
}

func (hr *HelmRepos) RemoveByName(name string) error {

	ok := hr.installed.Remove(name)

	if !ok {
		return errors.Errorf("Error removing repository %q.", name)
	}

	err := hr.RemoveRepoCache(name)

	if err != nil {
		return errors.Errorf("Error removing repository cache for %q.", name)
	}

	return hr.installed.WriteFile(hr.Settings.RepositoryConfig, 0644)
}

func (hr *HelmRepos) RemoveRepoCache(name string) error {

	if err := removeFile(hr.Settings.RepositoryCache, helmpath.CacheIndexFile(name)); err != nil {
		return err
	}

	if err := removeFile(hr.Settings.RepositoryCache, helmpath.CacheChartsFile(name)); err != nil {
		return err
	}

	return nil
}

func (hr *HelmRepos) shouldBeInstalled(name, url string) bool {

	for key, repository := range hr.Entries {
		if repository.Name == name && repository.Url == url {
			log.Debugf("Repo validation failed: index: %v name: %v already exists", key, repository.Name)
			return true
		}
	}

	return false
}

func (hr HelmRepos) Get(name string) (error, *repo.Entry) {

	return nil, hr.installed.Get(name)
}

func (hr *HelmRepo) GetCharts(settings *cli.EnvSettings, selector string) ([]*HelmChart, error) {

	var chartList []*HelmChart
	var indexFile *repo.IndexFile
	var jsonbody []byte
	var chartApiList helmv1alpha1.ChartList
	var err error

	rc := client.New()
	_ = rc.SetClient()

	jsonbody, err = rc.SetOptions(metav1.ListOptions{
		LabelSelector: selector,
	}).ListResources(hr.Namespace.Name, "charts", "helm.soer3n.info", "v1alpha1")

	if err = json.Unmarshal(jsonbody, &chartApiList); err != nil {
		return chartList, err
	}

	for _, v := range chartApiList.Items {
		chartList = append(chartList, NewChart(v.ConvertChartVersions(), settings, hr.Name))
		log.Debugf("new: %v", v)
	}

	if chartList == nil {

		if indexFile, err = repo.LoadIndexFile(hr.Settings.RepositoryCache + "/" + hr.Name + "-index.yaml"); err != nil {
			_ = hr.Update()
			indexFile, err = repo.LoadIndexFile(hr.Settings.RepositoryCache + "/" + hr.Name + "-index.yaml")
		}

		log.Debugf("IndexFileErr: %v", err)

		for _, chartMetadata := range indexFile.Entries {
			log.Debugf("ChartMetadata: %v", chartMetadata)
			chartList = append(chartList, NewChart(chartMetadata, settings, hr.Name))
		}
	}

	return chartList, nil
}

func (hr HelmRepo) GetEntryObj() (error, *repo.Entry) {

	return nil, &repo.Entry{
		Name: hr.Name,
		URL:  hr.Url,
	}
}

func (hr *HelmRepos) SetInstalledRepos() error {

	f, err := repo.LoadFile(hr.Settings.RepositoryConfig)

	if err != nil {
		err = f.WriteFile(hr.Settings.RepositoryConfig, 0644)
	}

	hr.installed = f
	return err
}

func (hr *HelmRepos) UpdateRepoFile(entry *repo.Entry) error {

	f, err := hr.readRepoFile()

	if err != nil {
		return err
	}

	log.Debugf("Repos before updating: %v", f)
	f.Update(entry)
	log.Debugf("Repos after updating: %v", f)
	err = f.WriteFile(hr.Settings.RepositoryConfig, 0644)

	if err != nil {
		return err
	}

	log.Debugf("%q has been added to your repositories", entry.Name)

	return nil
}

func (hr *HelmRepos) readRepoFile() (*repo.File, error) {

	b, err := ioutil.ReadFile(hr.Settings.RepositoryConfig)

	var f repo.File

	if err != nil && !os.IsNotExist(err) {
		return &f, err
	}

	if err := yaml.Unmarshal(b, &f); err != nil {
		return &f, err
	}

	return &f, nil
}

func (hr *HelmRepo) readRepoIndexFile() (*repo.IndexFile, error) {

	b, err := ioutil.ReadFile(hr.Settings.RepositoryCache + "/" + hr.Name + "-index.yaml")

	var f repo.IndexFile

	if err != nil && !os.IsNotExist(err) {
		return &f, err
	}

	if err := yaml.Unmarshal(b, &f); err != nil {
		return &f, err
	}

	return &f, nil
}

func (hr *HelmRepos) prepare() error {

	if hr.installed == nil {
		err := hr.SetInstalledRepos()

		if err != nil {
			return err
		}
	}

	return nil
}

func (hr *HelmRepos) Validate() error {

	hr.prepare()

	for key, repository := range hr.Entries {

		if err := hr.ValidateRepo(repository.Name, repository.Url); err != nil {
			log.Errorf("Repo validation error: index: %v name: %v", key, repository.Name)
			return err
		}
	}

	return nil
}

func (hr *HelmRepos) ValidateRepo(name string, url string) error {

	hr.prepare()

	// Check if Name is already set for other Repo
	if hr.installed.Has(name) && hr.installed.Get(name).URL != url {
		return errors.Errorf("Other Repo with that name already exists: %s", name)
	}

	return nil
}
