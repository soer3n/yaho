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
	"sync"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	helmutils "github.com/soer3n/yaho/pkg/helm"
	oputils "github.com/soer3n/yaho/pkg/utils"
)

// ReleaseGroupReconciler reconciles a ReleaseGroup object
type ReleaseGroupReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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
	_ = r.Log.WithValues("releasegroup", req.NamespacedName)
	_ = r.Log.WithValues("releasegroupsreq", req)

	// fetch app instance
	instance := &helmv1alpha1.ReleaseGroup{}

	err := r.Get(ctx, req.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HelmReleaseGroup resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HelmReleaseGroup")
		return ctrl.Result{}, err
	}

	hc := &helmutils.Client{
		Repos: &helmutils.Repos{},
		Env:   map[string]string{},
	}

	settings := hc.GetEnvSettings()
	hc.Env["RepositoryConfig"] = settings.RepositoryConfig
	hc.Env["RepositoryCache"] = settings.RepositoryCache
	hc.Env["RepositoryConfig"], hc.Env["RepositoryCache"] = oputils.GetLabelsByInstance(instance.ObjectMeta, hc.Env)

	if _, ok := instance.Labels["release"]; !ok {

		instance.Labels = map[string]string{
			"release": instance.ObjectMeta.Name,
		}
	}

	var wg sync.WaitGroup
	c := make(chan string, 10)
	spec := instance.Spec.Releases

	for _, release := range spec {
		wg.Add(1)

		go func(release helmv1alpha1.ReleaseSpec, c chan<- string) {
			defer wg.Done()

			helmRelease := &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      release.Name,
					Namespace: instance.ObjectMeta.Namespace,
					Labels: map[string]string{
						"release": release.Name,
						"chart":   release.Chart,
						"repo":    release.Repo,
					},
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:    release.Name,
					Repo:    release.Repo,
					Chart:   release.Chart,
					Version: release.Version,
				},
			}

			if instance.Spec.LabelSelector != "" {
				helmRelease.ObjectMeta.Labels["repoGroup"] = instance.Spec.LabelSelector
			}

			if release.ValuesTemplate != nil {
				helmRelease.Spec.ValuesTemplate = release.ValuesTemplate
			}

			err := controllerutil.SetControllerReference(instance, helmRelease, r.Scheme)

			if err == nil {

				installedRelease := &helmv1alpha1.Release{}
				err = r.Client.Get(context.Background(), client.ObjectKey{
					Namespace: helmRelease.ObjectMeta.Namespace,
					Name:      helmRelease.Spec.Name,
				}, installedRelease)

				if err != nil {
					if errors.IsNotFound(err) {
						if err = r.Client.Create(context.TODO(), helmRelease); err != nil {
							c <- err.Error()
						}
						c <- "Successfully installed release " + helmRelease.Name
					}
				} else {

					installedRelease.Spec = helmRelease.Spec
					if err = r.Client.Update(context.TODO(), installedRelease); err != nil {
						c <- err.Error()
					}

					c <- "Successfully updated release " + installedRelease.Name
				}
			}
		}(release, c)
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for i := range c {
		log.Info(i)
	}

	log.Infof("Reconciled HelmReleaseGroup %v", instance.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.ReleaseGroup{}).
		Complete(r)
}
