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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/release"
	oputils "github.com/soer3n/yaho/internal/utils"
	"github.com/soer3n/yaho/internal/values"
	"helm.sh/helm/v3/pkg/kube"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("HelmRelease resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmRelease")
		return ctrl.Result{}, err
	}

	var helmRelease *release.Release
	var requeue bool

	g := http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	releaseNamespace := instance.Spec.Namespace

	if releaseNamespace.Name == "" {
		releaseNamespace.Name = instance.ObjectMeta.Namespace
	}

	_ = os.Setenv("HELM_NAMESPACE", releaseNamespace.Name)

	settings := oputils.GetEnvSettings(map[string]string{})
	c := kube.Client{
		Factory: cmdutil.NewFactory(settings.RESTClientGetter()),
		Log:     nopLogger,
	}

	helmRelease = release.New(instance, settings, reqLogger, r.Client, &g, c)

	if requeue, err = r.handleFinalizer(helmRelease, instance); err != nil {
		reqLogger.Error(err, "Handle finalizer for release %v failed.", helmRelease.Name)
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

	var refList []*values.ValuesRef
	var valuesList []*helmv1alpha1.Values

	if instance.Spec.ValuesTemplate != nil && instance.Spec.ValuesTemplate.ValueRefs != nil {
		if valuesList, err = r.getValuesByReference(instance.Spec.ValuesTemplate.ValueRefs, instance.ObjectMeta.Namespace); err != nil {
			return r.syncStatus(ctx, instance, metav1.ConditionFalse, "failed", err.Error())
		}
	}

	reqLogger.Info("list of references", "name", instance.Spec.Name, "chart", instance.Spec.Chart, "repo", instance.Spec.Repo, "refs", refList)
	return r.update(helmRelease, releaseNamespace, valuesList, instance)
}

func (r *ReleaseReconciler) update(helmRelease *release.Release, releaseNamespace helmv1alpha1.Namespace, valuesList []*helmv1alpha1.Values, instance *helmv1alpha1.Release) (ctrl.Result, error) {
	refList, _ := r.getRefList(valuesList, instance)
	helmRelease.InitValuesTemplate(refList, instance.Spec.Version, instance.ObjectMeta.Namespace)
	controller, _ := r.getControllerRepo(instance.Spec.Repo, instance.ObjectMeta.Namespace)

	if instance.Spec.ValuesTemplate == nil {
		instance.Spec.ValuesTemplate = &helmv1alpha1.ValueTemplate{}
	}

	cm, c := helmRelease.GetParsedConfigMaps(instance.ObjectMeta.Namespace)

	for _, chart := range c {
		if err := r.updateChart(chart, controller); err != nil {
			return r.syncStatus(context.Background(), instance, metav1.ConditionFalse, "failed", err.Error())
		}
	}

	for _, configmap := range cm {
		if err := r.deployConfigMap(configmap, controller); err != nil {
			return r.syncStatus(context.Background(), instance, metav1.ConditionFalse, "failed", err.Error())
		}
	}

	// set flags for helm action from spec
	helmRelease.Flags = instance.Spec.Flags

	if err := helmRelease.Update(releaseNamespace); err != nil {
		return r.syncStatus(context.Background(), instance, metav1.ConditionFalse, "failed", err.Error())
	}

	r.Log.Info("Don't reconcile releases.")
	return r.syncStatus(context.Background(), instance, metav1.ConditionTrue, "success", "all up to date")
}

func (r *ReleaseReconciler) getRefList(valuesList []*helmv1alpha1.Values, instance *helmv1alpha1.Release) ([]*values.ValuesRef, error) {
	var refList, subRefList []*values.ValuesRef
	var err error
	for _, valueObj := range valuesList {

		if subRefList, err = r.collectValues(valueObj, 0, instance); err != nil {
			return refList, err
		}

		if err = r.updateValuesAnnotations(valueObj, instance); err != nil {
			return refList, err
		}

		refList = append(refList, subRefList...)
	}

	return refList, nil
}

func (r *ReleaseReconciler) getControllerRepo(name, namespace string) (*helmv1alpha1.Repo, error) {
	instance := &helmv1alpha1.Repo{}

	err := r.Get(context.Background(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("HelmRepo resource not found. Ignoring since object must be deleted")
			return instance, err
		}
		// Error reading the object - requeue the request.
		r.Log.Error(err, "Failed to get ControllerRepo")
		return instance, err
	}

	return instance, nil
}

