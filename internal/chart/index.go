package chart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
)

func GetChartIndexConfigMap(chartName, repository, namespace string, k8sclient client.WithWatch) (*v1.ConfigMap, error) {
	indexMap := &v1.ConfigMap{}

	//TODO: get it by label
	if err := k8sclient.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      "helm-" + repository + "-" + chartName + "-index",
	}, indexMap); err != nil {
		return nil, err
	}

	return indexMap, nil
}

func GetChartVersionFromIndexConfigmap(version string, indexMap *v1.ConfigMap) (*repo.ChartVersion, error) {

	var index repo.ChartVersions

	rawData, ok := indexMap.BinaryData["versions"]

	if !ok {
		return nil, errors.New("could not load versions from index")
	}

	if err := json.Unmarshal(rawData, &index); err != nil {
		return nil, err
	}

	currentVersion, _ := getParsedVersion(version, index)

	for _, v := range index {
		if v.Version == currentVersion {
			return v, nil
		}
	}

	return nil, fmt.Errorf("could not find version for %s from loaded index for chart %s", version, index[0].Metadata.Name)
}

func UpdateChartVersionsIndexConfigmap(cv *repo.ChartVersion, indexMap *v1.ConfigMap, repository, namespace string, k8sclient client.WithWatch, logger logr.Logger) error {

	var index repo.ChartVersions
	var err error
	data := map[string][]byte{}
	cm := &v1.ConfigMap{}
	repoChartVersions := &repo.ChartVersions{}
	versionIsValid := false

	// validate if there is any data in given index configmap
	if indexMap == nil || indexMap.BinaryData == nil {
		return fmt.Errorf("configmap is empty for chart %s/%s with version %s", repository, cv.Name, cv.Version)
	}
	rawData, ok := indexMap.BinaryData["versions"]

	if !ok {
		return errors.New("could not load versions from index. key not set in map")
	}

	// transform raw data into struct
	if err := json.Unmarshal(rawData, &index); err != nil {
		return err
	}

	// check if requested versions is in index
	for _, v := range index {
		if v.Version == cv.Version {
			// if requested version is found in input index we can return without any update
			logger.V(2).Info("version is valid", "chart", cv.Name, "repository", repository, "version", cv.Version)
			versionIsValid = true
		}
	}

	// return if the requested version was not found in given index struct
	if !versionIsValid {
		return fmt.Errorf("version %s for chart %s/%s is not valid", cv.Version, repository, cv.Name)
	}

	// TODO: should the chartversion not already present in index?
	index = append(index, cv)

	// transform index into binary data
	list, err := json.Marshal(index)

	if err != nil {
		return err
	}

	// get current remote index configmap
	if err := k8sclient.Get(context.TODO(), types.NamespacedName{
		Namespace: namespace,
		Name:      "helm-" + repository + "-" + cv.Name + "-index",
	}, cm); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.V(2).Info("index configmap not found. going to create it.", "chart", cv.Name, "repository", repository, "namespace", namespace)
			// we need to create the index configmap
			cm = &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-" + repository + "-" + cv.Name + "-index",
					Namespace: namespace,
					Labels: map[string]string{
						configMapRepoLabelKey: repository,
						configMapLabelKey:     cv.Name,
						configMapLabelType:    "index",
					},
				},
				BinaryData: map[string][]byte{},
			}

			data["versions"] = list

			cm.BinaryData = data
			// we do not set controller reference to a repository in remote cluster currently
			/*
				if err := controllerutil.SetControllerReference(instance, cm, scheme); err != nil {
					hr.logger.Error(err, "failed to set owner ref for chart", "chart", chart)
				}
			*/
			if err := k8sclient.Create(context.TODO(), cm); err != nil {
				if k8serrors.IsAlreadyExists(err) {
					logger.Info("chart configmap already exists", "chart", cv.Name, "name", cm.ObjectMeta.Name)
					if err := k8sclient.Update(context.TODO(), cm); err != nil {
						logger.Info("could not update repository index chart configmap", "chart", cv.Name, "name", cm.ObjectMeta.Name)
						return err
					}
					logger.Info("chart configmap of repository index configmap updated", "chart", cv.Name, "name", cm.ObjectMeta.Name)
					return nil
				}
				logger.Error(err, "not able to create index configmap", "chart", cv.Name)
				return err
			}
			logger.Info("chart configmap of repository index configmap created", "chart", cv.Name, "name", cm.ObjectMeta.Name)
			return nil
		} else {
			return err
		}
	}

	// transform list into map[string][]string
	data = cm.BinaryData
	repoChartVersionsSlice := []*repo.ChartVersion{}

	if len(data["versions"]) > 0 {
		// transform list into struct
		if err := json.Unmarshal(data["versions"], repoChartVersions); err != nil {
			return err
		}
		// readd current versions
		for _, rcv := range *repoChartVersions {
			repoChartVersionsSlice = append(repoChartVersionsSlice, rcv)
			if rcv.Version == cv.Version {
				// if requested version is in requested index found we can return without any update
				return nil
			}
		}
	}

	// add requested versions
	repoChartVersionsSlice = append(repoChartVersionsSlice, cv)
	repoChartVersions = (*repo.ChartVersions)(&repoChartVersionsSlice)

	// unmarshal data for update
	rawData, err = json.Marshal(repoChartVersions)

	if err != nil {
		return err
	}

	cm.BinaryData["versions"] = rawData

	// update configmap
	if err := k8sclient.Update(context.TODO(), cm); err != nil {
		return err
	}
	return nil

}

func (c *Chart) getChartURL(version, repository string) (string, error) {

	repoObj := &yahov1alpha2.Repository{}
	c.logger.V(2).Info("parsing chart url", "chart", c.Name, "repository", repository, "version", version)
	if err := c.kubernetes.client.Get(context.Background(), types.NamespacedName{Name: repository}, repoObj); err != nil {
		return "", err
	}

	for _, e := range c.helm.index {
		if e.Version == version {
			c.logger.V(2).Info("found valid version", "chart", c.Name, "repository", repository, "version", version)
			// use first url because it should be set in each case
			chartURL, err := repo.ResolveReferenceURL(repoObj.Spec.Source.URL, e.URLs[0])

			if err != nil {
				return "", err
			}

			return chartURL, nil
		}
	}

	return "", errors.New("could not set chartversion url")
}

func getParsedVersion(version string, index repo.ChartVersions) (string, error) {

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
