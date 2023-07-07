package release

/*
func (hc *Release) getChart(chartName, watchNamespace string, index repo.ChartVersions, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) (*helmchart.Chart, error) {

		hc.logger.Info("fetching chart related to release resource")
		charts := &yahov1alpha2.ChartList{}
		labelSetRepo, _ := labels.ConvertSelectorToLabelsMap(configMapRepoLabelKey + "=" + hc.Repo)
		labelSetChart, _ := labels.ConvertSelectorToLabelsMap(configMapLabelKey + "=" + chartName)
		ls := labels.Merge(labelSetRepo, labelSetChart)

		hc.logger.Info("selector", "labelset", ls)

		opts := &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(ls),
		}

		if err := hc.K8sClient.List(context.Background(), charts, opts); err != nil {
			return nil, err
		}

		if len(charts.Items) == 0 {
			return nil, errors.NewBadRequest("chart not found")
		}

		chartObj := &charts.Items[0]

		c, err := chartversion.New(hc.Version, watchNamespace, chartObj.ObjectMeta.Name, chartObj.Spec.Name, chartObj.Spec.Repository, vals, index, hc.scheme, hc.logger, hc.K8sClient, hc.getter)

		if err != nil {
			return nil, err
		}

		if c.Obj == nil {
			return nil, errors.NewBadRequest("could not load chart " + chartName + " from repository " + hc.Repo)
		}

		if len(c.Obj.Files) < 1 {
			return nil, errors.NewBadRequest("no files detected in chart struct")
		}

		return c.Obj, nil
	}
*/
func (hc *Release) validateChartSpecs() error {

	if err := hc.Chart.Validate(); err != nil {
		return err
	}

	for _, d := range hc.Chart.Dependencies() {

		if err := d.Validate(); err != nil {
			return err
		}
	}

	return nil
}
