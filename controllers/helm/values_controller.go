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
	"strings"
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValuesReconciler reconciles a Values object
type ValuesReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=helm.soer3n.info,resources=values,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=values/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=values/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Chart object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ValuesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("charts", req.NamespacedName)
	_ = r.Log.WithValues("chartsreq", req)

	// fetch app instance
	instance := &helmv1alpha1.Values{}

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("HelmValues resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmValues")
		return ctrl.Result{}, err
	}

	annotations := instance.GetAnnotations()

	if _, ok := annotations["releases"]; ok {
		releaseList := strings.Split(annotations["releases"], ",")

		for _, release := range releaseList {
			current := &helmv1alpha1.Release{}
			err := r.Client.Get(ctx, client.ObjectKey{
				Namespace: instance.ObjectMeta.Namespace,
				Name:      release,
			}, current)

			if err == nil {
				if meta.IsStatusConditionTrue(current.Status.Conditions, "synced") {
					condition := metav1.Condition{Type: "synced", Status: metav1.ConditionFalse, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: "valueschange", Message: "valuesupdated"}
					meta.SetStatusCondition(&current.Status.Conditions, condition)
					err = r.Status().Update(ctx, current)
					reqLogger.Info("Trigger release sync.")
					if err != nil {
						return ctrl.Result{}, err
					}
				}
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ValuesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Values{}).
		Complete(r)
}
