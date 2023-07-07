package chartversion

/*
func (chartVersion *ChartVersion) getChart(chartName string, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) (*chart.Chart, error) {

	helmChart := &chart.Chart{}

	chartVersion.logger.Info("fetching chart related to release resource")
	// TODO: fetch configmaps instead of chart resource
	// charts := &yahov1alpha2.ChartList{}
	/*labelSetRepo, _ := labels.ConvertSelectorToLabelsMap(configMapRepoLabelKey + "=" + chartVersion.repo)
	labelSetChart, _ := labels.ConvertSelectorToLabelsMap(configMapLabelKey + "=" + chartName)
	ls := labels.Merge(labelSetRepo, labelSetChart)

	chartVersion.logger.Info("selector", "labelset", ls)

	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(ls),
	}
	/*
		if err := chartVersion.k8sClient.List(context.Background(), charts, opts); err != nil {
			return nil, err
		}

		if len(charts.Items) == 0 {
			return nil, k8serrors.NewBadRequest("chart not found")
		}

		chartObj := &charts.Items[0]

	if err := LoadChartByResources(helmChart, chartName, chartPathOptions, vals); err != nil {
		return nil, err
	}

	return helmChart, nil
}
*/
