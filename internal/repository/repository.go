package repository

import (
	"context"
	b64 "encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var nopLogger = func(_ string, _ ...interface{}) {}

// New represents initialization of internal repo struct
func New(instance *helmv1alpha1.Repo, settings *cli.EnvSettings, reqLogger logr.Logger, k8sclient client.Client, g utils.HTTPClientInterface, c kube.Client) *Repo {
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
		wg:         &sync.WaitGroup{},
		mu:         sync.Mutex{},
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

// Deploy represents installation of subresources
func (hr *Repo) Deploy(instance *helmv1alpha1.Repo, scheme *runtime.Scheme) error {

	label, repoGroupLabelOk := instance.ObjectMeta.Labels["repoGroup"]
	selector := map[string]string{"repo": hr.Name}

	if repoGroupLabelOk && label != "" {
		selector["repoGroup"] = instance.ObjectMeta.Labels["repoGroup"]
	}

	repo := instance.DeepCopy()

	if err := hr.deployCharts(*repo, selector, scheme); err != nil {
		return err
	}

	hr.logger.Info("chart parsing for %s completed.", "chart", instance.ObjectMeta.Name)

	return nil
}

func (hr *Repo) getIndexByURL() (*repo.IndexFile, error) {
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

func (hr *Repo) getEntryObj() (*repo.Entry, error) {
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
