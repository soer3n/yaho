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
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/go-utils/k8sutils"
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
	_ = r.Log.WithValues("releasegroup", req.NamespacedName)
	_ = r.Log.WithValues("releasegroupsreq", req)

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
			log.Info("HelmReleaseGroup resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HelmReleaseGroup")
		return ctrl.Result{}, err
	}

	hc := &k8sutils.HelmClient{
		Repos: &k8sutils.HelmRepos{},
		Env:   map[string]string{},
	}

	settings := hc.GetEnvSettings()
	hc.Env["RepositoryConfig"] = settings.RepositoryConfig
	hc.Env["RepositoryCache"] = settings.RepositoryCache

	// var repoList []*k8sutils.HelmRepo
	// var helmRepo *k8sutils.HelmRepo
	var helmRelease *helmv1alpha1.Release
	var repoPath, repoCache string

	repoLabel, repoLabelOk := instance.ObjectMeta.Labels["repo"]
	repoGroupLabel, repoGroupLabelOk := instance.ObjectMeta.Labels["repoGroup"]

	if repoLabelOk {
		repoPath = filepath.Dir(hc.Env["RepositoryConfig"])
		repoCache = hc.Env["RepositoryCache"]
		if repoGroupLabelOk {
			hc.Env["RepositoryConfig"] = repoPath + "/" + instance.ObjectMeta.Namespace + "/" + repoGroupLabel + "/repositories.yaml"
			hc.Env["RepositoryCache"] = repoCache + "/" + instance.ObjectMeta.Namespace + "/" + repoGroupLabel
		} else {
			hc.Env["RepositoryConfig"] = repoPath + "/" + instance.ObjectMeta.Namespace + "/" + repoLabel + "/repositories.yaml"
			hc.Env["RepositoryCache"] = repoCache + "/" + instance.ObjectMeta.Namespace + "/" + repoLabel
		}
	}

	//repoResource := &helmv1alpha1.Repo{}

	spec := instance.Spec.Releases

	for _, release := range spec {

		//	err = r.Get(context.Background(), client.ObjectKey{
		//		Namespace: instance.ObjectMeta.Namespace,
		//		Name:      repoName,
		//	}, repoResource)

		if err != nil {
			return ctrl.Result{}, err
		}

		log.Infof("Trying HelmRelease %v", release.Name)

		helmRelease = &helmv1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Name:      release.Name,
				Namespace: instance.ObjectMeta.Namespace,
				Labels: map[string]string{
					"release":   release.Name,
					"chart":     release.Chart,
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

		err = controllerutil.SetControllerReference(instance, helmRelease, r.Scheme)

		if err != nil {
			return ctrl.Result{}, err
		}

		installedRelease := &helmv1alpha1.Release{}
		err = r.Client.Get(context.Background(), client.ObjectKey{
			Namespace: helmRelease.ObjectMeta.Namespace,
			Name:      helmRelease.Spec.Name,
		}, installedRelease)

		if err != nil {
			if errors.IsNotFound(err) {
				err = r.Client.Create(context.TODO(), helmRelease)

				if err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, err
		}

		installedRelease.Spec = helmRelease.Spec
		r.Client.Update(context.TODO(), installedRelease)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.ReleaseGroup{}).
		Complete(r)
}
