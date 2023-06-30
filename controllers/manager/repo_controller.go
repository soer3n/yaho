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
	"net/http"
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	"github.com/soer3n/yaho/internal/repository"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var nopLogger = func(_ string, _ ...interface{}) {}

const LabelPrefix = "yaho.soer3n.dev/"

// RepoReconciler reconciles a Repo object
type RepoReconciler struct {
	client.Client
	WatchNamespace string
	Log            logr.Logger
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
}

// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources="repositories",verbs=get;list;watch;update
// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources="charts",verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources=repositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources=repositories/finalizers,verbs=update

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

	reqLogger.Info("start reconcile loop")
	// fetch app instance
	instance := &helmv1alpha1.Repository{}

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

	synced := false
	if instance.Status.Synced == nil {
		instance.Status.Synced = &synced
	}

	g := http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	settings := utils.GetEnvSettings(map[string]string{})

	c := kube.Client{
		Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
		Log:     nopLogger,
	}

	hc = repository.New(instance, r.WatchNamespace, ctx, settings, reqLogger, r.Client, &g, c)

	// TODO:
	// should be before struct initialization
	// or divided into two tasks (finalizer creation and deletion)
	if requeue, err = r.handleFinalizer(hc, instance); err != nil {
		reqLogger.Info("Failed on handling finalizer", "repo", instance.Spec.Name)
		return ctrl.Result{}, err
	}

	if requeue {
		reqLogger.Info("Update resource after modifying finalizer.")
		if err := r.Update(ctx, instance); err != nil {
			reqLogger.Error(err, "error in reconciling")
		}

		return ctrl.Result{}, nil
	}

	if err = hc.Update(instance, r.Scheme); err != nil {
		return r.syncStatus(ctx, instance, err)
	}

	synced = true
	instance.Status.Synced = &synced

	reqLogger.Info("Repo deployed", "name", instance.Spec.Name, "namespace", instance.ObjectMeta.Namespace)
	reqLogger.Info("Don't reconcile repos.", "name", instance.Spec.Name)
	return r.syncStatus(ctx, instance, nil)
}

func (r *RepoReconciler) syncStatus(ctx context.Context, instance *helmv1alpha1.Repository, err error) (ctrl.Result, error) {
	stats := metav1.ConditionTrue
	message := ""
	reason := "install"

	// fetch umanaged charts related to current repository
	r.Log.Info("fetching unmanaged charts related to repository resource")
	unmanagedCharts := &helmv1alpha1.ChartList{}
	labelSetRepo, _ := labels.ConvertSelectorToLabelsMap(LabelPrefix + "repo=" + instance.Spec.Name)
	labelSetUnmanaged, _ := labels.ConvertSelectorToLabelsMap(LabelPrefix + "unmanaged=true")
	ls := labels.Merge(labelSetRepo, labelSetUnmanaged)

	r.Log.Info("selector", "labelset", ls)

	opts := &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(ls),
	}

	if err := r.List(context.Background(), unmanagedCharts, opts); err != nil {
		r.Log.Info("Error on listing unmanaged charts for repository %v", instance.Spec.Name)
	}

	r.Log.Info("unmanaged charts", "charts", unmanagedCharts.Items)

	newChartCount := int64(len(instance.Spec.Charts) + len(unmanagedCharts.Items))
	r.Log.Info("chartlength", "value", newChartCount)

	if err != nil {
		stats = metav1.ConditionFalse
		message = err.Error()
	}

	condition := metav1.Condition{Type: "synced", Status: stats, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: reason, Message: message}

	if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "synced", stats) || instance.Status.Conditions[0].Message != message {
		meta.SetStatusCondition(&instance.Status.Conditions, condition)
		instance.Status.Charts = &newChartCount
		_ = r.Status().Update(ctx, instance)
		return ctrl.Result{}, nil
	}

	if *instance.Status.Charts != newChartCount {
		instance.Status.Charts = &newChartCount
		_ = r.Status().Update(ctx, instance)
		return ctrl.Result{}, nil
	}

	if *instance.Status.Synced {
		r.Log.Info("Reconcile repo after status sync regular in 10 seconds.", "repo", instance.ObjectMeta.Name)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	r.Log.Info("Reconcile unsynced repo in 10 seconds.", "repo", instance.ObjectMeta.Name)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *RepoReconciler) handleFinalizer(hc *repository.Repo, instance *helmv1alpha1.Repository) (bool, error) {

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		controllerutil.RemoveFinalizer(instance, "finalizer.repo.yaho.soer3n.dev")
		return true, nil
	}

	if !utils.Contains(instance.GetFinalizers(), "finalizer.repo.yaho.soer3n.dev") {
		r.Log.Info("Adding Finalizer for the Repository Resource")
		controllerutil.AddFinalizer(instance, "finalizer.repo.yaho.soer3n.dev")
		return true, nil
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Repository{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
