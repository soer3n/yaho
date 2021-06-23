package helm

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"path"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	client "github.com/soer3n/apps-operator/pkg/client"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func NewHelmRepo(instance *helmv1alpha1.Repo, settings *cli.EnvSettings, k8sclient client.ClientInterface) *HelmRepo {

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

	parsedURL, err := url.Parse(cr.Config.URL)
	if err != nil {
		log.Infof("%v", err)
	}
	parsedURL.RawPath = path.Join(parsedURL.RawPath, "index.yaml")
	parsedURL.Path = path.Join(parsedURL.Path, "index.yaml")

	b, _ := cr.Client.Get(parsedURL.String(),
		getter.WithURL(cr.Config.URL),
		getter.WithInsecureSkipVerifyTLS(cr.Config.InsecureSkipTLSverify),
		getter.WithTLSClientConfig(cr.Config.CertFile, cr.Config.KeyFile, cr.Config.CAFile),
		getter.WithBasicAuth(cr.Config.Username, cr.Config.Password),
	)

	obj := &repo.IndexFile{}

	log.Infof("URL: %v", parsedURL.String())

	foo, err := ioutil.ReadAll(b)

	if err := yaml.UnmarshalStrict(foo, &obj); err != nil {
		log.Infof("%v", err)
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

	if jsonbody, err = hr.k8sClient.ListResources(hr.Namespace.Name, "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return chartList, err
	}

	if err = json.Unmarshal(jsonbody, &chartApiList); err != nil {
		return chartList, err
	}

	for _, v := range chartApiList.Items {
		chartList = append(chartList, NewChart(v.ConvertChartVersions(), settings, hr.Name, hr.k8sClient))
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
			chartList = append(chartList, NewChart(chartMetadata, settings, hr.Name, hr.k8sClient))
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
