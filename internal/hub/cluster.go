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
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	klog.V(0).Infof("new cluster: %v", string(cluster.WatchNamespace))

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
		klog.V(0).Infof("[%s] state for cluster %s is not ok. Error: %s", time.Now(), c.name, err.Error())
		return err
	}

	klog.V(0).Infof("[%s] state for cluster %s is ok.", time.Now(), c.name)

	for _, r := range currentReleaseList.Items {
		hc := &chart.Chart{}

		if err := syncChartResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
			klog.V(0).Infof("error on release in health check loop for cluster %s. Error: %s ...", c.name, err.Error())
			continue
		}

		//4. get repo index for each dependency in local cluster
		if err := syncChartDependencyResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
			klog.V(0).Infof("error on release in health check loop for cluster %s. Error: %s ...", c.name, err.Error())
			continue
		}

		//7. try to sync configmaps (index, default values, templates, crds) to remote cluster
		klog.V(0).Infof("event from cluster %s, release %s, chart %s, version %s\n", c.name, r.ObjectMeta.Name, r.Spec.Chart, r.Spec.Version)
	}

	return nil
}

func (c *Cluster) Start(tickerCtx context.Context, d time.Duration) {

	klog.V(0).Infof("initiate watcher for cluster %s...", c.name)
	releaseList := &yahov1alpha2.ReleaseList{}
	w, err := c.remoteClient.Watch(tickerCtx, releaseList, &client.ListOptions{})

	if err != nil {
		klog.V(0).Infof("%s.", err.Error())
	}

	go func(ctx context.Context) {
		klog.V(0).Infof("watcher for cluster %s started ...", c.name)
		defer klog.V(0).Infof("watcher for cluster %s stopped ...", c.name)

		for {
			select {
			case <-ctx.Done():
				klog.V(0).Infof("closing watcher channel for cluster %s ...", c.name)
				close(c.channel)
				return
			case res := <-w.ResultChan():
				r := res.Object.(*yahov1alpha2.Release)
				hc := &chart.Chart{}

				if err := syncChartResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
					klog.V(0).Infof("error on release in event loop for cluster %s. Error: %s ...", c.name, err.Error())
					continue
				}

				//4. get repo index for each dependency in local cluster
				if err := syncChartDependencyResources(c.name, r.Spec.Chart, r.Spec.Repo, r.Spec.Version, c.WatchNamespace, hc, c.localClient, c.remoteClient, c.scheme, c.logger); err != nil {
					klog.V(0).Infof("error on release in event loop for cluster %s. Error: %s ...", c.name, err.Error())
					continue
				}

				//7. try to sync configmaps (index, default values, templates, crds) to remote cluster
				klog.V(0).Infof("event %s from cluster %s, release %s, chart %s, version %s\n", res.Type, c.name, r.ObjectMeta.Name, r.Spec.Chart, r.Spec.Version)
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
		klog.V(0).Infof("error on release event for cluster %s. Error: %s ...", cluster, err.Error())
		return err
	}

	if len(ycList.Items) < 1 {
		klog.V(0).Infof("need to create chart %s/%s for cluster %s. Error: No items found by selectors ...", repository, chartname, cluster)
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

		if err := localClient.Create(context.TODO(), new, &client.CreateOptions{}); err != nil {
			klog.V(0).Infof("error on chart parsing for cluster %s. Error: %s ...", cluster, err.Error())
			return err
		}

		// return fmt.Errorf("need new init for chart %s/%s in cluster %s", repository, chartname, cluster)
		klog.V(0).Infof("need new init for chart %s/%s in cluster %s. Waiting for initial status", repository, chartname, cluster)
		if !utils.WatchForSubResourceSync(new, schema.GroupVersionResource{
			Group:    new.TypeMeta.GroupVersionKind().Group,
			Version:  new.TypeMeta.GroupVersionKind().Version,
			Resource: new.TypeMeta.GroupVersionKind().Kind,
		}, "", watch.Modified) {
			return fmt.Errorf("waiting for initial status of chart %s/%s failed in cluster %s", repository, chartname, cluster)
		}
	}

	yc := ycList.Items[0]

	klog.V(0).Infof("got initial status for chart %s/%s in cluster %s. Status: %v", repository, chartname, cluster, yc.Status)

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

		klog.V(0).Infof("chart status update needed for requested version %s/%s-%s for cluster %s.", repository, chartname, version, cluster)
		if err := localClient.Status().Update(context.TODO(), &yc); err != nil {
			klog.V(0).Infof("error on chart status update for requested version %s/%s-%s for cluster %s. Error: %s ...", repository, chartname, version, cluster, err.Error())
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
		klog.V(0).Infof("error on loading chart %s/%s-%s for cluster %s. Error: %s ...", repository, chartname, version, cluster, err.Error())
		return err
	}

	if len(hc.Templates) < 1 {
		return fmt.Errorf("no files could be parsed from configmap for chart %s/%s version %s for cluster %s", repository, chartname, version, cluster)
	}

	// klog.V(0).Infof("configmap changed... new: %v \n current: %v \n", configmap.BinaryData, current.BinaryData)
	// klog.V(0).Infof("manage subresources. cluster: %s, repository: %s, chart: %v", cluster, repository, hc)
	klog.V(0).Infof("manage subresources. cluster: %s, repository: %s, chart: %v, templates: %s, values: %s", cluster, repository, hc.Name(), len(hc.Templates), len(hc.Values))
	if err := yahochart.ManageSubResources(hc, cv, repository, namespace, localClient, remoteClient, scheme, logger); err != nil {
		klog.V(0).Infof("error on loading configmaps related to chart %s/%s-%s for cluster %s. Error: %s ...", repository, chartname, version, cluster, err.Error())
		return err
	}

	klog.V(0).Infof("loading dependencies. cluster: %s, repository: %s, chart: %v, dependencies: %v", cluster, repository, hc.Name(), hc.Metadata.Dependencies)
	if err := yahochart.LoadDependencies(hc, namespace, utils.GetEnvSettings(map[string]string{}), scheme, logger, localClient); err != nil {
		klog.V(0).Infof("error on loading dependencies. cluster: %s, repository: %s, chart: %v, dependencies: %v; error: %s", cluster, repository, hc.Name(), hc.Metadata.Dependencies, err.Error())
		return err
	}

	for _, dep := range hc.Dependencies() {
		klog.V(0).Infof("manage subresources for dependency chart %s. cluster: %s, repository: %s, chart: %v, templates: %s, values: %s", dep.Name(), cluster, repository, hc.Name(), len(hc.Templates), len(hc.Values))
		if err := yahochart.ManageSubResources(hc, cv, repository, namespace, localClient, remoteClient, scheme, logger); err != nil {
			klog.V(0).Infof("error on loading configmaps related to dependecy chart %s for cluster %s. chart: %s/%s-%s Error: %s ...", dep.Name(), repository, chartname, version, cluster, err.Error())
			return err
		}
	}

	return nil
}

func (c *Cluster) Stop() error {
	c.cancelFunc()
	return nil
}
