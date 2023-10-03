package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"helm.sh/helm/v3/pkg/repo"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	kyaml "sigs.k8s.io/yaml"
)

const configMapLabelKey = "yaho.soer3n.dev/chart"
const configMapRepoLabelKey = "yaho.soer3n.dev/repo"
const configMapLabelType = "yaho.soer3n.dev/type"

func NewClusterBackend(name, namespace, agentName, agentNamespace string, secret *v1.Secret, deployAgent bool, localClient client.WithWatch, defaults Defaults, scheme *runtime.Scheme, logger logr.Logger, cancelFunc context.CancelFunc) (*Cluster, error) {

	clusterClient, err := generateClusterClient(secret.Data["host"], []byte("/"), secret.Data["caData"], secret.Data["token"], secret.Data["kubeconfig"], false, scheme)

	if err != nil {
		return nil, err
	}

	cluster := &Cluster{
		name: name,
		agent: clusterAgent{
			Name:      agentName,
			Namespace: agentNamespace,
			Deploy:    deployAgent,
		},
		channel:        make(chan []byte),
		localClient:    localClient,
		remoteClient:   clusterClient,
		config:         secret.Data["kubeconfig"],
		WatchNamespace: namespace,
		defaults:       defaults,
		scheme:         scheme,
		logger:         logger,
		cancelFunc:     cancelFunc,
	}

	logger.V(0).Info("new cluster", "name", name, "watchNamespace", string(cluster.WatchNamespace))

	return cluster, nil
}

func generateClusterClient(host, APIPath, caData, token, kubeconfig []byte, insecure bool, scheme *runtime.Scheme) (client.WithWatch, error) {

	var clientCfg clientcmd.ClientConfig

	clientCfg, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	if err != nil {
		return nil, err
	}

	var cfg *rest.Config

	cfg, err = clientCfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	/*
		cfg := &rest.Config{
			Host:        string(host),
			APIPath:     string(APIPath),
			BearerToken: string(token),
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: insecure,
				CAData:   caData,
			},
		}
	*/
	c, err := client.NewWithWatch(cfg, client.Options{Scheme: scheme})

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Cluster) IsActive() bool {
	return c.agentIsAvailable()
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

func (c *Cluster) Update(defaults Defaults, secret *v1.Secret, scheme *runtime.Scheme) error {
	//TODO: rethink client building
	clusterClient, err := generateClusterClient(secret.Data["host"], []byte("/"), secret.Data["caData"], secret.Data["token"], secret.Data["kubeconfig"], false, scheme)

	if err != nil {
		return err
	}

	if !reflect.DeepEqual(c.remoteClient, clusterClient) {
		c.remoteClient = clusterClient
	}

	if c.agent.Deploy && !c.agentIsAvailable() {
		if err := c.deployAgent(); err != nil {
			return err
		}
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

	if c.agent.Deploy && !c.agentIsAvailable() {
		if err := c.deployAgent(); err != nil {
			c.logger.Error(err, "error on deploying agent before starting watcher")
			return
		}
	}

	c.logger.V(0).Info("initiate watcher", "cluster", c.name)
	releaseList := &yahov1alpha2.ReleaseList{}
	w, err := c.remoteClient.Watch(tickerCtx, releaseList, &client.ListOptions{})

	if err != nil {
		c.logger.V(0).Error(err, "error on creating watcher for cluster", "name", c.name)
		initList := &yahov1alpha2.ReleaseList{}
		ctx := context.TODO()
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				c.logger.Error(ctx.Err(), "timeout on waiting for release resources", "cluster", c.name)
				return
			default:
				if err := c.remoteClient.List(tickerCtx, initList, &client.ListOptions{}); err == nil {
					c.logger.V(0).Info("got initial release resource list for cluster", "name", c.name)
					break
				}
				time.Sleep(1 * time.Second)
			}
		}

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
				c.logger.V(0).Info("finished sync loop", "cluster", c.name, "chart", r.Spec.Chart, "version", r.Spec.Version)
			}
		}
	}(tickerCtx)
}

func (c *Cluster) agentIsAvailable() bool {

	agent := &appsv1.Deployment{}
	err := c.remoteClient.Get(context.TODO(), types.NamespacedName{Name: c.agent.Name, Namespace: c.agent.Namespace}, agent, &client.GetOptions{})

	return err != nil
}

