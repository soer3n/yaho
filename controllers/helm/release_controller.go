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

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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
	reqLogger := r.Log.WithValues("repos", req.NamespacedName)
	_ = r.Log.WithValues("reposreq", req)

	// fetch app instance
	instance := &helmv1alpha1.Release{}

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

	if releaseNamespace == "" {
		releaseNamespace = instance.ObjectMeta.Namespace
	}

	_ = os.Setenv("HELM_NAMESPACE", releaseNamespace)

	settings := utils.GetEnvSettings(map[string]string{})
	c := kube.Client{
		Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
		Log:     nopLogger,
	}

	helmRelease := release.New(instance, settings, reqLogger, r.Client, &g, c)

	if requeue, err = r.handleFinalizer(helmRelease, instance); err != nil {
		reqLogger.Error(err, "Handle finalizer for release %v failed.", helmRelease.Name)
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

	if instance.Spec.Values == nil {
		instance.Spec.Values = []string{}
	}

	if err := helmRelease.UpdateAffectedResources(r.Scheme); err != nil {
		return r.syncStatus(context.Background(), instance, metav1.ConditionFalse, "prepareFailed", err.Error())
	}

	if err := helmRelease.Update(); err != nil {
		return r.syncStatus(context.Background(), instance, metav1.ConditionFalse, "updateFailed", err.Error())
	}

	r.Log.Info("Don't reconcile releases.")
	return r.syncStatus(context.Background(), instance, metav1.ConditionTrue, "success", "all up to date")
}

func (r *ReleaseReconciler) handleFinalizer(helmRelease *release.Release, instance *helmv1alpha1.Release) (bool, error) {
	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		if err := helmRelease.RemoveRelease(); err != nil {
			return true, err
		}

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

func (r *ReleaseReconciler) syncStatus(ctx context.Context, instance *helmv1alpha1.Release, stats metav1.ConditionStatus, reason, message string) (ctrl.Result, error) {
	c := meta.FindStatusCondition(instance.Status.Conditions, "synced")
	if c != nil && c.Message == message && c.Status == stats {
		return ctrl.Result{}, nil
	}

	condition := metav1.Condition{Type: "synced", Status: stats, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: reason, Message: message}
	meta.SetStatusCondition(&instance.Status.Conditions, condition)

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info("Don't reconcile releases after sync.")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Release{}).
		Complete(r)
}
