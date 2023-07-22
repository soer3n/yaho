package hub

import (
	"context"
	"fmt"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	yahochart "github.com/soer3n/yaho/internal/chart"
	"github.com/soer3n/yaho/internal/utils"
)

const configMapLabelKey = "yaho.soer3n.dev/chart"
const configMapRepoLabelKey = "yaho.soer3n.dev/repo"
const configMapLabelType = "yaho.soer3n.dev/type"

func NewClusterBackend(name, namespace string, kubeconfig []byte, localClient client.WithWatch, defaults Defaults, scheme *runtime.Scheme, logger logr.Logger, cancelFunc context.CancelFunc) (*Cluster, error) {

	clusterClient, err := generateClusterClient(kubeconfig, scheme)

	if err != nil {
		return nil, err
	}

	cluster := &Cluster{
		name:           name,
		channel:        make(chan []byte),
		localClient:    localClient,
		remoteClient:   clusterClient,
		config:         kubeconfig,
		WatchNamespace: namespace,
		defaults:       defaults,
		scheme:         scheme,
		logger:         logger,
		cancelFunc:     cancelFunc,
	}

	logger.V(0).Info("new cluster", "name", name, "watchNamespace", string(cluster.WatchNamespace))

	return cluster, nil
}

func generateClusterClient(kubeconfig []byte, scheme *runtime.Scheme) (client.WithWatch, error) {

	var clientCfg clientcmd.ClientConfig

	clientCfg, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}

	var restCfg *rest.Config

	restCfg, err = clientCfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	c, err := client.NewWithWatch(restCfg, client.Options{Scheme: scheme})

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Cluster) IsActive() bool {
	return c.channel != nil
}

func (c *Cluster) GetName() string {
	return c.name
}

func (c *Cluster) GetChannel() chan []byte {
	return c.channel
}

func (c *Cluster) GetConfig() []byte {
	return c.config
}

func (c *Cluster) GetDefaults() Defaults {
	return c.defaults
}

func (c *Cluster) GetScheme() *runtime.Scheme {
	return c.scheme
}

func (c *Cluster) Update(defaults Defaults, kubeconfig []byte, scheme *runtime.Scheme) error {
	clusterClient, err := generateClusterClient(kubeconfig, scheme)

	if err != nil {
		return err
	}

	if !reflect.DeepEqual(c.remoteClient, clusterClient) {
		c.remoteClient = clusterClient
	}

	currentReleaseList := &yahov1alpha2.ReleaseList{}

	if err := c.remoteClient.List(context.TODO(), currentReleaseList, &client.ListOptions{}); err != nil {
		c.logger.V(0).Info("state for cluster is not ok.", "cluster", c.name, "error", err.Error())
		return err
	}

	c.logger.V(0).Info("state for cluster is ok.", "name", c.name)

	for _, r := range currentReleaseList.Items {
		hc := &chart.Chart{}

		if err := syncChartResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
			c.logger.V(0).Error(err, "error on release in health check loop for cluster...")
			continue
		}

		//4. get repo index for each dependency in local cluster
		if err := syncChartDependencyResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
			c.logger.V(0).Error(err, "error on release in health check loop for cluster...", "name", c.name)
			continue
		}

		//7. try to sync configmaps (index, default values, templates, crds) to remote cluster
		c.logger.V(0).Info("event from cluster.", "cluster", c.name, "release", r.ObjectMeta.Name, "chart", r.Spec.Chart, "version", r.Spec.Version)
	}

	return nil
}