func (c *Cluster) deployAgent() error {

	var err error
	controllerManagerConfigmap := &v1.ConfigMap{}

	c.logger.Info("try to get remote agent config", "cluster", c.name, "name", c.agent.Name, "namespace", c.agent.Namespace)
	err = c.remoteClient.Get(context.TODO(), types.NamespacedName{Name: "yaho-agent-config", Namespace: c.agent.Namespace}, controllerManagerConfigmap, &client.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		c.logger.Error(err, "other error than not found on getting remote agent config", "cluster", c.name, "name", c.agent.Name, "namespace", c.agent.Namespace)
		return err
	}

	if errors.IsNotFound(err) {
		c.logger.Info("create remote agent config", "cluster", c.name, "name", c.agent.Name, "namespace", c.agent.Namespace)
		controllerManagerConfig := `
---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: ControllerManagerConfig
healthProbeBindAddress: ":8081"
metricsBindAddress: "127.0.0.1:8080"
webhookPort: 9443
leaderElection:
  leaderElect: true
  resourceName: bb07iekd.soer3n.dev
`

		controllerManagerConfigmap = &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "yaho-agent-config",
				Namespace: c.agent.Namespace,
			},
			Data: map[string]string{
				"controller_manager_config.yaml": controllerManagerConfig,
			},
		}

		if err := c.remoteClient.Create(context.TODO(), controllerManagerConfigmap, &client.CreateOptions{}); err != nil {
			return err
		}
	}

	c.logger.Info("read and install crds for remote agent", "cluster", c.name, "name", c.agent.Name, "namespace", c.agent.Namespace)
	fs, err := os.ReadDir("config/crd/bases/")

	if err != nil {
		return err
	}

	for _, file := range fs {
		f, err := os.ReadFile("config/crd/bases/" + file.Name())

		if err != nil {
			return err
		}

		crd := &apiextensionsv1.CustomResourceDefinition{}
		d := []byte("---")
		new := bytes.ReplaceAll(f, d, []byte(""))

		j, err := kyaml.YAMLToJSON(new)

		if err != nil {
			return err
		}

		if err := json.Unmarshal(j, crd); err != nil {
			return err
		}

		if crd.Spec.Names.Kind != "Hub" {
			if err := c.remoteClient.Get(context.TODO(), types.NamespacedName{Name: crd.Name}, crd, &client.GetOptions{}); err != nil {
				if errors.IsNotFound(err) {
					if err := c.remoteClient.Create(context.TODO(), crd, &client.CreateOptions{}); err != nil {
						return err
					}
				}
			}
		}
	}

	userId := int64(65532)
	allowPrivilegeEscalation := false

	localObjRef := v1.LocalObjectReference{
		Name: "yaho-agent-config",
	}

	volumeSource := v1.VolumeSource{
		ConfigMap: &v1.ConfigMapVolumeSource{
			LocalObjectReference: localObjRef,
		},
	}

	agent := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.agent.Name,
			Namespace: c.agent.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"yaho.soer3n.dev/agent": c.agent.Name},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"yaho.soer3n.dev/agent": c.agent.Name},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "yaho-agent",
					SecurityContext: &v1.PodSecurityContext{
						RunAsUser: &userId,
					},
					Volumes: []v1.Volume{
						{
							VolumeSource: volumeSource,
							Name:         "agent-config",
						},
					},
					Containers: []v1.Container{
						{
							Name:    "agent",
							Image:   "soer3n/yaho:0.0.3",
							Command: []string{"/manager", "agent", "run"},
							Args:    []string{"--leader-elect", "--config=controller_manager_config.yaml", "--health-probe-bind-address=:8081", "--metrics-bind-address=127.0.0.1:8080"},
							Env: []v1.EnvVar{
								{
									Name:  "WATCH_NAMESPACE",
									Value: c.agent.Namespace,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "agent-config",
									MountPath: "/controller_manager_config.yaml",
									SubPath:   "controller_manager_config.yaml",
								},
							},
							Resources: v1.ResourceRequirements{},
							SecurityContext: &v1.SecurityContext{
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
							ReadinessProbe: &v1.Probe{
								InitialDelaySeconds: int32(5),
								PeriodSeconds:       int32(10),
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/readyz",
										Port: intstr.FromInt(8081),
									},
								},
							},
							LivenessProbe: &v1.Probe{
								InitialDelaySeconds: int32(15),
								PeriodSeconds:       int32(20),
								ProbeHandler: v1.ProbeHandler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt(8081),
									},
								},
							},
						},
						{
							Name:      "kube-rbac-proxy",
							Image:     "gcr.io/kubebuilder/kube-rbac-proxy:v0.11.0",
							Args:      []string{"--secure-listen-address=0.0.0.0:8443", "--upstream=http://127.0.0.1:8080/", "--logtostderr=true", "--v=0"},
							Resources: v1.ResourceRequirements{},
							SecurityContext: &v1.SecurityContext{
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
						},
					},
				},
			},
		},
	}

	c.logger.Info("create and install remote agent deployment", "cluster", c.name, "name", c.agent.Name, "namespace", c.agent.Namespace)
	if err := c.remoteClient.Create(context.TODO(), agent, &client.CreateOptions{}); err != nil {
		return err
	}
	return nil
}

