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
	"reflect"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// RepoGroupReconciler reconciles a RepoGroup object
type RepoGroupReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repogroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repogroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repogroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RepoGroup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *RepoGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("repos", req.NamespacedName)
	_ = r.Log.WithValues("reposreq", req)

	// fetch app instance
	instance := &helmv1alpha1.RepoGroup{}

	err := r.Get(ctx, req.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("HelmRepo resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HelmRepo")
		return ctrl.Result{}, err
	}

	spec := instance.Spec.Repos

	log.Infof("Trying to install HelmRepoSpecs: %v", spec)

	for _, repository := range spec {

		if err = r.deployRepo(repository, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// fetch charts
	repos := &helmv1alpha1.RepoList{}
	requirement, _ := labels.ParseToRequirements("repoGroup=" + instance.Spec.LabelSelector)
	opts := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(requirement[0]),
	}

	err = r.List(ctx, repos, opts)

	for _, repo := range repos.Items {
		exists := false
		for _, repository := range spec {
			if reflect.DeepEqual(repo.Spec, repository) {
				exists = true
				break
			}
		}

		if !exists {
			log.Infof("Delete unwanted repo: %v.\n", repo.Spec.Name)
			if err = r.Delete(ctx, &repo); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *RepoGroupReconciler) deployRepo(repository helmv1alpha1.RepoSpec, instance *helmv1alpha1.RepoGroup) error {

	log.Infof("Trying to install HelmRepo %v", repository.Name)
	helmRepo := &helmv1alpha1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      repository.Name,
			Namespace: instance.ObjectMeta.Namespace,
			Labels: map[string]string{
				"repo":      repository.Name,
				"repoGroup": instance.Spec.LabelSelector,
			},
		},
		Spec: helmv1alpha1.RepoSpec{
			Name: repository.Name,
			Url:  repository.Url,
		},
	}

	if repository.Auth != nil {
		helmRepo.Spec.Auth = &helmv1alpha1.Auth{
			User:     repository.Auth.User,
			Password: repository.Auth.Password,
			Cert:     repository.Auth.Cert,
			Key:      repository.Auth.Key,
			Ca:       repository.Auth.Ca,
		}
	}

	err := controllerutil.SetControllerReference(instance, helmRepo, r.Scheme)

	if err != nil {
		return err
	}

	installedRepo := &helmv1alpha1.Repo{}
	err = r.Client.Get(context.Background(), client.ObjectKey{
		Namespace: helmRepo.ObjectMeta.Namespace,
		Name:      helmRepo.Spec.Name,
	}, installedRepo)

	if err != nil {
		if errors.IsNotFound(err) {
			err = r.Client.Create(context.TODO(), helmRepo)

			if err != nil {
				return err
			}
		}
		return nil
	}

	installedRepo.Spec = helmRepo.Spec
	err = r.Client.Update(context.TODO(), installedRepo)

	if err != nil {
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.RepoGroup{}).
		Complete(r)
}
