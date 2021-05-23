package v1alpha1

import "github.com/Masterminds/semver/v3"

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
