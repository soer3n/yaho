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
	"os"
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/release"
	"github.com/soer3n/yaho/internal/utils"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.WithWatch
	WatchNamespace string
	Log            logr.Logger
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
}

// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Release object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("release", req.NamespacedName)
	_ = r.Log.WithValues("releasereq", req)

	// fetch app instance
	instance := &helmv1alpha1.Release{}

	reqLogger.Info("start reconcile loop")

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("HelmRelease resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmRelease")
		return ctrl.Result{}, err
	}

	var requeue bool

	g := http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	releaseNamespace := instance.Spec.Namespace

	if releaseNamespace == nil {
		releaseNamespace = &instance.ObjectMeta.Namespace
	}

	_ = os.Setenv("HELM_NAMESPACE", *releaseNamespace)

	config, err := r.getConfig(instance.Spec, instance.ObjectMeta.Namespace)

	var releaseRestGetter genericclioptions.RESTClientGetter
	kubeconfig := ""

	if err == nil {
		releaseRestGetter, err = utils.NewRESTClientGetter(config, instance.ObjectMeta.Namespace, r.WithWatch, r.Log)

		if err != nil {
			r.Log.Info(err.Error())
			return ctrl.Result{}, err
		}

		casted := releaseRestGetter.(*utils.HelmRESTClientGetter)
		kubeconfig = casted.KubeConfig

		//restConfig, err := releaseRestGetter.ToRESTConfig()

		//if err != nil {
		//	r.Log.Info("error on getting rest client from helm client", "msg", err.Error())
		//}

		//settings := utils.GetEnvSettings(map[string]string{
		// "KubeConfig": releaseRestGetter.KubeConfig,
		//	"KubeToken": restConfig.BearerToken,
		//})
	} else {
		r.Log.Info(err.Error())
	}

	synced := false
	instance.Status.Synced = &synced

	if releaseRestGetter == nil {
		releaseRestGetter = cli.New().RESTClientGetter()
	}

	helmRelease, err := release.New(instance, r.WatchNamespace, r.Scheme, utils.GetEnvSettings(map[string]string{}), reqLogger, r.WithWatch, &g, releaseRestGetter, []byte(kubeconfig))

	if instance.Status.Revision == nil {

		reqLogger.Info("set inital revision status")

		status := "initResource"
		instance.Status.Status = &status

		initRevision := 0
		instance.Status.Revision = &initRevision

		if err := r.syncStatus(ctx, instance, metav1.ConditionFalse, "initResource", "start struct initialization", status, synced, initRevision); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err != nil {
		reqLogger.Info(err.Error(), "error on init struct", err.Error())
		status := "initError"

		if *instance.Status.Status == status {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		instance.Status.Status = &status

		if err := r.syncStatus(ctx, instance, metav1.ConditionFalse, "initError", err.Error(), status, synced, helmRelease.Revision); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil

	if requeue, err = r.handleFinalizer(helmRelease, instance, isRepoMarkedToBeDeleted); err != nil {
		reqLogger.Error(err, "Handle finalizer for release %v failed.", helmRelease.Name)
		return ctrl.Result{}, err
	}

	if requeue {
		if isRepoMarkedToBeDeleted {
			if err := helmRelease.RemoveRelease(); err != nil {
				return ctrl.Result{}, err
			}
		}
		reqLogger.Info("Update resource after modifying finalizer.")
		if err := r.Update(context.TODO(), instance); err != nil {
			reqLogger.Error(err, "error in reconciling")
			return ctrl.Result{}, err
		}

		// return ctrl.Result{}, nil
	}

	if instance.Spec.Values == nil {
		instance.Spec.Values = []string{}
	}

	if err := helmRelease.Update(); err != nil {
		status := "updateFailed"
		instance.Status.Status = &status

		if err := r.syncStatus(ctx, instance, metav1.ConditionFalse, "updateFailed", err.Error(), status, synced, helmRelease.Revision); err != nil {
			reqLogger.Info(err.Error())
		}

		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	status := "success"
	synced = true

	if err := r.syncStatus(ctx, instance, metav1.ConditionTrue, "success", "all up to date", status, synced, helmRelease.Revision); err != nil {
		return ctrl.Result{}, err
	}

	reqLogger.Info("Don't reconcile releases.")
	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) handleFinalizer(helmRelease *release.Release, instance *helmv1alpha1.Release, isRepoMarkedToBeDeleted bool) (bool, error) {

	if isRepoMarkedToBeDeleted {
		controllerutil.RemoveFinalizer(instance, "finalizer.releases.helm.soer3n.info")
		return true, nil
	}

	if !utils.Contains(instance.GetFinalizers(), "finalizer.releases.helm.soer3n.info") {
		r.Log.Info("Adding Finalizer for the Release")
		controllerutil.AddFinalizer(instance, "finalizer.releases.helm.soer3n.info")
		return true, nil
	}

	return false, nil
}

func (r *ReleaseReconciler) syncStatus(ctx context.Context, instance *helmv1alpha1.Release, stats metav1.ConditionStatus, reason, message, status string, synced bool, revision int) error {

	r.Log.Info("sync status", "release", instance.GetName())
	instanceLabels := instance.GetLabels()

	r.Log.Info("current labels", "value", instanceLabels)

	if _, ok := instanceLabels["helm.soer3n.info/reconcile"]; ok {

		delete(instanceLabels, "helm.soer3n.info/reconcile")
		instance.ObjectMeta.Labels = instanceLabels

		if err := r.Update(ctx, instance); err != nil {
			r.Log.Info("error on updating resource labels.", "error", err.Error())
			return err
		}
	}

	r.Log.Info("current status", "value", instance.Status)

	instance.Status.Status = &status
	instance.Status.Synced = &synced

	c := meta.FindStatusCondition(instance.Status.Conditions, "synced")
	if c != nil && c.Message == message && c.Status == stats {
		if *instance.Status.Revision == revision {
			r.Log.Info("status resource is already up to date.")
			return nil
		}
	}

	instance.Status.Revision = &revision
	condition := metav1.Condition{Type: "synced", Status: stats, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: reason, Message: message}
	meta.SetStatusCondition(&instance.Status.Conditions, condition)

	r.Log.Info("updated labels", "value", instanceLabels)
	r.Log.Info("current release status", "value", instance.Status)

	if err := r.Status().Update(ctx, instance); err != nil {
		r.Log.Info("error on status resource update.", "error", err.Error())
		return err
	}

	r.Log.Info("updated status", "value", instance.Status)
	r.Log.Info("status resource and labels validated and updated.")
	return nil
}

func (r *ReleaseReconciler) getConfig(spec helmv1alpha1.ReleaseSpec, namespace string) (*helmv1alpha1.Config, error) {

	if spec.Config == nil {
		return nil, nil
	}

	instance := &helmv1alpha1.Config{}

	err := r.Get(context.Background(), types.NamespacedName{
		Name:      *spec.Config,
		Namespace: namespace,
	}, instance)

	if err != nil {
		return nil, err
	}

	return instance, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {

	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"helm.soer3n.info/reconcile": "true",
		},
	}

	lsPredicate, _ := predicate.LabelSelectorPredicate(selector)
	pred := predicate.Or(predicate.GenerationChangedPredicate{}, lsPredicate)

	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Release{}).
		WithEventFilter(pred).
		WithOptions(controller.Options{MaxConcurrentReconciles: 2}).
		Complete(r)
}
