package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmutils "github.com/soer3n/apps-operator/pkg/helm"
)

func HandleFinalizer(helmRepo *helmutils.HelmRepo, hc *helmutils.HelmClient, instance *metav1.ObjectMeta) (error) {

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		// Run finalization logic for memcachedFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		if err := removeRepo(helmRepo.Name, hc); err != nil {
			return err
		}
	}
	return nil
}

func removeRepo(name string, hc *helmutils.HelmClient) error {
		
		if err := hc.Repos.SetInstalledRepos(); err != nil {
			return err
		}

		if err := hc.Repos.RemoveByName(name); err != nil {
			return err
		}

		return nil
}

func removeRelease(helmRelease helmutils.HelmRelease) error {

	if err := helmRelease.Remove(); err != nil {
		return err
	}

	return nil
}
