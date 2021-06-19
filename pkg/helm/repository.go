package helm

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	client "github.com/soer3n/apps-operator/pkg/client"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewHelmRepo(instance *helmv1alpha1.Repo, settings *cli.EnvSettings, k8sclient *client.Client) *HelmRepo {

	var helmRepo *HelmRepo

	log.Debugf("Trying HelmRepo %v", instance.Spec.Name)

	helmRepo = &HelmRepo{
		Name: instance.Spec.Name,
		Url:  instance.Spec.Url,
		Namespace: Namespace{
			Name:    instance.ObjectMeta.Namespace,
			Install: false,
		},
		Settings:  settings,
		k8sClient: k8sclient,
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

	return helmRepo
}

func (hr HelmRepo) Update() error {

	var entry *repo.Entry
	var cr *repo.ChartRepository
	var err error

	if err, entry = hr.GetEntryObj(); err != nil {
		return errors.Wrapf(err, "error on initializing object for %q.", hr.Url)
	}

	if cr, err = repo.NewChartRepository(entry, getter.All(hr.Settings)); err != nil {
		return errors.Wrapf(err, "error on initializing repo %q ", hr.Url)
	}

	cr.CachePath = hr.Settings.RepositoryCache

	if _, err = cr.DownloadIndexFile(); err != nil {
		return errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", hr.Url)
	}

	return nil
}

func (hr HelmRepos) shouldBeInstalled(name, url string) bool {

	for key, repository := range hr.Entries {
		if repository.Name == name && repository.Url == url {
			log.Debugf("Repo validation failed: index: %v name: %v already exists", key, repository.Name)
			return true
		}
	}

	return false
}

func (hr HelmRepo) GetCharts(settings *cli.EnvSettings, selector string) ([]*HelmChart, error) {

	var chartList []*HelmChart
	var indexFile *repo.IndexFile
	var chartApiList helmv1alpha1.ChartList
	var jsonbody []byte
	var err error

	if jsonbody, err = hr.k8sClient.SetOptions(metav1.ListOptions{
		LabelSelector: selector,
	}).ListResources(hr.Namespace.Name, "charts", "helm.soer3n.info", "v1alpha1"); err != nil {
		return chartList, err
	}

	if err = json.Unmarshal(jsonbody, &chartApiList); err != nil {
		return chartList, err
	}

	for _, v := range chartApiList.Items {
		chartList = append(chartList, NewChart(v.ConvertChartVersions(), settings, hr.Name))
		log.Debugf("new: %v", v)
	}

	if chartList == nil {

		if indexFile, err = repo.LoadIndexFile(hr.Settings.RepositoryCache + "/" + hr.Name + "-index.yaml"); err != nil {
			if err = hr.Update(); err != nil {
				return chartList, err
			}
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
