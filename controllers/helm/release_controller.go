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
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	helmutils "github.com/soer3n/apps-operator/pkg/helm"
	oputils "github.com/soer3n/apps-operator/pkg/utils"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "synced", metav1.ConditionTrue) {
		condition := metav1.Condition{Type: "synced", Status: metav1.ConditionTrue, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: "reconciling", Message: "reconcileSuccess"}
		meta.SetStatusCondition(&instance.Status.Conditions, condition)

		if err := r.Status().Update(context.Background(), instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	var hc *helmutils.HelmClient
	var helmRelease *helmutils.HelmRelease

	log.Infof("Trying HelmRelease %v", instance.Spec.Name)

	if !oputils.Contains(instance.GetFinalizers(), "finalizer.releases.helm.soer3n.info") {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return ctrl.Result{}, err
		}

		if err = r.Update(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	if hc, err = helmutils.GetHelmClient(instance); err != nil {
		return ctrl.Result{}, err
	}

	helmRelease = hc.Releases.Entries[0]

	var refList, subRefList []*helmutils.ValuesRef
	var valuesList []*helmv1alpha1.Values

	if instance.Spec.ValuesTemplate != nil && instance.Spec.ValuesTemplate.ValueRefs != nil {
		if valuesList, err = r.getValuesByReference(instance.Spec.ValuesTemplate.ValueRefs, instance.ObjectMeta.Namespace); err != nil {
			return ctrl.Result{}, err
		}
	}

	for _, valueObj := range valuesList {

		if subRefList, err = r.collectValues(valueObj, 0, instance); err != nil {
			return ctrl.Result{}, err
		}

		if err = r.updateValuesAnnotations(valueObj, instance); err != nil {
			return ctrl.Result{}, err
		}

		for _, subValueObj := range subRefList {
			refList = append(refList, subValueObj)
		}
	}

	log.Infof("Trying HelmRelease %v", refList)

	helmRelease.ValuesTemplate = helmutils.NewValueTemplate(refList)
	helmRelease.Namespace.Name = instance.ObjectMeta.Namespace
	helmRelease.Version = instance.Spec.Version

	//if helmRelease.Values, err = helmRelease.GetValues(); err != nil {
	//	return ctrl.Result{}, err
	//}

	_, controller := r.getControllerRepo(instance.Spec.Repo, instance.ObjectMeta.Namespace)

	for _, configmap := range helmRelease.GetParsedConfigMaps() {
		if err := r.deployConfigMap(configmap, controller); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err = helmRelease.Update(); err != nil {
		return ctrl.Result{}, err
	}

	if _, err = r.handleFinalizer(hc, instance); err != nil {
		return ctrl.Result{}, err
	}

	/*if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "synced", metav1.ConditionTrue) {
		condition := metav1.Condition{Type: "synced", Status: metav1.ConditionTrue, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: "success", Message: "updateSuccessful"}
		meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}*/

	log.Info("Don't reconcile releases.")
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

func (r *ReleaseReconciler) getControllerRepo(name, namespace string) (error, *helmv1alpha1.Repo) {
	instance := &helmv1alpha1.Repo{}

	err := r.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, instance)

	log.Infof("Get: %v.\n", err)
	log.Infof("Namespace: %v", namespace)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("HelmRepo resource not found. Ignoring since object must be deleted")
			return err, instance
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ControllerRepo")
		return err, instance
	}

	return nil, instance

}

func (r *ReleaseReconciler) handleFinalizer(helmClient *helmutils.HelmClient, instance *helmv1alpha1.Release) (ctrl.Result, error) {

	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		if _, err := helmutils.HandleFinalizer(helmClient, instance.ObjectMeta); err != nil {
			return ctrl.Result{}, nil
		}

		controllerutil.RemoveFinalizer(instance, "finalizer.releases.helm.soer3n.info")

		if err := r.Update(context.Background(), instance); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) deployConfigMap(configmap v1.ConfigMap, instance *helmv1alpha1.Repo) error {

	if err := controllerutil.SetControllerReference(instance, &configmap, r.Scheme); err != nil {
		return err
	}

	current := &v1.ConfigMap{}
	err := r.Client.Get(context.Background(), client.ObjectKey{
		Namespace: configmap.ObjectMeta.Namespace,
		Name:      configmap.ObjectMeta.Name,
	}, current)

	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Client.Create(context.TODO(), &configmap); err != nil {
				return err
			}
		}
		return err
	}

	for key, data := range current.BinaryData {

		val, ok := configmap.BinaryData[key]
		compare := bytes.Compare(val, data)

		if !ok || compare != 0 {
			if err = r.Client.Delete(context.TODO(), current); err != nil {
				return err
			}

			if err = r.Client.Create(context.TODO(), &configmap); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ReleaseReconciler) collectValues(values *helmv1alpha1.Values, count int32, release *helmv1alpha1.Release) ([]*helmutils.ValuesRef, error) {
	var list []*helmutils.ValuesRef

	// secure against infinite loop
	if count > 10 {
		return list, nil
	}

	entry := &helmutils.ValuesRef{
		Ref:    values,
		Parent: "base",
	}

	list = append(list, entry)

	for _, ref := range values.Spec.Refs {

		helmRef := &helmv1alpha1.Values{}

		err := r.Client.Get(context.Background(), client.ObjectKey{
			Namespace: values.ObjectMeta.Namespace,
			Name:      ref,
		}, helmRef)

		if err != nil {
			return list, err
		}

		if err = r.updateValuesAnnotations(helmRef, release); err != nil {
			return list, err
		}

		if helmRef.Spec.Refs != nil {
			nestedRef, err := r.collectValues(helmRef, (count + 1), release)
			if err != nil {
				return list, err
			}
			for _, nested := range nestedRef {
				list = append(list, nested)
			}
		}

		entry := &helmutils.ValuesRef{
			Ref:    helmRef,
			Parent: values.ObjectMeta.Name,
		}

		list = append(list, entry)
	}

	return list, nil
}

func (r *ReleaseReconciler) updateValuesAnnotations(obj *helmv1alpha1.Values, release *helmv1alpha1.Release) error {

	currentAnnotations := obj.ObjectMeta.GetAnnotations()

	if value, ok := currentAnnotations["releases"]; !ok {
		obj.ObjectMeta.Annotations["releases"] = release.ObjectMeta.Name
		patch := []byte(`{"metadata":{"annotations":{"releases": "` + obj.ObjectMeta.Annotations["releases"] + `"}}}`)
		return r.Client.Patch(context.TODO(), obj, client.RawPatch(types.StrategicMergePatchType, patch))
	} else {
		if !oputils.Contains(strings.Split(value, ","), release.ObjectMeta.Name) {
			obj.ObjectMeta.Annotations["releases"] = currentAnnotations["releases"] + "," + release.ObjectMeta.Name
			patch := []byte(`{"metadata":{"annotations":{"releases": "` + obj.ObjectMeta.Annotations["releases"] + `"}}}`)
			return r.Client.Patch(context.TODO(), obj, client.RawPatch(types.StrategicMergePatchType, patch))
		}
	}

	return nil
}

func (r *ReleaseReconciler) getValuesByReference(refs []string, namespace string) ([]*helmv1alpha1.Values, error) {
	var list []*helmv1alpha1.Values

	for _, ref := range refs {

		helmRef := &helmv1alpha1.Values{}

		err := r.Client.Get(context.Background(), client.ObjectKey{
			Namespace: namespace,
			Name:      ref,
		}, helmRef)

		if err != nil {
			if errors.IsNotFound(err) {
				helmRef.ObjectMeta.Namespace = namespace
				helmRef.ObjectMeta.Name = ref
				err = r.Client.Create(context.TODO(), helmRef)

				if err != nil {
					return list, err
				}
			}

			return list, err
		}

		list = append(list, helmRef)
	}

	return list, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Release{}).
		Owns(&helmv1alpha1.Values{}).
		Complete(r)
}
