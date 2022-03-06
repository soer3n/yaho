package utils

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func LoadChartIndex(chart, repository, namespace string, c client.Client) (*repo.ChartVersions, error) {

	var rawData []byte

	obj := &v1.ConfigMap{}
	var versions *repo.ChartVersions

	if err := c.Get(context.Background(), types.NamespacedName{
		Name:      "helm-" + repository + "-" + chart + "-index",
		Namespace: namespace,
	}, obj); err != nil {
		return nil, err
	}

	rawData = obj.BinaryData["versions"]

	if err := json.Unmarshal(rawData, &versions); err != nil {
		return nil, err
	}

	return versions, nil
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
func GetChartVersion(version string, chart *helmv1alpha1.Chart) *helmv1alpha1.ChartVersion {
	versionObj := &helmv1alpha1.ChartVersion{}
	var constraint *semver.Constraints
	var v *semver.Version
	var err error

	current, _ := semver.NewVersion("0.0.0")
	// currentIndex := 0

	if constraint, err = semver.NewConstraint(version); err != nil {
		return versionObj
	}

	for _, item := range chart.Spec.Versions {
		if v, err = semver.NewVersion(item); err != nil {
			continue
		}

		if constraint.Check(v) && v.GreaterThan(current) {
			current = v
			// currentIndex = ix
			continue
		}
	}

	// return &chart.Spec.Versions[currentIndex]
	return versionObj
}

// ConvertChartVersions represents func for converting chart version from internal to official helm project struct
func ConvertChartVersions(chart *helmv1alpha1.Chart) []*repo.ChartVersion {
	var convertedVersions []*repo.ChartVersion
	/*
		for _, item := range chart.Spec.Versions {
			value := &repo.ChartVersion{
				Metadata: &helmchart.Metadata{
					Name:         chart.Name,
					Home:         chart.Spec.Home,
					Sources:      chart.Spec.Sources,
					// Version:      item.Name,
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
	*/
	return convertedVersions
}

func convertDependencies(version helmv1alpha1.ChartVersion) []*helmchart.Dependency {
	deps := []*helmchart.Dependency{}

	for _, dep := range version.Dependencies {
		deps = append(deps, &helmchart.Dependency{
			Name:       dep.Name,
			Version:    dep.Version,
			Repository: dep.Repo,
			Condition:  dep.Condition,
		})
	}

	return deps
}

// RandomString return a string with random chars of length n
func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		n, _ := rand.Int(rand.Reader, (big.NewInt(30)))
		b[i] = letters[n.Uint64()]
	}
	return string(b)
}