func (c *Cluster) Start(tickerCtx context.Context, d time.Duration) {

	c.logger.V(0).Info("initiate watcher", "cluster", c.name)
	releaseList := &yahov1alpha2.ReleaseList{}
	w, err := c.remoteClient.Watch(tickerCtx, releaseList, &client.ListOptions{})

	if err != nil {
		c.logger.V(0).Error(err, "error on creating watcher for cluster", "name", c.name)
	}

	go func(ctx context.Context) {
		c.logger.V(0).Info("watcher for cluster started ...", "cluster", c.name)
		defer c.logger.V(0).Info("watcher for cluster stopped ...", "cluster", c.name)

		for {
			select {
			case <-ctx.Done():
				c.logger.V(0).Info("closing watcher channel for cluster ...", "cluster", c.name)
				close(c.channel)
				return
			case res := <-w.ResultChan():
				r := res.Object.(*yahov1alpha2.Release)
				hc := &chart.Chart{}
				c.logger.V(0).Info("event from cluster", "event_type", res.Type, "cluster", c.name, "release", r.ObjectMeta.Name, "chart", r.Spec.Chart, "version", r.Spec.Version)
				if err := syncChartResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
					c.logger.V(0).Error(err, "error on release in event loop for cluster")
					continue
				}

				//4. get repo index for each dependency in local cluster
				if err := syncChartDependencyResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
					c.logger.V(0).Error(err, "error on release in event loop for cluster")
					continue
				}
			}
		}
	}(tickerCtx)
}

func syncChartDependencyResources(cluster, chartname, repository, version, namespace string, hc *chart.Chart, localClient, remoteClient client.WithWatch, scheme *runtime.Scheme, logger logr.Logger) error {

	for _, dep := range hc.Metadata.Dependencies {

		//5. validate each dependency chart version
		//6a. create dependency chart if it is not present
		//6b. update dependency chart resource status if version is not yet parsed
		dhc := &chart.Chart{}

		repo, err := yahochart.GetRepositoryNameByUrl(dep.Repository, localClient)

		if err != nil {
			return err
		}

		if err := syncChartResources(cluster, dep.Name, repo, dep.Version, namespace, dhc, localClient, remoteClient, scheme, logger); err != nil {
			return err
		}

		if len(hc.Dependencies()) > 0 {
			if err := syncChartDependencyResources(cluster, chartname, repository, version, namespace, dhc, localClient, remoteClient, scheme, logger); err != nil {
				return err
			}
		}
	}

	return nil
}

