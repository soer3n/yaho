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

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	helmutils "github.com/soer3n/apps-operator/pkg/helm"
	oputils "github.com/soer3n/apps-operator/pkg/utils"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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

	log.Infof("Request: %v.\n", req)

	err := r.Get(ctx, req.NamespacedName, instance)

	log.Infof("Get: %v.\n", err)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HelmRelease resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HelmRelease")
		return ctrl.Result{}, err
	}

	var hc *helmutils.HelmClient
	var helmRelease *helmutils.HelmRelease

	log.Infof("Trying HelmRelease %v", instance.Spec.Name)

	if !oputils.Contains(instance.GetFinalizers(), "finalizer.releases.helm.soer3n.info") {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	if hc, err = helmutils.GetHelmClient(instance); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	helmRelease = hc.Releases.Entries[0]

	if err = helmRelease.Update(); err != nil {
		return ctrl.Result{}, err
	}

	if _, err = r.handleFinalizer(hc, instance); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) addFinalizer(reqLogger logr.Logger, m *helmv1alpha1.Release) error {
	reqLogger.Info("Adding Finalizer for the Release")
	controllerutil.AddFinalizer(m, "finalizer.releases.helm.soer3n.info")

	// Update CR
	if err := r.Update(context.TODO(), m); err != nil {
		reqLogger.Error(err, "Failed to update Release with finalizer")
		return err
	}
	return nil
}

func (r *ReleaseReconciler) handleFinalizer(helmClient *helmutils.HelmClient, instance *helmv1alpha1.Release) (ctrl.Result, error) {

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		if err := helmutils.HandleFinalizer(helmClient, instance); err != nil {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(instance, "finalizer.releases.helm.soer3n.info")
	}
	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) collectValues(values *helmv1alpha1.Values, namespace string, count int32) ([]*helmutils.ValuesRef, error) {
	var list []*helmutils.ValuesRef

	// secure against infinite loop
	if count > 10 {
		return list, nil
	}

	for _, ref := range values.Spec.Refs {

		helmRef := &helmv1alpha1.Values{}

		err := r.Client.Get(context.Background(), client.ObjectKey{
			Namespace: namespace,
			Name:      ref,
		}, helmRef)

		if err != nil {
			return list, err
		}

		if helmRef.Spec.Refs != nil {
			nestedRef, err := r.collectValues(helmRef, namespace, (count + 1))
			if err != nil {
				return list, err
			}
			for _, nested := range nestedRef {
				list = append(list, nested)
			}
		} else {

			entry := &helmutils.ValuesRef{
				Ref:    helmRef,
				Weight: count,
			}

			list = append(list, entry)
		}
	}

	return list, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Release{}).
		Complete(r)
}
