package utils

import (
	helmutils "github.com/soer3n/apps-operator/pkg/helm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HandleFinalizer(hc *helmutils.HelmClient, instance interface{}) error {

	var metaObj *metav1.ObjectMeta
	metaObj, ok := instance.(*metav1.ObjectMeta)

	if !ok {
		return nil
	}

	isInstanceMarkedToBeDeleted := metaObj.GetDeletionTimestamp() != nil
	if isInstanceMarkedToBeDeleted {
		if len(hc.Repos.Entries) > 0 {
			if err := removeRepo(hc); err != nil {
				return err
			}
		}

		if len(hc.Releases.Entries) > 0 {
			if err := removeRelease(hc.Releases.Entries[0]); err != nil {
				return err
			}
		}

	}
	return nil
}

func removeRepo(hc *helmutils.HelmClient) error {

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

func removeRelease(helmRelease *helmutils.HelmRelease) error {

	if err := helmRelease.Remove(); err != nil {
		return err
	}

	return nil
}
