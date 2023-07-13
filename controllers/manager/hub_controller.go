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
	"reflect"
	"time"

	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"github.com/soer3n/yaho/internal/hub"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HubReconciler reconciles a Chart object
type HubReconciler struct {
	client.WithWatch
	Hubs           map[string]hub.Hub
	WatchNamespace string
	Log            logr.Logger
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Chart object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *HubReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("hubs", req.NamespacedName)
	_ = r.Log.WithValues("hubsreq", req)

	// fetch app instance
	instance := &yahov1alpha2.Hub{}

	reqLogger.Info("start reconcile loop")

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("HelmChart resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmChart")
		return ctrl.Result{}, err
	}

	currentHub, ok := r.Hubs[instance.ObjectMeta.Name]
	if !ok {
		r.Log.Info("setting current hub", "hub", instance.ObjectMeta.Name)
		currentHub = hub.Hub{
			Backends: make(map[string]hub.BackendInterface),
		}
		r.Hubs[instance.ObjectMeta.Name] = currentHub
	}

	// validate if specified clusters have an active channel
	for _, item := range instance.Spec.Clusters {
		r.Log.Info("parsing specified cluster", "hub", instance.ObjectMeta.Name, "cluster", item.Name)
		localCluster, ok := currentHub.Backends[item.Name]

		if !ok {
			r.Log.Info("initializing specified cluster", "hub", instance.ObjectMeta.Name, "cluster", item.Name)
			// TODO: get specified secret with kubeconfig
			secret := &v1.Secret{}

			if err := r.WithWatch.Get(ctx, types.NamespacedName{Name: item.Secret.Name, Namespace: item.Secret.Namespace}, secret, &client.GetOptions{}); err != nil {
				return ctrl.Result{}, err
			}

			r.Log.Info("initiate cluster", "name", item.Name, "namespace", r.WatchNamespace)

			ctx, cancelFunc := context.WithCancel(ctx)
			localCluster, err = hub.NewClusterBackend(item.Name, r.WatchNamespace, secret.Data[item.Secret.Key], r.WithWatch, hub.Defaults{}, r.Scheme, r.Log, cancelFunc)

			r.Log.Info("new cluster", "cluster", localCluster)

			if err != nil {
				return ctrl.Result{}, err
			}

			r.Log.Info("adding specified cluster to hub channel", "hub", instance.ObjectMeta.Name, "cluster", localCluster)

			duration, _ := time.ParseDuration(item.Interval)
			if err := currentHub.AddBackend(localCluster, ctx, duration); err != nil {
				return r.syncStatus(ctx, instance, currentHub, err)
			}
		}

		r.Log.Info("updating specified cluster", "hub", instance.ObjectMeta.Name, "cluster", item.Name)
		r.Hubs[instance.ObjectMeta.Name] = currentHub
		if err := currentHub.UpdateBackend(localCluster); err != nil {
			return r.syncStatus(ctx, instance, currentHub, err)
		}
	}

	// validate if there are removed clusters by comparing specified and items from status
	for key := range instance.Status.Backends {
		r.Log.Info("parsing cluster from status for validate deletion", "hub", instance.ObjectMeta.Name, "cluster", key)
		markedToDelete := true
		for _, i := range instance.Spec.Clusters {
			if i.Name == key {
				markedToDelete = false
				break
			}
		}
		if markedToDelete || len(instance.Spec.Clusters) < 1 {
			r.Log.Info("remove cluster", "hub", instance.ObjectMeta.Name, "cluster", key)
			if err := currentHub.RemoveBackend(key); err != nil {
				return r.syncStatus(ctx, instance, currentHub, err)
			}
		}
	}

	r.Hubs[instance.ObjectMeta.Name] = currentHub
	return r.syncStatus(ctx, instance, currentHub, nil)
}

func (r *HubReconciler) syncStatus(ctx context.Context, instance *yahov1alpha2.Hub, hub hub.Hub, err error) (ctrl.Result, error) {
	currentBackends := make(map[string]yahov1alpha2.HubBackend)

	for _, b := range instance.Spec.Clusters {
		currentBackends[b.Name] = yahov1alpha2.HubBackend{
			Address: "",
			InSync:  true,
		}
	}

	if !reflect.DeepEqual(instance.Status.Backends, currentBackends) {
		instance.Status.Backends = currentBackends

		r.Log.Info("update status", "hub", instance.ObjectMeta.Name)
		if err := r.Status().Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HubReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&yahov1alpha2.Hub{}).
		Complete(r)
}
