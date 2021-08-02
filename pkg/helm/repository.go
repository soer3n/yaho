package helm

import (
	"context"
	b64 "encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/apps-operator/internal/types"
	"github.com/soer3n/apps-operator/pkg/utils"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// NewHelmRepo represents initialization of internal repo struct
func NewHelmRepo(instance *helmv1alpha1.Repo, settings *cli.EnvSettings, k8sclient client.Client, g inttypes.HTTPClientInterface, c kube.Client) *Repo {

	var helmRepo *Repo

	log.Debugf("Trying HelmRepo %v", instance.Spec.Name)

	helmRepo = &Repo{
		Name: instance.Spec.Name,
		URL:  instance.Spec.URL,
		Namespace: Namespace{
			Name:    instance.ObjectMeta.Namespace,
			Install: false,
		},
		Settings:   settings,
		k8sClient:  k8sclient,
		getter:     g,
		helmClient: c,
	}

	if instance.Spec.AuthSecret != "" {
		secretObj := &v1.Secret{}
		creds := &Auth{}

		if err := k8sclient.Get(context.Background(), types.NamespacedName{Namespace: instance.ObjectMeta.Namespace, Name: instance.Spec.AuthSecret}, secretObj); err != nil {
			return nil
		}

		if _, ok := secretObj.Data["user"]; !ok {
			log.Info("Username empty for repo auth")
		}

		if _, ok := secretObj.Data["password"]; !ok {
			log.Info("Password empty for repo auth")
		}

		username, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["user"]))
		pw, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["password"]))
		creds.User = string(username)
		creds.Password = string(pw)

		helmRepo.Auth = &Auth{
			User:     strings.TrimSuffix(string(username), "\n"),
			Password: strings.TrimSuffix(string(pw), "\n"),
		}
	}

	return helmRepo
}

func (hr Repo) getIndexByURL() (*repo.IndexFile, error) {

	var parsedURL *url.URL
	var entry *repo.Entry
	var cr *repo.ChartRepository
	var res *http.Response
	var raw []byte
	var err error

	obj := &repo.IndexFile{}

	if entry, err = hr.getEntryObj(); err != nil {
		return obj, errors.Wrapf(err, "error on initializing object for %q.", hr.URL)
	}

	cr = &repo.ChartRepository{
		Config: entry,
	}

	if parsedURL, err = url.Parse(cr.Config.URL); err != nil {
		log.Infof("%v", err)
	}

	parsedURL.RawPath = path.Join(parsedURL.RawPath, "index.yaml")
	parsedURL.Path = path.Join(parsedURL.Path, "index.yaml")
	req, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)

	if err != nil {
		return obj, err
	}

	if hr.Auth != nil {
		if hr.Auth.User != "" && hr.Auth.Password != "" {
			req.SetBasicAuth(hr.Auth.User, hr.Auth.Password)
		}
	}

	if res, err = hr.getter.Do(req); err != nil {
		return obj, err
	}

	log.Infof("URL: %v", parsedURL.String())

	if raw, err = ioutil.ReadAll(res.Body); err != nil {
		return obj, err
	}

	if err := yaml.UnmarshalStrict(raw, &obj); err != nil {
		log.Infof("%v", err)
	}

	return obj, nil
}

// GetCharts represents returning list of internal chart structs for a given repo
func (hr Repo) GetCharts(settings *cli.EnvSettings, selectors map[string]string) ([]*Chart, error) {

	var chartList []*Chart
	var indexFile *repo.IndexFile
	var chartAPIList helmv1alpha1.ChartList
	var err error

	selectorObj := client.MatchingLabels{}

	for k, selector := range selectors {
		selectorObj[k] = selector
	}

	if err = hr.k8sClient.List(context.Background(), &chartAPIList, client.InNamespace(hr.Namespace.Name), selectorObj); err != nil {
		return chartList, err
	}

	for _, v := range chartAPIList.Items {
		chartList = append(chartList, NewChart(utils.ConvertChartVersions(&v), settings, hr.Name, hr.k8sClient, hr.getter, kube.Client{
			Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
			Log:     nopLogger,
		}))
		log.Debugf("new: %v", v)
	}

	if chartList == nil {

		if indexFile, err = hr.getIndexByURL(); err != nil {
			return chartList, err
		}

		log.Debugf("IndexFileErr: %v", err)

		if indexFile == nil {
			return chartList, nil
		}

		for _, chartMetadata := range indexFile.Entries {
			log.Debugf("ChartMetadata: %v", chartMetadata)
			chartList = append(chartList, NewChart(chartMetadata, settings, hr.Name, hr.k8sClient, hr.getter, kube.Client{
				Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
				Log:     nopLogger,
			}))
		}
	}

	return chartList, nil
}

func (hr Repo) getEntryObj() (*repo.Entry, error) {

	obj := &repo.Entry{
		Name: hr.Name,
		URL:  hr.URL,
	}

	if hr.Auth != nil {
		obj.CAFile = hr.Auth.Ca
		obj.CertFile = hr.Auth.Cert
		obj.KeyFile = hr.Auth.Key
		obj.Username = hr.Auth.User
		obj.Password = hr.Auth.Password
	}

	return obj, nil
}
