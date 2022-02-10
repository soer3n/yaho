package helm

import (
	"github.com/prometheus/common/log"
)

// HandleFinalizer represents func for managing action related to a change of a finalizer related to a release or repo resource
func HandleFinalizer(instance interface{}) (bool, error) {

	_, ok := instance.(*Repo)

	if ok {
		return true, nil
	}

	release, ok := instance.(*Release)

	if ok {
		if err := removeRelease(release); err != nil {
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