func (c *Cluster) deleteAgent() error {

	agent := &appsv1.Deployment{}

	if err := c.remoteClient.Get(context.TODO(), types.NamespacedName{Name: c.agent.Name, Namespace: c.agent.Namespace}, agent, &client.GetOptions{}); err != nil {
		return err
	}

	if err := c.remoteClient.Delete(context.TODO(), agent, &client.DeleteOptions{}); err != nil {
		return err
	}

	return nil
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

		if err := yahochart.LoadDependencies(dhc, namespace, utils.GetEnvSettings(map[string]string{}), scheme, logger, localClient); err != nil {
			logger.V(0).Error(err, "error on loading dependencies for cluster", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
			return err
		}

		if len(dhc.Dependencies()) > 0 {
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

	/*
		logger.V(0).Info("loading dependencies for cluster.", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
		if err := yahochart.LoadDependencies(hc, namespace, utils.GetEnvSettings(map[string]string{}), scheme, logger, localClient); err != nil {
			logger.V(0).Error(err, "error on loading dependencies for cluster", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
			return err
		}
		//TODO: we need to render also children of child charts
		for _, dep := range hc.Dependencies() {
			logger.V(0).Info("manage subresources for dependency chart", "dependency_chart", dep.Name(), "cluster", cluster, "repository", repository, "chart", hc.Name(), "template_count", len(hc.Templates), "value_count", len(hc.Values))
			if err := yahochart.ManageSubResources(hc, cv, repository, namespace, localClient, remoteClient, false, scheme, logger); err != nil {
				logger.V(0).Error(err, "error on loading configmaps related to dependecy chart...", "dependency_chart", dep.Name(), "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
				return err
			}
		}
	*/

	logger.V(0).Info("manage dependencies for cluster", "cluster", cluster, "repository", repository, "chart", hc.Name(), "template_count", len(hc.Templates), "values_count", len(hc.Values))
	if err := manageDependencyCharts(cluster, chartname, repository, version, namespace, hc, cv, localClient, remoteClient, scheme, logger); err != nil {
		logger.V(0).Error(err, "error on managing dependency charts related to chart.", "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
		return err
	}

	return nil
}

func manageDependencyCharts(cluster, chartname, repository, version, namespace string, hc *chart.Chart, cv *repo.ChartVersion, localClient, remoteClient client.WithWatch, scheme *runtime.Scheme, logger logr.Logger) error {

	logger.V(0).Info("loading dependencies for cluster.", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
	if err := yahochart.LoadDependencies(hc, namespace, utils.GetEnvSettings(map[string]string{}), scheme, logger, localClient); err != nil {
		logger.V(0).Error(err, "error on loading dependencies for cluster", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
		return err
	}

	for _, dep := range hc.Dependencies() {
		repoURL := ""
		for _, depStruct := range hc.Metadata.Dependencies {
			if depStruct.Name == dep.Name() {
				repoURL = depStruct.Repository
			}
		}

		if repoURL == "" {
			return fmt.Errorf("repository url not found")
		}

		repo, err := yahochart.GetRepositoryNameByUrl(repoURL, localClient)

		if err != nil {
			return err
		}

		if err := yahochart.ManageSubResources(hc, cv, repo, namespace, localClient, remoteClient, false, scheme, logger); err != nil {
			logger.V(0).Error(err, "error on loading configmaps related to dependecy chart...", "dependency_chart", dep.Name(), "repository", repository, "chart", chartname, "version", version, "cluster", cluster)
			return err
		}

		if err := yahochart.LoadDependencies(dep, namespace, utils.GetEnvSettings(map[string]string{}), scheme, logger, localClient); err != nil {
			logger.V(0).Error(err, "error on loading dependencies for cluster", "cluster", cluster, "repository", repository, "chart", hc.Name(), "dependencies", hc.Metadata.Dependencies)
			return err
		}

		// inner loop
		if err := manageDependencyCharts(cluster, dep.Name(), repo, dep.Metadata.Version, namespace, dep, cv, localClient, remoteClient, scheme, logger); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) Stop() error {
	c.cancelFunc()

	if !c.agent.Deploy {
		return nil
	}

	if err := c.deleteAgent(); err != nil {
		c.logger.Error(err, "error on deleting agent")
		return err
	}
	return nil
}
