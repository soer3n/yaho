package helm

import "github.com/prometheus/common/log"

// HandleFinalizer represents func for managing action related to a change of a finalizer related to a release or repo resource
func HandleFinalizer(hc *Client, instance interface{}) (bool, error) {
	if len(hc.Repos.Entries) > 0 {
		return true, nil
	}

	if len(hc.Releases.Entries) > 0 {
		if err := removeRelease(hc.Releases.Entries[0]); err != nil {
			return true, err
		}

		return true, nil
	}

	return false, nil
}

func removeRelease(helmRelease *Release) error {
	if _, err := helmRelease.getRelease(); err != nil {
		log.Debugf("%v", err)
		return nil
	}

	if err := helmRelease.Remove(); err != nil {
		return err
	}

	return nil
}
