package repository

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const configMapLabelKey = "yaho.soer3n.dev/chart"
const configMapRepoLabelKey = "yaho.soer3n.dev/repo"
const configMapLabelType = "yaho.soer3n.dev/type"

// New represents initialization of internal repo struct
func New(instance *helmv1alpha1.Repository, namespace string, ctx context.Context, settings *cli.EnvSettings, reqLogger logr.Logger, k8sclient client.Client, g utils.HTTPClientInterface, c kube.Client) *Repo {
	var helmRepo *Repo

	reqLogger.Info("Trying HelmRepo", "repo", instance.Spec.Name)

	helmRepo = &Repo{
		Name: instance.Spec.Name,
		URL:  instance.Spec.URL,
		Namespace: Namespace{
			Name:    namespace,
			Install: false,
		},
		Settings:   settings,
		K8sClient:  k8sclient,
		getter:     g,
		helmClient: c,
		logger:     reqLogger.WithValues("repo", instance.Spec.Name),
		wg:         &sync.WaitGroup{},
		mu:         sync.Mutex{},
		ctx:        ctx,
	}

	if instance.Spec.AuthSecret != "" {
		secretObj := &v1.Secret{}
		creds := &Auth{}

		if err := k8sclient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: instance.Spec.AuthSecret}, secretObj); err != nil {
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

	indexFile, err := helmRepo.getIndexByURL()

	if err != nil {
		helmRepo.logger.Error(err, "error on getting repo index file")
	}

	helmRepo.index = indexFile

	return helmRepo
}

func (hr *Repo) Update(instance *helmv1alpha1.Repository, scheme *runtime.Scheme) error {

	if err := hr.createIndexConfigmaps(instance, scheme); err != nil {
		return err
	}

	label, repoGroupLabelOk := instance.ObjectMeta.Labels["repoGroup"]
	selector := map[string]string{"repo": hr.Name}

	if repoGroupLabelOk && label != "" {
		selector["repoGroup"] = instance.ObjectMeta.Labels["repoGroup"]
	}

	repo := instance.DeepCopy()

	// fetch installed charts related to repository resource
	installedCharts := &helmv1alpha1.ChartList{}
	requirement, _ := labels.ParseToRequirements("repo=" + hr.Name)
	opts := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(requirement[0]),
	}

	if err := hr.K8sClient.List(context.Background(), installedCharts, opts); err != nil {
		hr.logger.Info("Error on listing charts for repository %v", hr.Name)
	}

	for _, item := range installedCharts.Items {

		// skip if chart resource is created manually
		if _, ok := item.ObjectMeta.Labels["unmanaged"]; ok {
			continue
		}

		contains := false
		for _, chart := range instance.Spec.Charts {

			if chart.Name == item.Spec.Name {
				contains = true
				break
			}
		}

		if !contains {
			hr.logger.Info("deleting chart", "repo", hr.Name, "chart", item.ObjectMeta.Name)

			obj := item.DeepCopy()
			if err := hr.K8sClient.Delete(context.Background(), obj); err != nil {
				hr.logger.Info("error on deleting chart", "repo", hr.Name, "chart", item.ObjectMeta.Name)
			}
		}

	}

	for _, chart := range instance.Spec.Charts {
		if err := hr.deployChart(repo, chart, selector, scheme); err != nil {
			return err
		}
	}

	hr.logger.Info("chart parsing for %s completed.", "chart", instance.ObjectMeta.Name)

	return nil
}

func (hr *Repo) createIndexConfigmaps(instance *helmv1alpha1.Repository, scheme *runtime.Scheme) error {

	for chart, versions := range hr.index.Entries {

		if len(instance.Spec.Charts) > 0 {
			ok := false
			for _, sc := range instance.Spec.Charts {
				if sc.Name == chart {
					ok = true
					break
				}
			}

			if !ok {
				continue
			}
		}

		list, err := json.Marshal(versions)

		if err != nil {
			hr.logger.Error(err, "error on marshaling chart versions")
			continue
		}

		data := map[string][]byte{}
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "helm-" + hr.Name + "-" + chart + "-index",
				Namespace: hr.Namespace.Name,
				Labels: map[string]string{
					configMapRepoLabelKey: hr.Name,
					configMapLabelKey:     chart,
					configMapLabelType:    "index",
				},
			},
			BinaryData: map[string][]byte{},
		}

		data["versions"] = list

		cm.BinaryData = data

		if err := controllerutil.SetControllerReference(instance, cm, scheme); err != nil {
			hr.logger.Error(err, "failed to set owner ref for chart", "chart", chart)
		}

		if err := hr.K8sClient.Create(hr.ctx, cm); err != nil {
			if k8serrors.IsAlreadyExists(err) {
				hr.logger.Info("chart configmap already exists", "chart", chart, "name", cm.ObjectMeta.Name)
				if err := hr.K8sClient.Update(hr.ctx, cm); err != nil {
					hr.logger.Info("could not update repository index chart configmap", "chart", chart, "name", cm.ObjectMeta.Name)
					return err
				}
				hr.logger.Info("chart configmap of repository index configmap updated", "chart", chart, "name", cm.ObjectMeta.Name)
				return nil
			}
			hr.logger.Info("chart configmap of repository index configmap created", "chart", chart, "name", cm.ObjectMeta.Name)
		}
	}

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

	hr.logger.Info("download index", "url", parsedURL.String())

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
