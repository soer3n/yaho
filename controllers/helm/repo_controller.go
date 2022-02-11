/*
Copyright 2021.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package helm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/chart"
	"github.com/soer3n/yaho/internal/repository"
	oputils "github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var nopLogger = func(_ string, _ ...interface{}) {}

// RepoReconciler reconciles a Repo object
type RepoReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repos/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Repo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *RepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("repos", req.NamespacedName)
	_ = r.Log.WithValues("reposreq", req)

	// fetch app instance
	instance := &helmv1alpha1.Repo{}

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("HelmRepo resource not found. Ignoring since object must be deleted", "repo", req.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmRepo")
		return ctrl.Result{}, err
	}

	var hc *repository.Repo
	var requeue bool

	g := http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	settings := oputils.GetEnvSettings(map[string]string{})

	c := kube.Client{
		Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
		Log:     nopLogger,
	}

	hc = repository.New(instance, settings, reqLogger, r.Client, &g, c)

	if requeue, err = r.handleFinalizer(hc, instance); err != nil {
		reqLogger.Info("Failed on handling finalizer", "repo", instance.Spec.Name)
		return ctrl.Result{}, err
	}

	if requeue {
		reqLogger.Info("Update resource after modifying finalizer.")
		if err := r.Update(context.TODO(), instance); err != nil {
			reqLogger.Error(err, "error in reconciling")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	err = r.deploy(instance, hc)

	reqLogger.Info("Repo deployed", "name", instance.Spec.Name, "namespace", instance.ObjectMeta.Namespace)
	reqLogger.Info("Don't reconcile repos.", "name", instance.Spec.Name)
	return r.syncStatus(context.Background(), instance, err)
}

func (r *RepoReconciler) deploy(instance *helmv1alpha1.Repo, helmRepo *repository.Repo) error {
	var chartList []*chart.Chart
	var err error

	settings := oputils.GetEnvSettings(map[string]string{})

	// helmRepo := hc.GetRepo(instance.Spec.Name)
	label, repoGroupLabelOk := instance.ObjectMeta.Labels["repoGroup"]
	selector := map[string]string{"repo": helmRepo.Name}

	if repoGroupLabelOk && label != "" {
		selector["repoGroup"] = instance.ObjectMeta.Labels["repoGroup"]
	}

	if chartList, err = helmRepo.GetCharts(settings, selector); err != nil {
		r.Log.Info("Error on getting charts", "repo", instance.Spec.Name)
	}

	chartObjMap := make(map[string]*helmv1alpha1.Chart)

	for _, chart := range chartList {
		chartObjMap = chart.AddOrUpdateChartMap(chartObjMap, instance)
	}

	// this is use just for using channels, goroutines and waitGroup
	// could be senseful here if we have to deal with big repositories
	var wg sync.WaitGroup
	c := make(chan string, 1)

	for _, chartObj := range chartObjMap {
		wg.Add(1)

		go func(helmChart *helmv1alpha1.Chart, instance *helmv1alpha1.Repo, c chan<- string) {
			defer wg.Done()

			if err := controllerutil.SetControllerReference(instance, helmChart, r.Scheme); err != nil {
				r.Log.Error(err, "failed to set controller ref")
			}

			installedChart := &helmv1alpha1.Chart{}
			err := r.Client.Get(context.Background(), client.ObjectKey{
				Namespace: helmChart.ObjectMeta.Namespace,
				Name:      helmChart.Spec.Name,
			}, installedChart)
			if err != nil {
				if errors.IsNotFound(err) {
					r.Log.Info("Trying to install HelmChart " + helmChart.Name)

					if err = r.Client.Create(context.TODO(), helmChart); err != nil {
						r.Log.Info(err.Error())
						c <- installedChart.Spec.Name
					}
				}

				c <- ""
				return
			}

			installedChart.Spec = helmChart.Spec

			err = r.Client.Update(context.TODO(), installedChart)

			if err != nil {
				c <- installedChart.Spec.Name
			} else {
				r.Log.Info("chart is up to date", "chart", installedChart.Spec.Name)
				c <- ""
			}
		}(chartObj, instance, c)
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	failedChartList := []string{}

	for i := range c {
		if i != "" {
			failedChartList = append(failedChartList, i)
		}
	}

	r.Log.Info("chart parsing for %s completed.", "chart", instance.ObjectMeta.Name)

	if len(failedChartList) > 0 {
		return fmt.Errorf("problems with charts: %v", strings.Join(failedChartList, ","))
	}

	return nil
}

func (r *RepoReconciler) syncStatus(ctx context.Context, instance *helmv1alpha1.Repo, err error) (ctrl.Result, error) {
	stats := metav1.ConditionTrue
	message := ""
	reason := "install"

	if err != nil {
		stats = metav1.ConditionFalse
		message = err.Error()
	}

	if meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "synced", stats) && instance.Status.Conditions[0].Message == message {
		return ctrl.Result{}, nil
	}

	condition := metav1.Condition{Type: "synced", Status: stats, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: reason, Message: message}
	meta.SetStatusCondition(&instance.Status.Conditions, condition)

	_ = r.Status().Update(ctx, instance)

	r.Log.Info("Don't reconcile repo after status sync.")
	return ctrl.Result{}, nil
}

func (r *RepoReconciler) handleFinalizer(hc *repository.Repo, instance *helmv1alpha1.Repo) (bool, error) {

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		controllerutil.RemoveFinalizer(instance, "finalizer.repo.helm.soer3n.info")
		return true, nil
	}

	if !oputils.Contains(instance.GetFinalizers(), "finalizer.repo.helm.soer3n.info") {
		r.Log.Info("Adding Finalizer for the Quarantine Resource")
		controllerutil.AddFinalizer(instance, "finalizer.repo.helm.soer3n.info")
		return true, nil
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Repo{}).
		Complete(r)
}
