package release

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (hc *Release) addDependencies(chart *helmchart.Chart, deps []*helmv1alpha1.ChartDep, vals chartutil.Values, selectors map[string]string) error {
	var chartList helmv1alpha1.ChartList
	var err error

	selectorObj := client.MatchingLabels{}

	for k, selector := range selectors {
		selectorObj[k] = selector
	}

	if err = hc.K8sClient.List(context.Background(), &chartList, selectorObj, client.InNamespace(hc.Namespace.Name)); err != nil {
		return err
	}

	options := &action.ChartPathOptions{}

	for _, item := range chartList.Items {
		for _, dep := range deps {
			if item.Spec.Name == dep.Name {
				options.RepoURL = dep.Repo
				options.Version = dep.Version
				var valueObj chartutil.Values

				depCondition := true
				conditional := strings.Split(dep.Condition, ".")

				if len(conditional) == 0 || len(conditional) > 2 {
					hc.logger.Error(err, "failed to parse conditional for subchart", "name", hc.Name, "dependency", dep.Name)
					continue
				}

				// parse sub values for dependency
				subChartCondition, _ := vals[conditional[0]].(map[string]interface{})

				// getting subchart default value configmap
				subVals := hc.getDefaultValuesFromConfigMap("helm-default-" + dep.Name + "-" + dep.Version)

				// parse conditional to boolean
				if subChartCondition != nil {
					keyAsString := string(fmt.Sprint(subChartCondition[conditional[1]]))
					depCondition, _ = strconv.ParseBool(keyAsString)
				}

				// check conditional
				if depCondition {

					subChart, _ := hc.getChart(item.Spec.Name, options, subVals)

					if valueObj, err = chartutil.ToRenderValues(subChart, subVals, chartutil.ReleaseOptions{}, nil); err != nil {
						return err
					}

					// get values as interface{}
					valueMap := valueObj.AsMap()["Values"]
					// cast to struct
					castedMap, _ := valueMap.(chartutil.Values)
					subChart.Values = castedMap
					// hc.Chart.AddDependency(subChart)
					chart.AddDependency(subChart)
				}
			}
		}
	}

	return nil
}
