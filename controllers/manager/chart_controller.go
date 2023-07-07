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

package manager

import (
	"context"
	"net/http"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"github.com/soer3n/yaho/internal/chart"
	"github.com/soer3n/yaho/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ChartReconciler reconciles a Chart object
type ChartReconciler struct {
	client.WithWatch
	WatchNamespace string
	Log            logr.Logger
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
}

// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources="repositories",verbs=get;list;watch
// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources=charts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources=charts/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=yaho.soer3n.dev,resources=charts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Chart object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *ChartReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("charts", req.NamespacedName)
	_ = r.Log.WithValues("chartsreq", req)

	// fetch app instance
	instance := &yahov1alpha2.Chart{}

	reqLogger.Info("start reconcile loop")

	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("HelmChart resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get HelmChart")
		return ctrl.Result{}, err
	}

	if instance.ObjectMeta.Labels == nil {
		instance.ObjectMeta.Labels = map[string]string{}
	}

	_, repoLabelIsSet := instance.ObjectMeta.Labels[LabelPrefix+"repo"]
	_, chartLabelIsSet := instance.ObjectMeta.Labels[LabelPrefix+"chart"]

	if !chartLabelIsSet {
		reqLogger.Info("update chart label")
		// set chart as label
		instance.ObjectMeta.Labels[LabelPrefix+"chart"] = instance.Spec.Name
	}

	if !repoLabelIsSet {
		reqLogger.Info("update repo label and add unmanaged label")
		// set repository as label
		instance.ObjectMeta.Labels[LabelPrefix+"repo"] = instance.Spec.Repository
		// set unmanaged label
		instance.ObjectMeta.Labels[LabelPrefix+"unmanaged"] = "true"
	}

	if !chartLabelIsSet || !repoLabelIsSet {
		// update resource after modifying labels and exit current reconcile loop
		_ = r.WithWatch.Update(ctx, instance)
	}

	settings := utils.GetEnvSettings(map[string]string{})

	g := http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	hc, err := chart.New(instance.Spec.Name, instance.Spec.Repository, r.WatchNamespace, &instance.Status, settings, r.Scheme, reqLogger, r.WithWatch, &g, settings.RESTClientGetter(), []byte{})

	if err != nil {
		reqLogger.Error(err, "failed to initialize chart resource struct", "name", instance.ObjectMeta.Name)
		return r.syncStatus(ctx, instance, hc.Status)
	}

	if err := hc.Update(instance); err != nil {
		reqLogger.Error(err, "failed to updatechart resource", "name", instance.ObjectMeta.Name)
		return r.syncStatus(ctx, instance, hc.Status)
	}

	//versions = "synced"
	//instance.Status.Versions = &versions

	if instance.Spec.CreateDeps {
		if err := hc.CreateOrUpdateSubCharts(); err != nil {
			condition := metav1.Condition{
				Type:               "dependenciesSync",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "chart update",
				Message:            err.Error(),
			}
			meta.SetStatusCondition(&instance.Status.Conditions, condition)
			reqLogger.Error(err, "error on managing subcharts. Reconciling.", "name", instance.ObjectMeta.Name)

		}

		//deps = "synced"
		//instance.Status.Dependencies = &deps
	}

	reqLogger.Info("chart resource is up to date", "name", instance.ObjectMeta.Name)

	return r.syncStatus(ctx, instance, hc.Status)
}

func (r *ChartReconciler) syncStatus(ctx context.Context, instance *yahov1alpha2.Chart, status *yahov1alpha2.ChartStatus) (ctrl.Result, error) {

	changed := false
	if !reflect.DeepEqual(instance.Status.ChartVersions, status.ChartVersions) {
		changed = true
		instance.Status.ChartVersions = status.ChartVersions
	}

	if r.setConditions(instance, status) {
		changed = true
		instance.Status.Conditions = status.Conditions
	}

	if !changed {
		r.Log.Info("nothing to update on status", "chart", instance.ObjectMeta.Name)
		return ctrl.Result{}, nil
	}

	if err := r.Status().Update(ctx, instance); err != nil {
		r.Log.Info("could not update status. reconcile", "chart", instance.ObjectMeta.Name)
		return ctrl.Result{}, err
	}

	r.Log.Info("don't reconcile.", "chart", instance.ObjectMeta.Name)
	return ctrl.Result{}, nil
}

func (r *ChartReconciler) setConditions(instance *yahov1alpha2.Chart, auth *yahov1alpha2.ChartStatus) bool {
	// TODO: use a map here
	changed := false
	if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "indexLoaded", meta.FindStatusCondition(auth.Conditions, "indexLoaded").Status) {
		changed = true
		condition := metav1.Condition{Type: "indexLoaded", Status: meta.FindStatusCondition(auth.Conditions, "indexLoaded").Status, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: meta.FindStatusCondition(auth.Conditions, "indexLoaded").Reason, Message: meta.FindStatusCondition(auth.Conditions, "indexLoaded").Message}
		meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}
	if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "configmapCreate", meta.FindStatusCondition(auth.Conditions, "configmapCreate").Status) {
		changed = true
		condition := metav1.Condition{Type: "configmapCreate", Status: meta.FindStatusCondition(auth.Conditions, "configmapCreate").Status, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: meta.FindStatusCondition(auth.Conditions, "configmapCreate").Reason, Message: meta.FindStatusCondition(auth.Conditions, "configmapCreate").Message}
		meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}
	if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "remoteSync", meta.FindStatusCondition(auth.Conditions, "remoteSync").Status) {
		changed = true
		condition := metav1.Condition{Type: "remoteSync", Status: meta.FindStatusCondition(auth.Conditions, "remoteSync").Status, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: meta.FindStatusCondition(auth.Conditions, "remoteSync").Reason, Message: meta.FindStatusCondition(auth.Conditions, "remoteSync").Message}
		meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}
	if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "dependenciesSync", meta.FindStatusCondition(auth.Conditions, "dependenciesSync").Status) {
		changed = true
		condition := metav1.Condition{Type: "dependenciesSync", Status: meta.FindStatusCondition(auth.Conditions, "dependenciesSync").Status, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: meta.FindStatusCondition(auth.Conditions, "dependenciesSync").Reason, Message: meta.FindStatusCondition(auth.Conditions, "dependenciesSync").Message}
		meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}
	if !meta.IsStatusConditionPresentAndEqual(instance.Status.Conditions, "prepareChart", meta.FindStatusCondition(auth.Conditions, "prepareChart").Status) {
		changed = true
		condition := metav1.Condition{Type: "prepareChart", Status: meta.FindStatusCondition(auth.Conditions, "prepareChart").Status, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: meta.FindStatusCondition(auth.Conditions, "prepareChart").Reason, Message: meta.FindStatusCondition(auth.Conditions, "prepareChart").Message}
		meta.SetStatusCondition(&instance.Status.Conditions, condition)
	}

	return changed
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&yahov1alpha2.Chart{}).
		Complete(r)
}