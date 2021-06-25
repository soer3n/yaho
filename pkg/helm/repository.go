package helm

import (
	"bytes"
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

func NewHelmRepo(instance *helmv1alpha1.Repo, settings *cli.EnvSettings, k8sclient client.ClientInterface, g getter.Getter) *HelmRepo {

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
		getter:    g,
	}

	if instance.Spec.Auth != nil {
		helmRepo.Auth = &HelmAuth{
			User:     instance.Spec.Auth.User,
			Password: instance.Spec.Auth.Password,
			Cert:     instance.Spec.Auth.Cert,
			Key:      instance.Spec.Auth.Key,
			Ca:       instance.Spec.Auth.Ca,
		}
	}

	return helmRepo
}

func (hr HelmRepo) getIndexByUrl() (*repo.IndexFile, error) {

	var parsedURL *url.URL
	var entry *repo.Entry
	var cr *repo.ChartRepository
	var res *bytes.Buffer
	var raw []byte
	var err error

	obj := &repo.IndexFile{}

	if err, entry = hr.GetEntryObj(); err != nil {
		return obj, errors.Wrapf(err, "error on initializing object for %q.", hr.Url)
	}

	cr = &repo.ChartRepository{
		Config: entry,
		Client: hr.getter,
	}

	if parsedURL, err = url.Parse(cr.Config.URL); err != nil {
		log.Infof("%v", err)
	}

	parsedURL.RawPath = path.Join(parsedURL.RawPath, "index.yaml")
	parsedURL.Path = path.Join(parsedURL.Path, "index.yaml")

	if res, err = cr.Client.Get(parsedURL.String(),
		getter.WithURL(cr.Config.URL),
		getter.WithInsecureSkipVerifyTLS(cr.Config.InsecureSkipTLSverify),
		getter.WithTLSClientConfig(cr.Config.CertFile, cr.Config.KeyFile, cr.Config.CAFile),
		getter.WithBasicAuth(cr.Config.Username, cr.Config.Password),
	); err != nil {
		return obj, err
	}

	log.Infof("URL: %v", parsedURL.String())

	if raw, err = ioutil.ReadAll(res); err != nil {
		return obj, err
	}

	if err := yaml.UnmarshalStrict(raw, &obj); err != nil {
		log.Infof("%v", err)
	}

	return obj, nil
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
	var chartAPIList helmv1alpha1.ChartList
	var jsonbody []byte
	var err error

	if jsonbody, err = hr.k8sClient.ListResources(hr.Namespace.Name, "charts", "helm.soer3n.info", "v1alpha1", metav1.ListOptions{
		LabelSelector: selector,
	}); err != nil {
		return chartList, err
	}

	if err = json.Unmarshal(jsonbody, &chartAPIList); err != nil {
		return chartList, err
	}

	for _, v := range chartAPIList.Items {
		chartList = append(chartList, NewChart(v.ConvertChartVersions(), settings, hr.Name, hr.k8sClient))
		log.Debugf("new: %v", v)
	}

	if chartList == nil {

		if indexFile, err = hr.getIndexByUrl(); err != nil {
			return chartList, err
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

	obj := &repo.Entry{
		Name: hr.Name,
		URL:  hr.Url,
	}

	if hr.Auth != nil {
		obj.CAFile = hr.Auth.Ca
		obj.CertFile = hr.Auth.Cert
		obj.KeyFile = hr.Auth.Key
		obj.Username = hr.Auth.User
		obj.Password = hr.Auth.Password
	}

	return nil, obj
}
