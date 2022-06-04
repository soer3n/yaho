package chart

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/chartversion"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
)

// New represents initialization of internal chart struct
func New(instance *helmv1alpha1.Chart, namespace string, settings *cli.EnvSettings, scheme *runtime.Scheme, logger logr.Logger, k8sclient client.WithWatch, g utils.HTTPClientInterface, getter genericclioptions.RESTClientGetter, kubeconfig []byte) (*Chart, error) {

	var err error

	chart := &Chart{}

	logger.Info("init chart")
	config, err := utils.InitActionConfig(getter, kubeconfig, logger)

	if err != nil {
		logger.Info("Error on getting action config for chart")
		return nil, err
	}

	logger.Info("init metadata")
	chart.setMetadata(instance, namespace, config, settings, logger, k8sclient, g)

	chart.logger.Info("load chart struct")
	ix, err := utils.LoadChartIndex(chart.Name, chart.Repo, namespace, k8sclient)

	if err != nil {
		chart.logger.Info(err.Error())
		return nil, err
	}

	chart.index = *ix

	chart.logger.Info("set versions")
	if err := chart.setVersions(instance, namespace, scheme); err != nil {
		chart.logger.Info(err.Error())
		return nil, err
	}

	return chart, nil
}

func (c *Chart) Update(instance *helmv1alpha1.Chart) error {

	c.logger.Info("create or update configmaps for versions")
	if err := c.updateVersions(); err != nil {
		return err
	}

	return nil
}

func (c *Chart) CreateOrUpdateSubCharts() error {

	c.logger.Info("create or update chart resources for dependencies")
	for _, v := range c.Versions {
		if err := v.CreateOrUpdateSubCharts(); err != nil {
			c.logger.Info("error on managing subchart", "child", v.Version.Name, "error", err.Error())
			return err
		}
	}

	return nil
}

func (c *Chart) setVersions(instance *helmv1alpha1.Chart, namespace string, scheme *runtime.Scheme) error {

	var chartVersions ChartVersions

	for _, version := range instance.Spec.Versions {
		c.logger.Info("init version struct", "version", version)
		obj, err := chartversion.New(version, namespace, instance, nil, c.index, scheme, c.logger, c.K8sClient, c.getter)

		if err != nil {
			c.logger.Info(err.Error(), "version", version)
			return err
		}

		if c.Deprecated == nil {
			instance.Status.Deprecated = &obj.Version.Deprecated
		}

		if c.Type == nil {
			instance.Status.Type = &obj.Version.Type
		}

		if c.Tags == nil {
			instance.Status.Tags = &obj.Version.Tags
		}

		chartVersions = append(chartVersions, obj)
	}

	c.Versions = chartVersions
	return nil
}

func (c *Chart) updateVersions() error {

	for _, v := range c.Versions {
		c.logger.Info("prepare object", "version", v.Version)
		if err := v.Prepare(c.helmConfig); err != nil {
			return err
		}

		c.logger.Info("create or update configmaps", "version", v.Version)
		if err := v.ManageSubResources(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Chart) setMetadata(instance *helmv1alpha1.Chart, namespace string, config *action.Configuration, settings *cli.EnvSettings, logger logr.Logger, k8sclient client.WithWatch, g utils.HTTPClientInterface) {
	c.Name = instance.Spec.Name
	c.Namespace = namespace
	c.helmConfig = config
	c.Client = action.NewInstall(config)
	c.Settings = settings
	c.Repo = instance.Spec.Repository
	c.K8sClient = k8sclient
	// URL:       chartURL,
	c.getter = g
	c.logger = logger.WithValues("repo", instance.Spec.Repository, "chart", instance.Spec.Name)
	c.mu = &sync.Mutex{}
}
