package release

// RemoveRelease represents func for managing action related to a change of a finalizer related to a release or repo resource
func (hr *Release) RemoveRelease() error {
	if _, err := hr.getRelease(); err != nil {
		return nil
	}

	if err := hr.Remove(); err != nil {
		return err
	}

	return nil
}
