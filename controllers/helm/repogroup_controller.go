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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	reqLogger := r.Log.WithValues("repos", req.NamespacedName)
	_ = r.Log.WithValues("reposreq", req)

	// fetch app instance
	instance := &helmv1alpha1.RepoGroup{}

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("HelmRepoGroup resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmRepoGroup")
		return ctrl.Result{}, err
	}

	// fetch owned repos
	repos := &helmv1alpha1.RepoList{}
	requirement, _ := labels.ParseToRequirements("repoGroup=" + instance.Spec.LabelSelector)
	opts := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(requirement[0]),
	}

	if err := r.List(context.Background(), repos, opts); err != nil {
		r.Log.Info("Error on listing repos for group %v", instance.Spec.LabelSelector)
	}

	spec := instance.Spec.Repos
	remove := make(chan helmv1alpha1.Repo)
	create := make(chan helmv1alpha1.Repo)
	quit := make(chan bool)
	counter := 0

	go func() {
		for _, repo := range repos.Items {
			exists := false

			for _, repository := range spec {
				if repo.Name == repository.Name {
					exists = true
					break
				}
			}

			if !exists {
				remove <- repo
			}
		}
		quit <- true
	}()

	go func() {
		for _, repository := range spec {
			create <- helmv1alpha1.Repo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      repository.Name,
					Namespace: instance.ObjectMeta.Namespace,
					Labels: map[string]string{
						"repo":      repository.Name,
						"repoGroup": instance.Spec.LabelSelector,
					},
				},
				Spec: repository,
			}
		}
		quit <- true
	}()

	for {
		select {
		case f := <-remove:
			r.removeRepo(f, instance, ctx)
		case g := <-create:
			r.deployRepo(g, instance, ctx)
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

func (r *RepoGroupReconciler) removeRepo(repo helmv1alpha1.Repo, instance *helmv1alpha1.RepoGroup, ctx context.Context) {
	if err := r.Delete(ctx, &repo); err != nil {
		r.Log.Error(err, "error on remove", "group", instance.ObjectMeta.Name, "repo", repo.Name)
	}
	r.Log.Info("repo removed", "group", instance.ObjectMeta.Name, "repo", repo.Name)
}

func (r *RepoGroupReconciler) deployRepo(g helmv1alpha1.Repo, instance *helmv1alpha1.RepoGroup, ctx context.Context) {
	repo := g.DeepCopy()
	if err := controllerutil.SetControllerReference(instance, repo, r.Scheme); err != nil {
		r.Log.Error(err, "error on setting ref", "group", instance.ObjectMeta.Name, "repo", repo.Name)
		return
	}

	installedRepo := &helmv1alpha1.Repo{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Namespace: repo.ObjectMeta.Namespace,
		Name:      repo.Spec.Name,
	}, installedRepo)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info(err.Error(), "group", instance.ObjectMeta.Name, "repo", repo.Name)

			if err = r.Client.Create(ctx, repo); err != nil {
				r.Log.Error(err, "error on create", "group", instance.ObjectMeta.Name, "repo", repo.Name)
			}

			r.Log.Info("repo created", "group", instance.ObjectMeta.Name, "repo", repo.Name)
		}
		return
	}
	r.Log.Info("Repo already installed.", "group", instance.ObjectMeta.Name, "repo", repo.Name)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.RepoGroup{}).
		Complete(r)
}
