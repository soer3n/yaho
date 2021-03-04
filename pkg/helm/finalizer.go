package helm

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HandleFinalizer(hc *HelmClient, instance interface{}) (bool, error) {

	var metaObj *metav1.ObjectMeta
	metaObj, ok := instance.(*metav1.ObjectMeta)

	if !ok {
		return false, nil
	}

	isInstanceMarkedToBeDeleted := metaObj.GetDeletionTimestamp() != nil
	if isInstanceMarkedToBeDeleted {
		if len(hc.Repos.Entries) > 0 {
			if err := removeRepo(hc); err != nil {
				return true, err
			}
		}

		if len(hc.Releases.Entries) > 0 {
			if err := removeRelease(hc.Releases.Entries[0]); err != nil {
				return true, err
			}
		}

	}
	return false, nil
}

func removeRepo(hc *HelmClient) error {

	helmRepo := hc.Repos.Entries[0]
	name := helmRepo.Name

	if err := hc.Repos.SetInstalledRepos(); err != nil {
		return err
	}

	if err := hc.Repos.RemoveByName(name); err != nil {
		return err
	}

	return nil
}

func removeRelease(helmRelease *HelmRelease) error {

	if err := helmRelease.Remove(); err != nil {
		return err
	}

	return nil
}