func syncChartResources(cluster, chartname, repository, version, namespace string, hc *chart.Chart, localClient, remoteClient client.WithWatch, scheme *runtime.Scheme, logger logr.Logger) error {
	// 1. get repo index for requested chart in local cluster
	cm, err := yahochart.GetChartIndexConfigMap(chartname, repository, namespace, localClient)

	if err != nil {
		return err
	}
	// 2. validate requested chart version
	cv, err := yahochart.GetChartVersionFromIndexConfigmap(version, cm)

	if err != nil {
		return err
	}

	ycList := &yahov1alpha2.ChartList{}

	labelSetRepo, _ := labels.ConvertSelectorToLabelsMap(configMapRepoLabelKey + "=" + repository)
	labelSetChart, _ := labels.ConvertSelectorToLabelsMap(configMapLabelKey + "=" + chartname)
	ls := labels.Merge(labelSetRepo, labelSetChart)

	if err := localClient.List(context.TODO(), ycList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(ls),
	}); err != nil {
		logger.V(0).Error(err, "error on release event for cluster...", "cluster", "chart", chartname, "repository", repository)
		return err
	}

	if len(ycList.Items) < 1 {
		logger.V(0).Info("need to create chart for cluster. No items found by selectors...", "repository", repository, "chart", chartname, "cluster", cluster)

		//3a. create chart if it is not present
		new := &yahov1alpha2.Chart{
			ObjectMeta: metav1.ObjectMeta{
				Name: chartname + "-" + repository,
				Labels: map[string]string{
					configMapRepoLabelKey: repository,
					configMapLabelKey:     chartname,
				},
			},
			Spec: yahov1alpha2.ChartSpec{
				Name:       chartname,
				Repository: repository,
				CreateDeps: true,
			},
		}

		ownerRepository := &yahov1alpha2.Repository{}

		if err := localClient.Get(context.TODO(), types.NamespacedName{Name: repository}, ownerRepository, &client.GetOptions{}); err != nil {
			logger.V(0).Error(err, "error on getting owner repository for chart for cluster...", "cluster", cluster, "chart", chartname, "repository", repository)
			return err
		}

		if err := controllerutil.SetControllerReference(ownerRepository, new, scheme); err != nil {
			logger.Error(err, "failed to set owner ref for chart", "chart", chartname)
		}

		if err := localClient.Create(context.TODO(), new, &client.CreateOptions{}); err != nil {
			logger.V(0).Error(err, "error on chart parsing for cluster...", "cluster", cluster, "chart", chartname)
			return err
		}

		// return fmt.Errorf("need new init for chart %s/%s in cluster %s", repository, chartname, cluster)
		logger.V(0).Info("need new init for chart in cluster. Waiting for initial status", "repository", repository, "chart", chartname, "cluster", cluster)
		if !utils.WatchForSubResourceSync(new, schema.GroupVersionResource{
			Group:    new.TypeMeta.GroupVersionKind().Group,
			Version:  new.TypeMeta.GroupVersionKind().Version,
			Resource: new.TypeMeta.GroupVersionKind().Kind,
		}, "", watch.Modified) {
			return fmt.Errorf("waiting for initial status of chart %s/%s failed in cluster %s", repository, chartname, cluster)
		}
	}

	yc := ycList.Items[0]

	logger.V(0).Info("got initial status for chart in cluster.", "repository", repository, "chart", chartname, "cluster", cluster, "status", yc.Status)

	// 3b. update chart resource status if version is not yet parsed
	_, isPresent := yc.Status.ChartVersions[cv.Version]
	if !isPresent {
		if yc.Status.ChartVersions == nil {
			yc.Status.ChartVersions = make(map[string]yahov1alpha2.ChartVersion)
		}
		yc.Status.ChartVersions[cv.Version] = yahov1alpha2.ChartVersion{
			Loaded:    false,
			Specified: false,
		}

		logger.V(0).Info("chart status update needed for requested version for cluster.", "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
		if err := localClient.Status().Update(context.TODO(), &yc); err != nil {
			logger.V(0).Error(err, "error on chart status update for requested version for cluster.", "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
			return err
		}
	}

	options := &action.ChartPathOptions{
		Version:               cv.Version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}

	// now we know that everything is present, so that we can load the chart
	err = yahochart.LoadChartByResources(localClient, logger, hc, cv, chartname, repository, namespace, options, map[string]interface{}{})

	if err != nil {
		logger.V(0).Error(err, "error on loading chart for cluster...", "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
		return err
	}

	if len(hc.Templates) < 1 {
		return fmt.Errorf("no files could be parsed from configmap for chart %s/%s version %s for cluster %s", repository, chartname, version, cluster)
	}

	// klog.V(0).Infof("configmap changed... new: %v \n current: %v \n", configmap.BinaryData, current.BinaryData)
	// klog.V(0).Infof("manage subresources. cluster: %s, repository: %s, chart: %v", cluster, repository, hc)
	logger.V(0).Info("manage subresources for cluster", "cluster", cluster, "repository", repository, "chart", hc.Name(), "template_count", len(hc.Templates), "values_count", len(hc.Values))
	if err := yahochart.ManageSubResources(hc, cv, repository, namespace, localClient, remoteClient, false, scheme, logger); err != nil {
		logger.V(0).Error(err, "error on loading configmaps related to chart.", "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
		return err
	}

	logger.V(0).Info("loading dependencies for cluster.", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
	if err := yahochart.LoadDependencies(hc, namespace, utils.GetEnvSettings(map[string]string{}), scheme, logger, localClient); err != nil {
		logger.V(0).Error(err, "error on loading dependencies for cluster", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
		return err
	}

	for _, dep := range hc.Dependencies() {
		logger.V(0).Info("manage subresources for dependency chart", "dependency_chart", dep.Name(), "cluster", cluster, "repository", repository, "chart", hc.Name(), "template_count", len(hc.Templates), "value_count", len(hc.Values))
		if err := yahochart.ManageSubResources(hc, cv, repository, namespace, localClient, remoteClient, false, scheme, logger); err != nil {
			logger.V(0).Error(err, "error on loading configmaps related to dependecy chart...", "dependency_chart", dep.Name(), "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
			return err
		}
	}

	return nil
}

func (c *Cluster) Stop() error {
	c.cancelFunc()
	return nil
}
