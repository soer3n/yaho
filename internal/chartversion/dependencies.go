package chartversion

/*
func (chartVersion *ChartVersion) addDependencies() error {

		repoSelector := make(map[string]string)
		var group *string

		chartVersion.logger.Info("set selector")
		// TODO: use client to get reousrce and check then labels
		repoObj := &yahov1alpha2.Repository{}
		if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{Name: chartVersion.repo}, repoObj); err != nil {
			return err
		}
		if _, ok := repoObj.ObjectMeta.Labels[configMapRepoGroupLabelKey]; ok {
			if len(repoObj.ObjectMeta.Labels[configMapRepoGroupLabelKey]) > 1 {
				repoSelector[configMapRepoGroupLabelKey] = repoObj.ObjectMeta.Labels[configMapRepoGroupLabelKey]
				v := repoObj.ObjectMeta.Labels[configMapRepoGroupLabelKey]
				group = &v
			}
		}

		repoSelector[configMapRepoLabelKey] = chartVersion.repo

		chartVersion.logger.Info("create dependencies")
		chartVersion.deps = chartVersion.createDependenciesList(group)

		if err := chartVersion.loadDependencies(repoSelector); err != nil {
			return err
		}

		return nil
	}

	func (chartVersion *ChartVersion) createDependenciesList(group *string) []*yahov1alpha2.ChartDep {
		deps := make([]*yahov1alpha2.ChartDep, 0)

		if chartVersion.Obj == nil {
			return deps
		}

		for _, dep := range chartVersion.Version.Dependencies {

			repository, err := chartVersion.getRepoName(dep, group)

			if err != nil {
				chartVersion.logger.Info(err.Error())
			}

			for _, d := range chartVersion.Obj.Dependencies() {
				if err := chartVersion.updateIndexIfNeeded(d, dep); err != nil {
					chartVersion.logger.Info(err.Error())
				}
			}

			deps = append(deps, &yahov1alpha2.ChartDep{
				Name:      dep.Name,
				Version:   dep.Version,
				Repo:      repository,
				Condition: dep.Condition,
			})
		}

		return deps
	}

func (chartVersion *ChartVersion) updateIndexIfNeeded(d *chart.Chart, dep *chart.Dependency) error {

		if d.Metadata.Name == dep.Name {
			ix, err := utils.LoadChartIndex(chartVersion.Version.Name, chartVersion.repo, chartVersion.namespace, chartVersion.k8sClient)

			if err != nil {
				return err
			}

			x := *ix

			for k, e := range *ix {
				if e.Metadata.Version == chartVersion.Version.Version {
					if err := chartVersion.updateIndexVersion(d, dep, e, x, k); err != nil {
						return err
					}
				}
			}

			return nil
		}

		return nil
	}

func (chartVersion *ChartVersion) updateIndexVersion(d *chart.Chart, dep *chart.Dependency, e *repo.ChartVersion, x repo.ChartVersions, k int) error {

		for sk := range e.Metadata.Dependencies {
			if d.Metadata.Version != dep.Version {
				chartVersion.logger.Info("update index", "old", dep.Version, "new", d.Metadata.Version)
				dep.Version = d.Metadata.Version
				x[k].Metadata.Dependencies[sk].Version = d.Metadata.Version
				b, err := json.Marshal(x)

				if err != nil {
					return err
				}

				m := &v1.ConfigMap{}
				if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{
					Namespace: chartVersion.namespace,
					Name:      "helm-" + chartVersion.repo + "-" + chartVersion.owner + "-index",
				}, m); err != nil {
					return err
				}

				m.BinaryData = map[string][]byte{
					"versions": b,
				}

				if err := chartVersion.k8sClient.Update(context.Background(), m); err != nil {
					return err
				}
				return nil
			}
		}

		return nil
	}

func (chartVersion *ChartVersion) getRepoName(dep *chart.Dependency, group *string) (string, error) {

		repository := chartVersion.repo

		// if chartVersion.Obj != nil {

		// TODO: more logic needed for handling unmanaged charts !!!
		repoObj := &yahov1alpha2.Repository{}
		if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{Name: chartVersion.repo}, repoObj); err != nil {
			return repository, err
		}
		if repoObj.Spec.URL != dep.Repository {

			ls := labels.Set{}

			if group != nil {
				// filter repositories by group selector if set
				ls = labels.Merge(ls, labels.Set{"repoGroup": *group})
			}

			repoList := &yahov1alpha2.RepositoryList{}

			if err := chartVersion.k8sClient.List(context.Background(), repoList, &client.ListOptions{
				LabelSelector: labels.SelectorFromSet(ls),
			}); err != nil {
				return repository, err
			}

			if len(repoList.Items) == 0 {
				return repository, nil
			}

			for _, r := range repoList.Items {
				if r.Spec.URL == dep.Repository {
					return r.Name, nil

				}
			}
		}
		// }

		return repository, nil
	}
*/
