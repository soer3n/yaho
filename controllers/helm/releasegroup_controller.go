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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
)

// ReleaseGroupReconciler reconciles a ReleaseGroup object
type ReleaseGroupReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=helm.soer3n.info,resources=releasegroups,verbs=get;list;watch;create;update;patch;delete
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
	_ = r.Log.WithValues("releases", req.NamespacedName)
	_ = r.Log.WithValues("releasessreq", req)

	// fetch app instance
	instance := &helmv1alpha1.ReleaseGroup{}

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

	var helmRelease *helmv1alpha1.Release
	spec := instance.Spec.Releases

	for key, release := range spec {

		log.Infof("Trying to install HelmRepo %v index %v", release.Name, key)
		helmRelease = &helmv1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Name: release.Name,
				Labels: map[string]string{
					"repo":      release.Repo,
					"repoGroup": instance.Spec.LabelSelector,
				},
			},
			Spec: helmv1alpha1.ReleaseSpec{
				Name:  release.Name,
				Repo:  release.Repo,
				Chart: release.Chart,
			},
		}

		if release.ValuesTemplate != nil {
			if release.ValuesTemplate.Values != nil {
				helmRelease.Spec.ValuesTemplate.Values = release.ValuesTemplate.Values
			}
			if release.ValuesTemplate.ValueFiles != nil {
				helmRelease.Spec.ValuesTemplate.ValueFiles = release.ValuesTemplate.ValueFiles
			}
		}

		err = r.Client.Create(context.TODO(), helmRelease)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.ReleaseGroup{}).
		Complete(r)
}
