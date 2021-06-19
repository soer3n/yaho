package helm

import "github.com/prometheus/common/log"

func HandleFinalizer(hc *HelmClient, instance interface{}) (bool, error) {

	if len(hc.Repos.Entries) > 0 {
		return true, nil
	}

	if len(hc.Releases.Entries) > 0 {
		if err := removeRelease(hc.Releases.Entries[0]); err != nil {
			return true, err
		}
	}
	return false, nil
}

func removeRepo(hc *HelmClient) error {

	helmRepo := hc.Repos.Entries[0]
	name := helmRepo.Name

	if err := hc.setInstalledRepos(); err != nil {
		return err
	}

	if err := hc.RemoveByName(name); err != nil {
		return err
	}

	return nil
}

func removeRelease(helmRelease *HelmRelease) error {

	if _, err := helmRelease.getRelease(); err != nil {
		log.Debugf("%v", err)
		return nil
	}

	if err := helmRelease.Remove(); err != nil {
		return err
	}

	return nil
}
