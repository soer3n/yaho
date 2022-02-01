package utils

import (
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	types "github.com/soer3n/yaho/apis/helm/v1alpha1"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Contains represents func for checking if a string is in a list of strings
func Contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// GetLabelsByInstance represents func for parsing labels by k8s objectMeta and env map
func GetLabelsByInstance(metaObj metav1.ObjectMeta, env map[string]string) (string, string) {
	var repoPath, repoCache string

	repoPath = filepath.Dir(env["RepositoryConfig"])
	repoCache = env["RepositoryCache"]

	repoLabel, repoLabelOk := metaObj.Labels["repo"]
	repoGroupLabel, repoGroupLabelOk := metaObj.Labels["repoGroup"]

	if repoLabelOk {
		if repoGroupLabelOk {
			repoPath = repoPath + "/" + metaObj.Namespace + "/" + repoGroupLabel
			repoCache = repoCache + "/" + metaObj.Namespace + "/" + repoGroupLabel
		} else {
			repoPath = repoPath + "/" + metaObj.Namespace + "/" + repoLabel
			repoCache = repoCache + "/" + metaObj.Namespace + "/" + repoLabel
		}
	}

	return repoPath + "/repositories.yaml", repoCache
}

// GetChartVersion represents func for returning a struct for a version of a chart
func GetChartVersion(version string, chart *types.Chart) *types.ChartVersion {
	versionObj := &types.ChartVersion{}
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

// ConvertChartVersions represents func for converting chart version from internal to official helm project struct
func ConvertChartVersions(chart *types.Chart) []*repo.ChartVersion {
	var convertedVersions []*repo.ChartVersion

	for _, item := range chart.Spec.Versions {
		value := &repo.ChartVersion{
			Metadata: &helmchart.Metadata{
				Name:         chart.Name,
				Home:         chart.Spec.Home,
				Sources:      chart.Spec.Sources,
				Version:      item.Name,
				Description:  chart.Spec.Description,
				Dependencies: convertDependencies(item),
				Keywords:     chart.Spec.Keywords,
				Maintainers:  chart.Spec.Maintainers,
				Icon:         chart.Spec.Icon,
				APIVersion:   chart.Spec.APIVersion,
				Condition:    chart.Spec.Condition,
				Tags:         chart.Spec.Tags,
				AppVersion:   chart.Spec.AppVersion,
				Deprecated:   chart.Spec.Deprecated,
				Annotations:  chart.Spec.Annotations,
				KubeVersion:  chart.Spec.KubeVersion,
				Type:         chart.Spec.Type,
			},
			URLs: []string{item.URL},
		}

		convertedVersions = append(convertedVersions, value)
	}

	return convertedVersions
}

func convertDependencies(version types.ChartVersion) []*helmchart.Dependency {
	deps := []*helmchart.Dependency{}

	for _, dep := range version.Dependencies {
		deps = append(deps, &helmchart.Dependency{
			Name:       dep.Name,
			Version:    dep.Version,
			Repository: dep.Repo,
		})
	}

	return deps
}
