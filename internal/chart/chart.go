package chart

import (
	"sync"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/chartversion"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
)

// New represents initialization of internal chart struct
func New(instance *helmv1alpha1.Chart, settings *cli.EnvSettings, scheme *runtime.Scheme, logger logr.Logger, k8sclient client.Client, g utils.HTTPClientInterface, c kube.Client) *Chart {

	var err error

	chart := &Chart{}

	logger.Info("init chart")
	config, err := utils.InitActionConfig(settings, c)

	if err != nil {
		logger.Info("Error on getting action config for chart")
		return chart
	}

	logger.Info("init metadata")
	chart.setMetadata(instance, config, settings, logger, k8sclient, g)

	chart.logger.Info("load chart struct")
	ix, err := utils.LoadChartIndex(chart.Name, chart.Repo, instance.ObjectMeta.Namespace, k8sclient)

	if err != nil {
		chart.logger.Info(err.Error())
		return chart
	}

	chart.index = *ix

	chart.logger.Info("set versions")
	if err := chart.setVersions(instance, scheme); err != nil {
		chart.logger.Info(err.Error())
	}

	return chart
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

func (c *Chart) setVersions(instance *helmv1alpha1.Chart, scheme *runtime.Scheme) error {

	var chartVersions ChartVersions

	for _, version := range instance.Spec.Versions {
		c.logger.Info("init version struct", "version", version)
		obj, err := chartversion.New(version, instance, nil, c.index, scheme, c.logger, c.K8sClient, c.getter)

		if err != nil {
			c.logger.Info(err.Error(), "version", version)
			return err
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

func (c *Chart) setMetadata(instance *helmv1alpha1.Chart, config *action.Configuration, settings *cli.EnvSettings, logger logr.Logger, k8sclient client.Client, g utils.HTTPClientInterface) {
	c.Name = instance.ObjectMeta.Name
	c.Namespace = instance.ObjectMeta.Namespace
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
