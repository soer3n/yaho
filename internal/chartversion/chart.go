package chartversion

import (
	"context"
	"errors"
	"net/http"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/repo"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (chartVersion *ChartVersion) setChart(chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) error {

	helmChart := &chart.Chart{}
	chartObj := &helmv1alpha1.Chart{}

	if err := chartVersion.k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: chartVersion.owner.ObjectMeta.Namespace,
		Name:      chartVersion.owner.Spec.Name,
	}, chartObj); err != nil {
		return err
	}

	if err := chartVersion.loadChartByResources(helmChart, chartObj, chartPathOptions, vals); err != nil {
		return err
	}

	chartVersion.Obj = helmChart
	return nil
}

func (chartVersion *ChartVersion) loadChartByURL(releaseClient *action.Install) error {

	releaseClient.ReleaseName = chartVersion.owner.Name

	if chartVersion.Version == nil {
		return errors.New("no chartversion loaded")
	}

	releaseClient.Version = chartVersion.Version.Version
	releaseClient.ChartPathOptions.RepoURL = chartVersion.repo.Spec.URL
	credentials := &Auth{}

	if chartVersion.repo.Spec.AuthSecret != "" {
		credentials = chartVersion.getCredentials()
	}

	if err := chartVersion.downloadChart(credentials); err != nil {
		return err
	}

	return nil
}

func (chartVersion *ChartVersion) setChartURL(index repo.ChartVersions) error {

	for _, e := range index {
		if e.Version == chartVersion.Version.Version {
			// use first url because it should be set in each case
			chartURL, err := repo.ResolveReferenceURL(chartVersion.repo.Spec.URL, chartVersion.Version.URLs[0])

			if err != nil {
				return err
			}

			chartVersion.url = chartURL
			return nil
		}
	}

	return errors.New("could not set chartversion url")
}

func (chartVersion *ChartVersion) loadChartByResources(helmChart *chart.Chart, apiObj *helmv1alpha1.Chart, chartPathOptions *action.ChartPathOptions, vals map[string]interface{}) error {

	chartVersion.wg.Add(3)

	go func() {
		defer chartVersion.wg.Done()
		chartVersion.setVersion(helmChart, apiObj, chartPathOptions)
	}()

	go func() {
		defer chartVersion.wg.Done()
		chartVersion.setValues(helmChart, apiObj, chartPathOptions, vals)
	}()

	go func() {
		defer chartVersion.wg.Done()
		chartVersion.setFiles(helmChart, apiObj, chartPathOptions)
	}()

	chartVersion.wg.Wait()

	if len(helmChart.Files) < 1 {
		return k8serrors.NewBadRequest("no files detected for chart resource")
	}

	// validate after channels are closed
	if err := helmChart.Validate(); err != nil {
		return err
	}

	return nil
}

func (chartVersion *ChartVersion) downloadChart(opts *Auth) error {
	var resp *http.Response
	var err error

	req, err := http.NewRequest(http.MethodGet, chartVersion.url, nil)
	if err != nil {
		chartVersion.logger.Info(err.Error())
		return err
	}

	if opts != nil {
		if opts.User != "" && opts.Password != "" {
			req.SetBasicAuth(opts.User, opts.Password)
		}
	}

	if resp, err = chartVersion.getter.Do(req); err != nil {
		chartVersion.logger.Info(err.Error())
		return err
	}

	chart, err := loader.LoadArchive(resp.Body)

	if err != nil {
		chartVersion.logger.Info(err.Error())
		return err
	}

	chartVersion.Obj = chart
	return nil
}