func (r *ReleaseReconciler) handleFinalizer(helmRelease *release.Release, instance *helmv1alpha1.Release) (bool, error) {
	isRepoMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isRepoMarkedToBeDeleted {
		if err := helmRelease.RemoveRelease(); err != nil {
			return true, err
		}

		controllerutil.RemoveFinalizer(instance, "finalizer.releases.helm.soer3n.info")
		return true, nil
	}

	if !oputils.Contains(instance.GetFinalizers(), "finalizer.releases.helm.soer3n.info") {
		r.Log.Info("Adding Finalizer for the Release")
		controllerutil.AddFinalizer(instance, "finalizer.releases.helm.soer3n.info")
		return true, nil
	}

	return false, nil
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

	return nil
}

func (r *ReleaseReconciler) updateChart(chart helmv1alpha1.Chart, instance *helmv1alpha1.Repo) error {
	current := &helmv1alpha1.Chart{}
	err := r.Client.Get(context.Background(), client.ObjectKey{
		Namespace: chart.ObjectMeta.Namespace,
		Name:      chart.ObjectMeta.Name,
	}, current)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Client.Create(context.TODO(), &chart); err != nil {
				return err
			}
		}
		return err
	}

	if err = r.Client.Update(context.TODO(), &chart); err != nil {
		return err
	}

	return nil
}

func (r *ReleaseReconciler) collectValues(specValues *helmv1alpha1.Values, count int32, release *helmv1alpha1.Release) ([]*values.ValuesRef, error) {
	var list []*values.ValuesRef

	// secure against infinite loop
	if count > 10 {
		return list, nil
	}

	entry := &values.ValuesRef{
		Ref:    specValues,
		Parent: "base",
	}

	list = append(list, entry)

	for _, ref := range specValues.Spec.Refs {

		helmRef := &helmv1alpha1.Values{}

		if err := r.Client.Get(context.Background(), client.ObjectKey{
			Namespace: specValues.ObjectMeta.Namespace,
			Name:      ref,
		}, helmRef); err != nil {
			return list, err
		}

		if err := r.updateValuesAnnotations(helmRef, release); err != nil {
			r.Log.Info("annotations error: %v", err)
			return list, err
		}

		if helmRef.Spec.Refs != nil {
			nestedRef, err := r.collectValues(helmRef, (count + 1), release)
			if err != nil {
				return list, err
			}

			list = append(list, nestedRef...)
		}

		entry := &values.ValuesRef{
			Ref:    helmRef,
			Parent: specValues.ObjectMeta.Name,
		}

		list = append(list, entry)
	}

	return list, nil
}

func (r *ReleaseReconciler) updateValuesAnnotations(obj *helmv1alpha1.Values, release *helmv1alpha1.Release) error {
	var patch []byte
	var value string
	var ok bool

	currentAnnotations := obj.ObjectMeta.GetAnnotations()

	if value, ok = currentAnnotations["releases"]; !ok {
		if currentAnnotations == nil {
			obj.ObjectMeta.Annotations = make(map[string]string)
		}

		obj.ObjectMeta.Annotations["releases"] = release.ObjectMeta.Name
		patch := []byte(`{"metadata":{"annotations":{"releases": "` + obj.ObjectMeta.Annotations["releases"] + `"}}}`)
		return r.Client.Patch(context.TODO(), obj, client.RawPatch(types.MergePatchType, patch))
	}

	if !oputils.Contains(strings.Split(value, ","), release.ObjectMeta.Name) {
		obj.ObjectMeta.Annotations["releases"] = currentAnnotations["releases"] + "," + release.ObjectMeta.Name
		patch = []byte(`{"metadata":{"annotations":{"releases": "` + obj.ObjectMeta.Annotations["releases"] + `"}}}`)
		return r.Client.Patch(context.TODO(), obj, client.RawPatch(types.MergePatchType, patch))
	}

	return nil
}

func (r *ReleaseReconciler) syncStatus(ctx context.Context, instance *helmv1alpha1.Release, stats metav1.ConditionStatus, reason, message string) (ctrl.Result, error) {
	if meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "synced", stats) && instance.Status.Conditions[0].Message == message {
		return ctrl.Result{}, nil
	}

	condition := metav1.Condition{Type: "synced", Status: stats, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: reason, Message: message}
	meta.SetStatusCondition(&instance.Status.Conditions, condition)

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	r.Log.Info("Don't reconcile releases after sync.")
	return ctrl.Result{}, nil
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
		Complete(r)
}
