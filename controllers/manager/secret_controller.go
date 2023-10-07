/*
Copyright 2023.

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

package manager

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/soer3n/yaho/internal/hub"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecertReconciler reconciles a Secret Object and validating if there is a relationship to a configured hub
type SecretReconciler struct {
	client.WithWatch
	Hubs           map[string]hub.Hub
	WatchNamespace string
	Log            logr.Logger
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("secrets", req.NamespacedName)
	_ = r.Log.WithValues("secretsreq", req)

	// fetch app instance
	instance := &v1.Secret{}

	reqLogger.Info("start reconcile loop")

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Secret resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get Secret")
		return ctrl.Result{}, err
	}

	// 1. validate if there is a related hub
	// 2a. no -> return
	// 2b. yes -> check if hub is already managing the corresponding cluster
	// 3a. yes
	// 3aa. kubeconfig content has not changed -> return
	// 3ab. kubeconfig content has changed -> update backend -> return
	// 3b. no -> create new hub backend -> return
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Secret{}).
		Complete(r)
}
