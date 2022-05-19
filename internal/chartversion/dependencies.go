package chartversion

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"encoding/json"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (chartVersion *ChartVersion) addDependencies() error {

	repoSelector := make(map[string]string)
	var group *string

	chartVersion.logger.Info("set selector")
	if _, ok := chartVersion.owner.ObjectMeta.Labels["repoGroup"]; ok {
		if len(chartVersion.owner.ObjectMeta.Labels["repoGroup"]) > 1 {
			repoSelector["repoGroup"] = chartVersion.owner.ObjectMeta.Labels["repoGroup"]
			v := chartVersion.owner.ObjectMeta.Labels["repoGroup"]
			group = &v
		}
	}

	repoSelector["repo"] = chartVersion.repo.Name

	chartVersion.logger.Info("create dependencies")
	chartVersion.deps = chartVersion.createDependenciesList(group)

	if err := chartVersion.loadDependencies(repoSelector); err != nil {
		return err
	}

	return nil
}

func (chartVersion *ChartVersion) createDependenciesList(group *string) []*helmv1alpha1.ChartDep {
	deps := make([]*helmv1alpha1.ChartDep, 0)

	if chartVersion.Obj == nil {
		return deps
	}

	for _, dep := range chartVersion.Version.Dependencies {

		repository, err := chartVersion.getRepoName(dep, group)

		if err != nil {
			chartVersion.logger.Info(err.Error())
		}

		for _, d := range chartVersion.Obj.Dependencies() {
			if err := chartVersion.updateIndexIfNeeded(d, dep); err != nil {
				chartVersion.logger.Info(err.Error())
			}
		}

		deps = append(deps, &helmv1alpha1.ChartDep{
			Name:      dep.Name,
			Version:   dep.Version,
			Repo:      repository,
			Condition: dep.Condition,
		})
	}

	return deps
}

func (chartVersion *ChartVersion) updateIndexIfNeeded(d *chart.Chart, dep *chart.Dependency) error {

	if d.Metadata.Name == dep.Name {
		ix, err := utils.LoadChartIndex(chartVersion.Version.Name, chartVersion.owner.Spec.Repository, chartVersion.namespace, chartVersion.k8sClient)

		if err != nil {
			return err
		}

		x := *ix

		for k, e := range *ix {
			if e.Metadata.Version == chartVersion.Version.Version {
				if err := chartVersion.updateIndexVersion(d, dep, e, x, k); err != nil {
					return err
				}
			}
		}

		return nil
	}

	return nil
}

func (chartVersion *ChartVersion) updateIndexVersion(d *chart.Chart, dep *chart.Dependency, e *repo.ChartVersion, x repo.ChartVersions, k int) error {

	for sk := range e.Metadata.Dependencies {
		if d.Metadata.Version != dep.Version {
			chartVersion.logger.Info("update index", "old", dep.Version, "new", d.Metadata.Version)
			dep.Version = d.Metadata.Version
			x[k].Metadata.Dependencies[sk].Version = d.Metadata.Version
			b, err := json.Marshal(x)

			if err != nil {
				return err
			}

			m := &v1.ConfigMap{}
			if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{
				Namespace: chartVersion.namespace,
				Name:      "helm-" + chartVersion.owner.Spec.Repository + "-" + chartVersion.owner.Spec.Name + "-index",
			}, m); err != nil {
				return err
			}

			m.BinaryData = map[string][]byte{
				"versions": b,
			}

			if err := chartVersion.k8sClient.Update(context.Background(), m); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func (chartVersion *ChartVersion) getRepoName(dep *chart.Dependency, group *string) (string, error) {

	repository := chartVersion.repo.Spec.Name

	// if chartVersion.Obj != nil {

	// TODO: more logic needed for handling unmanaged charts !!!
	if chartVersion.repo.Spec.URL != dep.Repository {

		ls := labels.Set{}

		if group != nil {
			// filter repositories by group selector if set
			ls = labels.Merge(ls, labels.Set{"repoGroup": *group})
		}

		repoList := &helmv1alpha1.RepositoryList{}

		if err := chartVersion.k8sClient.List(context.Background(), repoList, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(ls),
		}); err != nil {
			return repository, err
		}

		if len(repoList.Items) == 0 {
			return repository, nil
		}

		for _, r := range repoList.Items {
			if r.Spec.URL == dep.Repository {
				return r.Name, nil

			}
		}
	}
	// }

	return repository, nil
}

func (chartVersion *ChartVersion) loadDependencies(selectors map[string]string) error {
	var chartList helmv1alpha1.ChartList
	var err error

	opts := &client.ListOptions{
		LabelSelector: labels.NewSelector(),
		// Namespace:     chartVersion.owner.Namespace,
	}

	for k, selector := range selectors {
		r, _ := labels.NewRequirement(k, selection.Equals, []string{selector})
		opts.LabelSelector = opts.LabelSelector.Add(*r)
	}

	if err = chartVersion.k8sClient.List(context.Background(), &chartList, opts); err != nil {
		return err
	}

	options := &action.ChartPathOptions{}

	for _, item := range chartList.Items {
		for _, dep := range chartVersion.deps {
			if item.Spec.Name == dep.Name {
				options.RepoURL = dep.Repo
				options.Version = dep.Version
				var valueObj chartutil.Values

				depCondition := true
				conditional := strings.Split(dep.Condition, ".")

				if len(conditional) == 0 || len(conditional) > 2 {
					chartVersion.logger.Error(err, "failed to parse conditional for subchart", "name", chartVersion.Version.Name, "dependency", dep.Name)
					continue
				}

				// parse sub values for dependency
				subChartCondition, _ := chartVersion.Obj.Values[conditional[0]].(map[string]interface{})

				// getting subchart default value configmap
				subVals := chartVersion.getDefaultValuesFromConfigMap(dep.Name, dep.Version)

				// parse conditional to boolean
				if subChartCondition != nil {
					keyAsString := string(fmt.Sprint(subChartCondition[conditional[1]]))
					depCondition, _ = strconv.ParseBool(keyAsString)
				}

				// check conditional
				if depCondition {

					ix, err := utils.LoadChartIndex(dep.Name, dep.Repo, chartVersion.namespace, chartVersion.k8sClient)

					if err != nil {
						return err
					}

					obj := item.DeepCopy()
					subChart, err := New(dep.Version, chartVersion.namespace, obj, subVals, *ix, chartVersion.scheme, chartVersion.logger, chartVersion.k8sClient, chartVersion.getter)

					if err != nil {
						chartVersion.logger.Info("could not load subchart", "child", item.Spec.Name)
						return err
					}

					if subChart.Obj == nil {
						return errors.NewBadRequest("could not load subchart " + item.Spec.Name)
					}

					if valueObj, err = chartutil.ToRenderValues(subChart.Obj, subVals, chartutil.ReleaseOptions{}, nil); err != nil {
						return err
					}

					// get values as interface{}
					valueMap := valueObj.AsMap()["Values"]
					// cast to struct
					castedMap, _ := valueMap.(chartutil.Values)
					subChart.Obj.Values = castedMap
					chartVersion.Obj.AddDependency(subChart.Obj)
				}
			}
		}
	}

	return nil
}
