package helm

import (
	"context"
	b64 "encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/internal/types"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var nopLogger = func(_ string, _ ...interface{}) {}

// NewHelmRepo represents initialization of internal repo struct
func NewHelmRepo(instance *helmv1alpha1.Repo, settings *cli.EnvSettings, reqLogger logr.Logger, k8sclient client.Client, g inttypes.HTTPClientInterface, c kube.Client) *Repo {
	var helmRepo *Repo

	reqLogger.Info("Trying HelmRepo", "repo", instance.Spec.Name)

	helmRepo = &Repo{
		Name: instance.Spec.Name,
		URL:  instance.Spec.URL,
		Namespace: Namespace{
			Name:    instance.ObjectMeta.Namespace,
			Install: false,
		},
		Settings:   settings,
		K8sClient:  k8sclient,
		getter:     g,
		helmClient: c,
		logger:     reqLogger.WithValues("repo", instance.Spec.Name),
	}

	if instance.Spec.AuthSecret != "" {
		secretObj := &v1.Secret{}
		creds := &Auth{}

		if err := k8sclient.Get(context.Background(), types.NamespacedName{Namespace: instance.ObjectMeta.Namespace, Name: instance.Spec.AuthSecret}, secretObj); err != nil {
			return nil
		}

		if _, ok := secretObj.Data["user"]; !ok {
			reqLogger.Info("Username empty for repo auth")
		}

		if _, ok := secretObj.Data["password"]; !ok {
			reqLogger.Info("Password empty for repo auth")
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
		return obj, errors.Wrapf(err, "error on initializing object %v with url %v", hr.Name, hr.URL)
	}

	cr = &repo.ChartRepository{
		Config: entry,
	}

	if parsedURL, err = url.Parse(cr.Config.URL); err != nil {
		hr.logger.Error(err, "failed on parsing url", "name", hr.Name, "url", hr.URL)
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

	if raw, err = ioutil.ReadAll(res.Body); err != nil {
		return obj, err
	}

	if err := yaml.UnmarshalStrict(raw, &obj); err != nil {
		hr.logger.Error(err, "error on unmarshaling http body to index file")
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

	if err = hr.K8sClient.List(context.Background(), &chartAPIList, client.InNamespace(hr.Namespace.Name), selectorObj); err != nil {
		return chartList, err
	}

	for i := range chartAPIList.Items {
		chartList = append(chartList, NewChart(utils.ConvertChartVersions(&chartAPIList.Items[i]), settings, hr.logger, hr.Name, hr.K8sClient, hr.getter, kube.Client{
			Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
			Log:     nopLogger,
		}))
		hr.logger.Info("new..", "name", chartAPIList.Items[i].ObjectMeta.Name)
	}

	if chartList == nil {

		if indexFile, err = hr.getIndexByURL(); err != nil {
			hr.logger.Error(err, "error on getting repo index file")
			return chartList, err
		}

		if indexFile == nil {
			return chartList, nil
		}

		for _, chartMetadata := range indexFile.Entries {
			hr.logger.Info("initializing chart struct by metadata", "repo", hr.Name)
			chartList = append(chartList, NewChart(chartMetadata, settings, hr.logger, hr.Name, hr.K8sClient, hr.getter, kube.Client{
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
