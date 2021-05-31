/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helm

import (
	"context"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValuesReconciler reconciles a Values object
type ValuesReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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
	_ = r.Log.WithValues("charts", req.NamespacedName)
	_ = r.Log.WithValues("chartsreq", req)

	// fetch app instance
	instance := &helmv1alpha1.Values{}

	log.Infof("Request: %v.\n", req)

	err := r.Get(ctx, req.NamespacedName, instance)

	log.Infof("Get: %v.\n", err)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HelmValues resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HelmValues")
		return ctrl.Result{}, err
	}

	annotations := instance.GetAnnotations()

	if _, ok := annotations["releases"]; ok {
		releaseList := strings.Split(annotations["releases"], ",")

		for _, release := range releaseList {
			current := &helmv1alpha1.Release{}
			err := r.Client.Get(context.Background(), client.ObjectKey{
				Namespace: instance.ObjectMeta.Namespace,
				Name:      release,
			}, current)

			if err == nil {
				if meta.IsStatusConditionTrue(current.Status.Conditions, "synced") {
					condition := metav1.Condition{Type: "synced", Status: metav1.ConditionFalse, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: "valueschange", Message: "valuesupdated"}
					meta.SetStatusCondition(&current.Status.Conditions, condition)
					err = r.Status().Update(ctx, current)
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
