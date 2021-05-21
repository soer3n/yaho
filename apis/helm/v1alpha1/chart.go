package v1alpha1

func (chart *Chart) GetChartVersion(version string) *ChartVersion {
	versionObj := &ChartVersion{}

	for _, item := range chart.Spec.Versions {
		if item.Name == version {
			return &item
		}
	}

	return versionObj
}
