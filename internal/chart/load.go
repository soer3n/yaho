package chart

import (
	"context"
	"net/http"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetChartByURL represents func for downloading chart
func GetChartByURL(url string, opts *Auth, g utils.HTTPClientInterface) (*chart.Chart, error) {
	var resp *http.Response
	var err error

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return &chart.Chart{}, err
	}

	if opts != nil {
		if opts.User != "" && opts.Password != "" {
			req.SetBasicAuth(opts.User, opts.Password)
		}
	}

	if resp, err = g.Do(req); err != nil {
		return &chart.Chart{}, err
	}

	return loader.LoadArchive(resp.Body)
}

// GetChartURL represents func for returning url for downloading a chart
func GetChartURL(rc client.Client, chart, version, namespace string) (string, error) {
	var err error

	chartObj := &helmv1alpha1.Chart{}

	if err = rc.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: chart}, chartObj); err != nil {
		return "", err
	}

	return utils.GetChartVersion(version, chartObj).URL, nil
}
