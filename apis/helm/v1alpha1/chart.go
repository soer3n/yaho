package v1alpha1

import (
	"github.com/Masterminds/semver/v3"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

func (chart *Chart) GetChartVersion(version string) *ChartVersion {
	versionObj := &ChartVersion{}
	var constraint *semver.Constraints
	var v *semver.Version
	var err error

	if constraint, err = semver.NewConstraint(version); err != nil {
		return versionObj
	}

	for _, item := range chart.Spec.Versions {
		if v, err = semver.NewVersion(item.Name); err != nil {
			continue
		}

		if constraint.Check(v) {
			return &item
		}
	}

	return versionObj
}

func (chart *Chart) ConvertChartVersions() []*repo.ChartVersion {
	var convertedVersions []*repo.ChartVersion

	for _, item := range chart.Spec.Versions {
		value := &repo.ChartVersion{
			Metadata: &helmchart.Metadata{
				Name:        chart.Name,
				Home:        chart.Spec.Home,
				Sources:     chart.Spec.Sources,
				Version:     item.Name,
				Description: chart.Spec.Description,
				Keywords:    chart.Spec.Keywords,
				Maintainers: chart.Spec.Maintainers,
				Icon:        chart.Spec.Icon,
				APIVersion:  chart.Spec.APIVersion,
				Condition:   chart.Spec.Condition,
				Tags:        chart.Spec.Tags,
				AppVersion:  chart.Spec.AppVersion,
				Deprecated:  chart.Spec.Deprecated,
				Annotations: chart.Spec.Annotations,
				KubeVersion: chart.Spec.KubeVersion,
				Type:        chart.Spec.Type,
			},
			URLs: []string{item.URL},
		}

		convertedVersions = append(convertedVersions, value)
	}

	return convertedVersions
}
