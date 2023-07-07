package chartversion

/*
const configMapLabelKey = "yaho.soer3n.dev/chart"
const configMapRepoLabelKey = "yaho.soer3n.dev/repo"
const configMapRepoGroupLabelKey = "yaho.soer3n.dev/repoGroup"
const configMapLabelType = "yaho.soer3n.dev/type"
const configMapLabelSubName = "yaho.soer3n.dev/subname"

// TODO: what does this mean?
const configMapLabelUnmanaged = "yaho.soer3n.dev/unmanaged"

func New(version, namespace, chartResourceName, chartName, repoName string, vals chartutil.Values, index repo.ChartVersions, scheme *runtime.Scheme, logger logr.Logger, k8sclient client.WithWatch, g utils.HTTPClientInterface) (*ChartVersion, error) {

	obj := &ChartVersion{
		mu:        sync.Mutex{},
		wg:        sync.WaitGroup{},
		owner:     chartName,
		namespace: namespace,
		scheme:    scheme,
		k8sClient: k8sclient,
		logger:    logger,
		getter:    g,
	}

	parsedVersion, err := obj.getParsedVersion(version, index)

	if err != nil {
		obj.logger.Info("could not parse semver version", "version", version)
		return nil, err
	}

	for _, cv := range index {
		if cv.Version == parsedVersion {
			obj.Version = cv
			obj.Version.Version = parsedVersion
			break
		}
	}

	if obj.Version == nil {
		return obj, errors.New("chart version is not valid")
	}

	//TODO: no need to request repository resource
	// repo, err := obj.getControllerRepo()

	if err != nil {
		logger.Info(err.Error())
		return obj, err
	}

	obj.repo = repoName

	if err := obj.setChartURL(index); err != nil {
		return obj, err
	}

	options := &action.ChartPathOptions{
		Version:               version,
		InsecureSkipTLSverify: false,
		Verify:                false,
	}

	if vals == nil {
		vals = obj.getDefaultValuesFromConfigMap(chartName, parsedVersion)
	}

	c, err := obj.getChart(chartName, options, vals)

	if err != nil {
		obj.logger.Info(err.Error())
	}

	obj.Obj = c

	if err := obj.addDependencies(); err != nil {
		obj.logger.Info(err.Error())
	}

	return obj, nil
}

// this method should only be called within controllers of manager directory or moved to chart model!
func (chartVersion *ChartVersion) Prepare(config *action.Configuration) error {

	releaseClient := action.NewInstall(config)

	if chartVersion.Obj == nil {
		chartVersion.logger.Info("load chart obj")
		err := chartVersion.loadChartByURL(releaseClient)

		if err != nil {
			return err
		}
	}

	if err := chartVersion.addDependencies(); err != nil {
		return err
	}

	return nil
}

func (chartVersion *ChartVersion) CreateOrUpdateSubCharts() error {

	for _, e := range chartVersion.deps {
		chartVersion.logger.Info("create or update child chart", "child", e.Name, "version", e.Version)
		if err := chartVersion.createOrUpdateSubChart(e); err != nil {
			chartVersion.logger.Info("failed to manage subchart", "chart", e.Name, "error", err.Error())
			return err
		}
	}

	return nil
}

func (chartVersion *ChartVersion) getControllerRepo() (*yahov1alpha2.Repository, error) {
	instance := &yahov1alpha2.Repository{}

	// TODO:should we use a pointer?
	if chartVersion.owner == "" {
		return instance, errors.New("chart api resource not present")
	}

	err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{
		Name: chartVersion.repo,
	}, instance)

	if err != nil {
		if k8serrors.IsNotFound(err) {
			chartVersion.logger.Info("HelmRepo resource not found.", "name", chartVersion.repo)
			return instance, err
		}
		// Error reading the object - requeue the request.
		chartVersion.logger.Error(err, "Failed to get ControllerRepo")
		return instance, err
	}

	return instance, nil
}

func (chartVersion *ChartVersion) getCredentials() *Auth {
	// TODO: again creds... implement it in config resource as a map for each repository and/or global auth
	repoObj := &yahov1alpha2.Repository{}
	if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{Name: chartVersion.repo}, repoObj); err != nil {
		return nil
	}
	namespace := chartVersion.namespace
	secretObj := &v1.Secret{}
	creds := &Auth{}

	if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: repoObj.Spec.AuthSecret}, secretObj); err != nil {
		return nil
	}

	if _, ok := secretObj.Data["user"]; !ok {
		chartVersion.logger.Info("Username empty for repo auth")
	}

	if _, ok := secretObj.Data["password"]; !ok {
		chartVersion.logger.Info("Password empty for repo auth")
	}

	username, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["user"]))
	pw, _ := b64.StdEncoding.DecodeString(string(secretObj.Data["password"]))
	creds.User = string(username)
	creds.Password = string(pw)

	return creds
}



func (chartVersion *ChartVersion) watchForSubResourceSync(subResource *yahov1alpha2.Chart) bool {

	r := &yahov1alpha2.ChartList{
		Items: []yahov1alpha2.Chart{
			*subResource,
		},
	}

	watcher, err := chartVersion.k8sClient.Watch(context.Background(), r)

	if err != nil {
		chartVersion.logger.Info("cannot get watcher for subresource")
		return false
	}

	defer watcher.Stop()

	select {
	case res := <-watcher.ResultChan():
		ch := res.Object.(*yahov1alpha2.Chart)

		if res.Type == watch.Modified {

			synced := "synced"
			if *ch.Status.Dependencies == synced && *ch.Status.Versions == synced {
				return true
			}
		}
	case <-time.After(10 * time.Second):
		return false
	}

	return false
}

func (chartVersion *ChartVersion) getParsedVersion(version string, index repo.ChartVersions) (string, error) {

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
*/
