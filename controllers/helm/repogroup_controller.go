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
			reqLogger.Info("HelmRepo resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmRepo")
		return ctrl.Result{}, err
	}

	// fetch owned repos
	repos := &helmv1alpha1.RepoList{}
	requirement, _ := labels.ParseToRequirements("repoGroup=" + instance.Spec.LabelSelector)
	opts := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(requirement[0]),
	}

	if err = r.List(context.Background(), repos, opts); err != nil {
		reqLogger.Info("Error on listing repos for group %v", instance.Spec.LabelSelector)
	}

	r.removeUnwantedRepos(repos, instance)

	reqLogger.Info("Trying to install HelmRepoSpecs", "groupname", instance.ObjectMeta.Name, "repos", instance.Spec.Repos)

	r.deployRepos(instance)

	return ctrl.Result{}, nil
}

func (r *RepoGroupReconciler) removeUnwantedRepos(repos *helmv1alpha1.RepoList, instance *helmv1alpha1.RepoGroup) {
	var wg sync.WaitGroup
	c := make(chan string, 10)
	spec := instance.Spec.Repos

	r.Log.Info("Trying to delete unwanted resoucres", "groupname", instance.ObjectMeta.Name, "repos", spec)

	for _, repo := range repos.Items {
		exists := false
		wg.Add(1)

		go func(spec []helmv1alpha1.RepoSpec, repo helmv1alpha1.Repo, c chan<- string) {
			defer wg.Done()

			for _, repository := range spec {
				if repo.Name == repository.Name {
					exists = true
					break
				}
			}

			if !exists {
				c <- "Delete unwanted repo: " + repo.Name
				if err := r.Delete(context.Background(), &helmv1alpha1.Repo{
					ObjectMeta: metav1.ObjectMeta{
						Name:      repo.Name,
						Namespace: instance.Namespace,
					},
				}); err != nil {
					c <- err.Error()
				}
			}
		}(spec, repo, c)

	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for i := range c {
		r.Log.Info(i)
	}
}

func (r *RepoGroupReconciler) deployRepos(instance *helmv1alpha1.RepoGroup) {
	var wg sync.WaitGroup
	c := make(chan string, 10)
	spec := instance.Spec.Repos

	for _, repository := range spec {
		wg.Add(1)

		go func(repository helmv1alpha1.RepoSpec, c chan<- string) {
			defer wg.Done()

			c <- "Trying to install HelmRepo " + repository.Name

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
					URL:  repository.URL,
				},
			}

			if repository.AuthSecret != "" {
				helmRepo.Spec.AuthSecret = repository.AuthSecret
			}

			if err := controllerutil.SetControllerReference(instance, helmRepo, r.Scheme); err != nil {
				c <- err.Error()
			}

			installedRepo := &helmv1alpha1.Repo{}
			err := r.Client.Get(context.Background(), client.ObjectKey{
				Namespace: helmRepo.ObjectMeta.Namespace,
				Name:      helmRepo.Spec.Name,
			}, installedRepo)
			if err != nil {
				if errors.IsNotFound(err) {
					c <- err.Error()

					if err = r.Client.Create(context.TODO(), helmRepo); err != nil {
						c <- err.Error()
					}
				}
			}
		}(repository, c)
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for i := range c {
		r.Log.Info(i)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.RepoGroup{}).
		Complete(r)
}
