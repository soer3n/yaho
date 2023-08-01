package chart

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"github.com/soer3n/yaho/internal/chartversion"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func LoadDependencies(hc *chart.Chart, namespace string, settings *cli.EnvSettings, scheme *runtime.Scheme, logger logr.Logger, c client.WithWatch) error {
	// TODO: load chart by configmaps using label selector
	var err error

	options := &action.ChartPathOptions{}

	for _, dep := range hc.Metadata.Dependencies {
		options.RepoURL = dep.Repository
		options.Version = dep.Version
		var valueObj chartutil.Values
		index := &v1.ConfigMapList{}

		labelSetRepo, _ := labels.ConvertSelectorToLabelsMap(configMapLabelType + "=index")
		labelSetChart, _ := labels.ConvertSelectorToLabelsMap(configMapLabelKey + "=" + dep.Name)
		ls := labels.Merge(labelSetRepo, labelSetChart)

		logger.Info("selector", "labelset", ls)

		opts := &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(ls),
			Namespace:     namespace,
		}

		if err := c.List(context.TODO(), index, opts); err != nil {
			return err
		}

		repoName, ok := index.Items[0].ObjectMeta.Labels[configMapRepoLabelKey]

		if !ok {
			logger.V(2).Info("repo label not found for dependency", "type", "index", "chart", dep.Name)
		}

		depCondition := true
		conditional := strings.Split(dep.Condition, ".")

		if len(conditional) == 0 || len(conditional) > 2 {
			logger.Error(err, "failed to parse conditional for subchart", "name", hc.Name, "dependency", dep.Name)
			continue
		}

		// parse sub values for dependency
		subChartCondition, _ := hc.Values[conditional[0]].(map[string]interface{})

		if err != nil {
			return err
		}

		ix, err := utils.LoadChartIndex(dep.Name, repoName, namespace, c)
		logger.V(2).Info("loading chart index for dependency", "chart", hc.Name(), "dependency", dep.Name, "version", dep.Version, "index", ix.Len())

		if err != nil {
			return err
		}

		parsedVersion, _ := getParsedVersion(dep.Version, *ix)

		// getting subchart default value configmap
		subVals := chartversion.GetDefaultValuesFromConfigMap(dep.Name, repoName, parsedVersion, namespace, c, logger)

		// parse conditional to boolean
		if subChartCondition != nil {
			keyAsString := string(fmt.Sprint(subChartCondition[conditional[1]]))
			depCondition, _ = strconv.ParseBool(keyAsString)
		}

		// check conditional
		if depCondition {

			var cv *repo.ChartVersion
			logger.V(0).Info("parsing version for dependency chart", "chart", hc.Name(), "dependency", dep.Name, "version", dep.Version, "parsedVersion", parsedVersion, "index", ix.Len())

			for _, item := range *ix {
				if item.Version == parsedVersion {
					logger.V(2).Info("found parsed version for dependency chart in index", "chart", hc.Name(), "dependency", dep.Name, "version", parsedVersion, "index", ix.Len())
					cv = item
				}
			}

			dhc := &chart.Chart{}

			options := &action.ChartPathOptions{
				Version:               cv.Version,
				InsecureSkipTLSverify: false,
				Verify:                false,
			}

			logger.V(0).Info("loading source configmaps for dependency", "chart", hc.Name(), "dependency", dep.Name, "version", parsedVersion, "options", options, "chartversion", cv)
			if err := LoadChartByResources(c, logger, dhc, cv, dep.Name, repoName, namespace, options, subVals); err != nil {
				return err
			}

			if dhc.Files == nil {
				return errors.New("could not load subchart " + dep.Name)
			}

			//TODO: we need to also load the dependencies of current dependency chart
			if err := LoadDependencies(dhc, namespace, settings, scheme, logger, c); err != nil {
				return err
			}

			if valueObj, err = chartutil.ToRenderValues(dhc, subVals, chartutil.ReleaseOptions{}, chartutil.DefaultCapabilities); err != nil {
				return err
			}

			// get values as interface{}
			valueMap := valueObj.AsMap()["Values"]
			// cast to struct
			castedMap, _ := valueMap.(chartutil.Values)
			dhc.Values = castedMap
			hc.AddDependency(dhc)
		}
	}

	return nil
}

func GetRepositoryNameByUrl(url string, c client.WithWatch) (string, error) {
	var name string

	r := &yahov1alpha2.RepositoryList{}
	if err := c.List(context.TODO(), r, &client.ListOptions{}); err != nil {
		return name, err
	}
	found := false
	for _, repository := range r.Items {
		if repository.Spec.Source.URL == url {
			name = repository.ObjectMeta.Name
			found = true
			break
		}
	}

	if !found {
		return name, errors.New("repository not found")
	}

	return name, nil
}

