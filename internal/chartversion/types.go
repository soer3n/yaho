package chartversion

import (
	"sync"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ChartVersion represents struct with needed data for returning needed data for managing a release
type ChartVersion struct {
	Version       *repo.ChartVersion
	Obj           *chart.Chart
	deps          []*helmv1alpha1.ChartDep
	repo          *helmv1alpha1.Repository
	owner         *helmv1alpha1.Chart
	scheme        *runtime.Scheme
	url           string
	Templates     []*chart.File
	CRDs          []*chart.File
	DefaultValues map[string]interface{}
	k8sClient     client.Client
	getter        utils.HTTPClientInterface
	logger        logr.Logger
	mu            sync.Mutex
	wg            sync.WaitGroup
}

// Auth represents struct with auth data for a repo
type Auth struct {
	User     string
	Password string
	Cert     string
	Key      string
	Ca       string
}
