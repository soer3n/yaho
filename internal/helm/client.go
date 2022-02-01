package helm

import (
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/types"
	oputils "github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var nopLogger = func(_ string, _ ...interface{}) {}

// NewHelmClient represents initialization of the client for managing related stuff
func NewHelmClient(instance interface{}, k8sClient client.Client, g types.HTTPClientInterface) *Client {
	hc := &Client{
		Repos:    &Repos{},
		Releases: &Releases{},
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

	c := kube.Client{
		Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
		Log:     nopLogger,
	}

	if err := hc.manageEntries(instance, k8sClient, g, c); err != nil {
		return hc
	}

	hc.Repos.Settings = settings
	return hc
}

// GetRepo represents func for returning internal repo struct by name
func (hc *Client) GetRepo(name string) *Repo {
	for _, repo := range hc.Repos.Entries {
		if name == repo.Name {
			return repo
		}
	}

	return nil
}

// GetRelease represents func for returning internal release struct by name of it and its repo
func (hc *Client) GetRelease(name, repo string) *Release {
	for _, release := range hc.Releases.Entries {
		if release.Name == name && release.Repo == repo {
			return release
		}
	}
	return nil
}

func (hc *Client) manageEntries(instance interface{}, k8sclient client.Client, g types.HTTPClientInterface, helmClient kube.Client) error {
	var releaseObj *helmv1alpha1.Release
	repoObj, ok := instance.(*helmv1alpha1.Repo)
	settings := hc.GetEnvSettings()

	if ok {
		hc.Repos.Entries = append(hc.Repos.Entries, NewHelmRepo(repoObj, settings, k8sclient, g, helmClient))
	}

	if releaseObj, ok = instance.(*helmv1alpha1.Release); ok {
		hc.Releases.Entries = append(hc.Releases.Entries, NewHelmRelease(releaseObj, settings, k8sclient, g, helmClient))
	}

	return nil
}