// shoud only be called within manager controllers!
func (c *Chart) CreateOrUpdateSubCharts() error {

	c.logger.V(0).Info("create or update chart resources for dependencies")
	for k := range c.Status.ChartVersions {

		var cv *repo.ChartVersion
		for _, ix := range c.helm.index {
			if ix.Version == k {
				cv = ix
			}
		}

		if cv == nil {
			c.logger.V(0).Info("could not load chart version from current index, continue ...", "chart", c.Name, "version", k)
			continue
		}

		for _, dep := range cv.Metadata.Dependencies {
			repoName, err := GetRepositoryNameByUrl(dep.Repository, c.kubernetes.client)

			if err != nil {
				c.logger.Error(err, "error on getting repository by url", "chart", dep.Name)
				condition := metav1.Condition{
					Type:               "dependenciesSync",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             "chartUpdate",
					Message:            fmt.Sprintf("failed to get repository name for chart %s", dep.Name),
				}
				meta.SetStatusCondition(c.Status.Conditions, condition)
				return err
			}

			if err := c.createOrUpdateSubChart(dep, repoName); err != nil {
				c.logger.V(2).Info("error on managing subchart", "child", dep.Name, "error", err.Error())
				condition := metav1.Condition{
					Type:               "dependenciesSync",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Time{Time: time.Now()},
					Reason:             "chartUpdate",
					Message:            fmt.Sprintf("failed to create chart resource for %s/%s", repoName, dep.Name),
				}
				meta.SetStatusCondition(c.Status.Conditions, condition)
				return err
			}

		}
	}

	condition := metav1.Condition{
		Type:               "dependenciesSync",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "chartUpdate",
		Message:            "successful synced",
	}
	meta.SetStatusCondition(c.Status.Conditions, condition)

	return nil
}

func (c *Chart) createOrUpdateSubChart(dep *chart.Dependency, repository string) error {

	c.logger.V(0).Info("fetching dependency charts related to requested resource")

	charts := &yahov1alpha2.ChartList{}
	labelSetRepo, _ := labels.ConvertSelectorToLabelsMap(configMapRepoLabelKey + "=" + repository)
	labelSetChart, _ := labels.ConvertSelectorToLabelsMap(configMapLabelKey + "=" + dep.Name)
	ls := labels.Merge(labelSetRepo, labelSetChart)

	c.logger.V(2).Info("selector", "labelset", ls)

	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(ls),
	}

	if err := c.kubernetes.client.List(context.Background(), charts, opts); err != nil {
		return err
	}

	var group *string

	if len(charts.Items) == 0 {
		c.logger.V(2).Info("chart not found")

		obj := &yahov1alpha2.Chart{
			ObjectMeta: metav1.ObjectMeta{
				Name: dep.Name + "-" + repository,
			},
			Spec: yahov1alpha2.ChartSpec{
				Name:       dep.Name,
				Repository: repository,
				CreateDeps: true,
				Versions:   []string{dep.Version},
			},
		}

		if obj.ObjectMeta.Labels == nil {
			obj.ObjectMeta.Labels = map[string]string{}
		}

		// TODO: use client to get reousrce and check then labels
		repoObj := &yahov1alpha2.Repository{}
		if err := c.kubernetes.client.Get(context.Background(), types.NamespacedName{Name: repository}, repoObj); err != nil {
			return err
		}
		if v, ok := repoObj.ObjectMeta.Labels[configMapRepoGroupLabelKey]; ok {
			group = &v
		}

		if group != nil {
			obj.ObjectMeta.Labels[configMapRepoGroupLabelKey] = *group
		}

		obj.ObjectMeta.Labels[configMapRepoLabelKey] = repository
		obj.ObjectMeta.Labels[configMapLabelUnmanaged] = "true"

		// TODO: we also need to get repository object with k8s client
		if err := controllerutil.SetControllerReference(repoObj, obj, c.kubernetes.scheme); err != nil {
			return err
		}

		if err := c.kubernetes.client.Create(context.TODO(), obj); err != nil {
			return err
		}

		if !c.watchForSubResourceSync(obj) {
			return errors.New("subresource" + obj.ObjectMeta.Name + "not synced")

		}

		return nil
	}

	current := &charts.Items[0]
	// group = nil

	if utils.Contains(current.Spec.Versions, dep.Version) {
		return nil
	}

	current.Spec.Versions = append(current.Spec.Versions, dep.Version)

	if err := c.kubernetes.client.Update(context.TODO(), current); err != nil {
		return err
	}

	if !utils.WatchForSubResourceSync(current, schema.GroupVersionResource{
		Group:    current.TypeMeta.GroupVersionKind().Group,
		Version:  current.TypeMeta.GroupVersionKind().Version,
		Resource: current.TypeMeta.GroupVersionKind().Kind,
	}, "", watch.Modified) {
		return errors.New("subresource" + current.ObjectMeta.Name + "not synced")
	}

	return nil
}

func (c *Chart) watchForSubResourceSync(subResource *yahov1alpha2.Chart) bool {

	r := &yahov1alpha2.ChartList{
		Items: []yahov1alpha2.Chart{
			*subResource,
		},
	}

	watcher, err := c.kubernetes.client.Watch(context.Background(), r)

	if err != nil {
		c.logger.Info("cannot get watcher for subresource")
		return false
	}

	defer watcher.Stop()

	select {
	case res := <-watcher.ResultChan():
		//ch := res.Object.(*yahov1alpha2.Chart)

		if res.Type == watch.Modified {

			//synced := "synced"
			//if *ch.Status.Dependencies == synced && *ch.Status.Versions == synced {
			return true
			//}
		}
	case <-time.After(10 * time.Second):
		return false
	}

	return false
}
