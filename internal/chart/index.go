package chart

import (
	"context"
	"encoding/json"
	"errors"

	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Masterminds/semver/v3"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
)

func GetChartIndexConfigMap(chartName, repository, namespace string, k8sclient client.WithWatch) (*v1.ConfigMap, error) {
	indexMap := &v1.ConfigMap{}

	//TODO: get it by label
	if err := k8sclient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      "helm-" + repository + "-" + chartName + "-index",
	}, indexMap); err != nil {
		return indexMap, err
	}

	return indexMap, nil
}

func GetChartVersionFromIndexConfigmap(version string, indexMap *v1.ConfigMap) (*repo.ChartVersion, error) {

	var index repo.ChartVersions

	rawData, ok := indexMap.BinaryData["versions"]

	if !ok {
		return nil, errors.New("could not load version from index")
	}

	if err := json.Unmarshal(rawData, &index); err != nil {
		return nil, err
	}

	for _, v := range index {
		if v.Version == version {
			return v, nil
		}
	}

	return nil, errors.New("could not load version from index")
}

func (c *Chart) getChartURL(version, repository string) (string, error) {

	repoObj := &yahov1alpha2.Repository{}
	c.logger.Info("parsing chart url", "chart", c.Name, "repository", repository, "version", version)
	if err := c.kubernetes.client.Get(context.Background(), types.NamespacedName{Name: repository}, repoObj); err != nil {
		return "", err
	}

	for _, e := range c.helm.index {
		if e.Version == version {
			c.logger.Info("found valid version", "chart", c.Name, "repository", repository, "version", version)
			// use first url because it should be set in each case
			chartURL, err := repo.ResolveReferenceURL(repoObj.Spec.URL, e.URLs[0])

			if err != nil {
				return "", err
			}

			return chartURL, nil
		}
	}

	return "", errors.New("could not set chartversion url")
}

func (c *Chart) getParsedVersion(version string, index repo.ChartVersions) (string, error) {

	var constraint *semver.Constraints
	var v *semver.Version
	var err error

	current, _ := semver.NewVersion("0.0.0")

	if constraint, err = semver.NewConstraint(version); err != nil {
		return "", err
	}

	for _, e := range index {
		if v, err = semver.NewVersion(e.Version); err != nil {
			return "", err
		}

		if constraint.Check(v) && v.GreaterThan(current) {
			current = v
			continue
		}
	}

	return current.String(), nil
}
