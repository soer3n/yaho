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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	"github.com/soer3n/go-utils/k8sutils"
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

	var helmRelease *k8sutils.HelmChart

	log.Infof("Trying HelmRelease %v", instance.Spec.Name)

	if !contains(instance.GetFinalizers(), "finalizer.releases.helm.soer3n.info") {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	hc, err := r.getHelmClient(instance)

	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Update(ctx, instance)

	if err != nil {
		return ctrl.Result{}, err
	}

	helmRelease = hc.Charts.Entries[0]

	if err = helmRelease.Update(); err != nil {
		return ctrl.Result{}, err
	}

	_, err = r.handleFinalizer(helmRelease, instance)

	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Update(ctx, instance)

	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) addFinalizer(reqLogger logr.Logger, m *helmv1alpha1.Release) error {
	reqLogger.Info("Adding Finalizer for the Release")
	controllerutil.AddFinalizer(m, "finalizer.releases.helm.soer3n.info")

	// Update CR
	err := r.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update Release with finalizer")
		return err
	}
	return nil
}

func (r *ReleaseReconciler) handleFinalizer(helmRelease *k8sutils.HelmChart, instance *helmv1alpha1.Release) (ctrl.Result, error) {

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		// Run finalization logic for memcachedFinalizer. If the
		// finalization logic fails, don't remove the finalizer so
		// that we can retry during the next reconciliation.
		log.Infof("Deletion: %v.\n", helmRelease)
		err := helmRelease.Remove()

		if err != nil {
			return ctrl.Result{}, err
		}
		// Remove memcachedFinalizer. Once all finalizers have been
		// removed, the object will be deleted.
		controllerutil.RemoveFinalizer(instance, "finalizer.releases.helm.soer3n.info")
	}
	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) getHelmClient(instance *helmv1alpha1.Release) (*k8sutils.HelmClient, error) {

	hc := &k8sutils.HelmClient{
		Repos: &k8sutils.HelmRepos{},
		Env:   map[string]string{},
	}

	settings := hc.GetEnvSettings()
	hc.Env["RepositoryConfig"] = settings.RepositoryConfig
	hc.Env["RepositoryCache"] = settings.RepositoryCache

	var releaseList []*k8sutils.HelmChart
	var helmRelease *k8sutils.HelmChart

	hc.Env["RepositoryConfig"], hc.Env["RepositoryCache"] = r.getLabelsByInstance(instance, hc.Env)

	err := r.Update(context.TODO(), instance)

	if err != nil {
		return &k8sutils.HelmClient{}, err
	}

	log.Infof("Trying HelmRepo %v", instance.Spec.Name)

	helmRelease = &k8sutils.HelmChart{
		Name:     instance.Spec.Name,
		Repo:     instance.Spec.Repo,
		Chart:    instance.Spec.Chart,
		Settings: hc.GetEnvSettings(),
	}

	actionConfig, err := helmRelease.GetActionConfig(helmRelease.Settings)

	if err != nil {
		return &k8sutils.HelmClient{}, err
	}

	helmRelease.Config = actionConfig

	log.Infof("HelmRelease config path: %v", helmRelease.Settings.RepositoryCache)

	if instance.Spec.ValuesTemplate != nil {
		if instance.Spec.ValuesTemplate.Values != nil {
			helmRelease.ValuesTemplate.Values = instance.Spec.ValuesTemplate.Values
		}
		if instance.Spec.ValuesTemplate.ValueFiles != nil {
			helmRelease.ValuesTemplate.ValueFiles = instance.Spec.ValuesTemplate.ValueFiles
		}
	}

	releaseList = append(releaseList, helmRelease)

	hc.Charts.Entries = releaseList
	hc.Charts.Settings = hc.GetEnvSettings()
	return hc, nil
}

func (r *ReleaseReconciler) getLabelsByInstance(instance *helmv1alpha1.Release, env map[string]string) (string, string) {

	var repoPath, repoCache string

	repoPath = filepath.Dir(env["RepositoryConfig"])
	repoCache = env["RepositoryCache"]

	repoLabel, repoLabelOk := instance.ObjectMeta.Labels["repo"]
	repoGroupLabel, repoGroupLabelOk := instance.ObjectMeta.Labels["repoGroup"]

	if repoLabelOk {
		if repoGroupLabelOk {
			repoPath = repoPath + "/" + instance.ObjectMeta.Namespace + "/" + repoGroupLabel + "/repositories.yaml"
			repoCache = repoCache + "/" + instance.ObjectMeta.Namespace + "/" + repoGroupLabel
		} else {
			repoPath = repoPath + "/" + instance.ObjectMeta.Namespace + "/" + repoLabel + "/repositories.yaml"
			repoCache = repoCache + "/" + instance.ObjectMeta.Namespace + "/" + repoLabel
		}
	}

	if _, ok := instance.ObjectMeta.Labels["release"]; !ok {

		instance.ObjectMeta.Labels = map[string]string{
			"release": instance.Spec.Name,
		}
	}

	return repoPath, repoCache
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Release{}).
		Complete(r)
}
