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

// RepoReconciler reconciles a Repo object
type RepoReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm.soer3n.info,resources=repoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Repo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *RepoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("repos", req.NamespacedName)
	_ = r.Log.WithValues("reposreq", req)

	// fetch app instance
	instance := &helmv1alpha1.Repo{}

	log.Infof("Request: %v.\n", req)

	err := r.Get(ctx, req.NamespacedName, instance)

	log.Infof("Get: %v.\n", err)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HelmRepo resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get HelmRepo")
		return ctrl.Result{}, err
	}

	_, helmRepo, err := r.deployRepo(instance)

	if err != nil {
		return ctrl.Result{}, err
	}

	hc, err := r.getHelmClient(instance)

	if err != nil {
		return ctrl.Result{}, err
	}

	if _, err := r.handleFinalizer(reqLogger, helmRepo, hc, instance); err != nil {
		return ctrl.Result{}, err
	}

	//if repoGroupLabelOk {
	//	helmRepo.Name = repoGroupLabel
	//}

	chartList, err := helmRepo.GetCharts()

	if err != nil {
		return ctrl.Result{}, err
	}

	log.Infof("HelmChartCount: %v", len(chartList))

	chartObjMap := make(map[string]*helmv1alpha1.Chart)

	for _, chartMeta := range chartList {
		chartObjMap, err = helmutils.AddOrUpdateChartMap(chartObjMap, chartMeta, instance)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	for key, chartObj := range chartObjMap {
		log.Infof("Trying to install HelmChart %v", key)
		_, err = r.deployChart(chartObj, instance)

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("Don't reconcile.")
	return ctrl.Result{}, nil
}

func (r *RepoReconciler) addFinalizer(reqLogger logr.Logger, m *helmv1alpha1.Repo) error {
	reqLogger.Info("Adding Finalizer for the Repo")
	controllerutil.AddFinalizer(m, "finalizer.repo.helm.soer3n.info")

	// Update CR
	err := r.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update Repo with finalizer")
		return err
	}
	return nil
}

func (r *RepoReconciler) handleFinalizer(reqLogger logr.Logger, helmRepo *helmutils.HelmRepo, hc *helmutils.HelmClient, instance *helmv1alpha1.Repo) (ctrl.Result, error) {

	if !oputils.Contains(instance.GetFinalizers(), "finalizer.repo.helm.soer3n.info") {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := helmutils.HandleFinalizer(hc, instance); err != nil {
		return ctrl.Result{}, nil
	}

	controllerutil.RemoveFinalizer(instance, "finalizer.repo.helm.soer3n.info")
	return ctrl.Result{}, nil
}

func (r *RepoReconciler) deployRepo(instance *helmv1alpha1.Repo) (ctrl.Result, *helmutils.HelmRepo, error) {

	hc, err := r.getHelmClient(instance)

	if err != nil {
		return ctrl.Result{}, &helmutils.HelmRepo{}, err
	}

	helmRepo := hc.Repos.Entries[0]

	log.Infof("Get: %v.\n", helmRepo.Settings)

	if err = helmRepo.Update(); err != nil {
		return ctrl.Result{}, &helmutils.HelmRepo{}, err
	}

	err, entryObj := helmRepo.GetEntryObj()

	if err != nil {
		return ctrl.Result{}, &helmutils.HelmRepo{}, err
	}

	err = hc.Repos.UpdateRepoFile(entryObj)

	if err != nil {
		return ctrl.Result{}, &helmutils.HelmRepo{}, err
	}

	return ctrl.Result{}, helmRepo, nil
}

func (r *RepoReconciler) getHelmClient(instance *helmv1alpha1.Repo) (*helmutils.HelmClient, error) {

	hc := &helmutils.HelmClient{
		Repos: &helmutils.HelmRepos{},
		Env:   map[string]string{},
	}

	settings := hc.GetEnvSettings()
	hc.Env["RepositoryConfig"] = settings.RepositoryConfig
	hc.Env["RepositoryCache"] = settings.RepositoryCache

	var repoList []*helmutils.HelmRepo
	var helmRepo *helmutils.HelmRepo

	hc.Env["RepositoryConfig"], hc.Env["RepositoryCache"] = oputils.GetLabelsByInstance(instance.ObjectMeta, hc.Env)

	log.Infof("Trying HelmRepo %v", instance.Spec.Name)

	helmRepo = &helmutils.HelmRepo{
		Name:     instance.Spec.Name,
		Url:      instance.Spec.Url,
		Settings: hc.GetEnvSettings(),
	}

	if instance.Spec.Auth != nil {
		helmRepo.Auth = helmutils.HelmAuth{
			User:     instance.Spec.Auth.User,
			Password: instance.Spec.Auth.Password,
			Cert:     instance.Spec.Auth.Cert,
			Key:      instance.Spec.Auth.Key,
			Ca:       instance.Spec.Auth.Ca,
		}
	}

	repoList = append(repoList, helmRepo)

	hc.Repos.Entries = repoList
	hc.Repos.Settings = hc.GetEnvSettings()
	return hc, nil
}

func (r *RepoReconciler) deployChart(helmChart *helmv1alpha1.Chart, instance *helmv1alpha1.Repo) (ctrl.Result, error) {

	err := controllerutil.SetControllerReference(instance, helmChart, r.Scheme)

	if err != nil {
		return ctrl.Result{}, err
	}

	installedChart := &helmv1alpha1.Chart{}
	err = r.Client.Get(context.Background(), client.ObjectKey{
		Namespace: helmChart.ObjectMeta.Namespace,
		Name:      helmChart.Spec.Name,
	}, installedChart)

	if err != nil {
		if errors.IsNotFound(err) {
			err = r.Client.Create(context.TODO(), helmChart)

			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	installedChart.Spec = helmChart.Spec
	err = r.Client.Update(context.TODO(), installedChart)

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RepoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Repo{}).
		Complete(r)
}
