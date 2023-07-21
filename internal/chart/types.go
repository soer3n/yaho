package chart

import (
	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Chart represents struct for data needed for managing chart
type Chart struct {
	Name       string
	Namespace  string
	Repo       string
	Status     *ChartStatus
	helm       helm
	kubernetes kubernetes
	getter     utils.HTTPClientInterface
	logger     logr.Logger
}

type kubernetes struct {
	scheme *runtime.Scheme
	client client.WithWatch
}

type helm struct {
	client   *action.Install
	settings *cli.EnvSettings
	index    repo.ChartVersions
	config   *action.Configuration
}

type ChartStatus struct {
	Conditions    *[]metav1.Condition
	ChartVersions map[string]yahov1alpha2.ChartVersion
	LinkedCharts  []string
	Deprecated    bool
}

// Auth represents struct with auth data for a repo
type Auth struct {
	User     string
	Password string
	Cert     string
	Key      string
	Ca       string
}
