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

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReleaseGroupReconciler reconciles a ReleaseGroup object
type ReleaseGroupReconciler struct {
	client.Client
	WatchNamespace string
	Log            logr.Logger
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
}

// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releasegroups,verbs=get;list;watch
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releases,verbs=get;list;watch;update;patch;delete
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releasegroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releasegroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ReleaseGroup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ReleaseGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("releasegroup", req.NamespacedName)
	_ = r.Log.WithValues("releasegroupsreq", req)

	// fetch app instance
	instance := &helmv1alpha1.ReleaseGroup{}

	reqLogger.Info("start reconcile loop")

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("HelmReleaseGroup resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmReleaseGroup")
		return ctrl.Result{}, err
	}

	var requeue bool

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil

	if requeue, err = r.handleFinalizer(instance, isRepoMarkedToBeDeleted, ctx); err != nil {
		reqLogger.Error(err, "Handle finalizer for release group %v failed.", instance.Spec.Name)
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

	// fetch owned repos
	releases := &helmv1alpha1.ReleaseList{}
	requirement, _ := labels.ParseToRequirements("releaseGroup=" + instance.Spec.LabelSelector)
	opts := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(requirement[0]),
	}

	if err := r.List(context.Background(), releases, opts); err != nil {
		r.Log.Info("Error on listing releases for group %v", instance.Spec.LabelSelector)
	}

	spec := instance.Spec.Releases
	remove := make(chan helmv1alpha1.Release)
	create := make(chan helmv1alpha1.Release)
	quit := make(chan bool)
	counter := 0

	go func() {
		for _, release := range releases.Items {
			exists := false

			for _, specRelease := range spec {
				if release.Name == specRelease.Name {
					exists = true
					break
				}
			}

			if !exists {
				remove <- release
			}
		}
		quit <- true
	}()

	go func() {
		for _, release := range spec {
			create <- helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      release.Name,
					Namespace: instance.ObjectMeta.Namespace,
					Labels: map[string]string{
						"release":      release.Name,
						"releaseGroup": instance.Spec.LabelSelector,
					},
				},
				Spec: release,
			}
		}
		quit <- true
	}()

	for {
		select {
		case f := <-remove:
			r.removeRelease(f, instance, ctx)
		case g := <-create:
			r.deployRelease(g, instance, ctx)
		case v := <-quit:
			if v {
				counter++
			}
			if counter == 2 {
				return ctrl.Result{}, nil
			}
		}
	}

}

func (r *ReleaseGroupReconciler) removeRelease(g helmv1alpha1.Release, instance *helmv1alpha1.ReleaseGroup, ctx context.Context) {
	if err := r.Delete(ctx, &g); err != nil {
		r.Log.Error(err, "error on remove", "group", instance.ObjectMeta.Name, "release", g.Name)
	}
	r.Log.Info("release removed", "group", instance.ObjectMeta.Name, "release", g.Name)
}

func (r *ReleaseGroupReconciler) deployRelease(g helmv1alpha1.Release, instance *helmv1alpha1.ReleaseGroup, ctx context.Context) {

	release := g.DeepCopy()

	/*
		if err := controllerutil.SetControllerReference(instance, release, r.Scheme); err != nil {
			r.Log.Error(err, "error on setting ref", "group", instance.ObjectMeta.Name, "release", release.Name)
			return
		}
	*/

	installedRepo := &helmv1alpha1.Release{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: release.ObjectMeta.Namespace,
		Name:      release.Spec.Name,
	}, installedRepo)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info(err.Error(), "group", instance.ObjectMeta.Name, "release", release.Name)

			if err = r.Client.Create(ctx, release); err != nil {
				r.Log.Error(err, "error on create", "group", instance.ObjectMeta.Name, "release", release.Name)
			}

			r.Log.Info("release created", "group", instance.ObjectMeta.Name, "release", release.Name)
		}
		return
	}
	r.Log.Info("Release already installed.", "group", instance.ObjectMeta.Name, "release", release.Name)
}

func (r *ReleaseGroupReconciler) handleFinalizer(instance *helmv1alpha1.ReleaseGroup, isRepoMarkedToBeDeleted bool, ctx context.Context) (bool, error) {

	if isRepoMarkedToBeDeleted {
		for _, rel := range instance.Spec.Releases {
			r.removeRelease(helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rel.Name,
					Namespace: instance.ObjectMeta.Namespace,
				},
				Spec: rel}, instance, ctx)
		}
		controllerutil.RemoveFinalizer(instance, "finalizer.releasegroups.helm.soer3n.info")
		return true, nil
	}

	if !utils.Contains(instance.GetFinalizers(), "finalizer.releasegroups.helm.soer3n.info") {
		r.Log.Info("Adding Finalizer for the Release Group")
		controllerutil.AddFinalizer(instance, "finalizer.releasegroups.helm.soer3n.info")
		return true, nil
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.ReleaseGroup{}).
		Complete(r)
}
