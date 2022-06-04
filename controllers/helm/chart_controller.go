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
	"time"

	"github.com/go-logr/logr"
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
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
	instance := &helmv1alpha1.Chart{}

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

	// set intial status
	versions := "notSynced"
	deps := "notSynced"
	instance.Status.Versions = &versions
	instance.Status.Dependencies = &deps

	if instance.ObjectMeta.Labels == nil {
		instance.ObjectMeta.Labels = map[string]string{}
	}

	_, repoLabelIsSet := instance.ObjectMeta.Labels["repo"]
	_, chartLabelIsSet := instance.ObjectMeta.Labels["chart"]

	if !chartLabelIsSet {
		reqLogger.Info("update chart label")
		// set chart as label
		instance.ObjectMeta.Labels["chart"] = instance.Spec.Name
		_ = r.WithWatch.Update(ctx, instance)
	}

	if !repoLabelIsSet {
		reqLogger.Info("update repo label and add unmanaged label")
		// set repository as label
		instance.ObjectMeta.Labels["repo"] = instance.Spec.Repository
		// set unmanaged label
		instance.ObjectMeta.Labels["unmanaged"] = "true"
		// update resource after modifying labels and exit current reconcile loop
		_ = r.WithWatch.Update(ctx, instance)
		// return ctrl.Result{}, nil
	}

	settings := utils.GetEnvSettings(map[string]string{})

	g := http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	hc, err := chart.New(instance, r.WatchNamespace, settings, r.Scheme, reqLogger, r.WithWatch, &g, settings.RESTClientGetter(), []byte{})

	if err != nil {
		reqLogger.Info("failed to initialize chart resource struct", "name", instance.ObjectMeta.Name)
		return r.syncStatus(ctx, instance, metav1.ConditionTrue, "initChartFailed", err.Error())
	}

	if err := hc.Update(instance); err != nil {
		reqLogger.Info("failed to updatechart resource", "name", instance.ObjectMeta.Name)
		return r.syncStatus(ctx, instance, metav1.ConditionTrue, "createConfigmapsFailed", err.Error())
	}

	versions = "synced"
	instance.Status.Versions = &versions

	if instance.Spec.CreateDeps {
		if err := hc.CreateOrUpdateSubCharts(); err != nil {
			reqLogger.Info("error on managing subcharts. Reconciling.", "name", instance.ObjectMeta.Name, "error", err.Error())
			return r.syncStatus(ctx, instance, metav1.ConditionTrue, "createDepsFailed", err.Error())

		}

		deps = "synced"
		instance.Status.Dependencies = &deps
	}

	reqLogger.Info("chart up to date", "name", instance.ObjectMeta.Name)

	return r.syncStatus(ctx, instance, metav1.ConditionTrue, "success", "all up to date")
}

func (r *ChartReconciler) syncStatus(ctx context.Context, instance *helmv1alpha1.Chart, stats metav1.ConditionStatus, reason, message string) (ctrl.Result, error) {
	c := meta.FindStatusCondition(instance.Status.Conditions, "synced")
	if c != nil && c.Message == message && c.Status == stats {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	condition := metav1.Condition{Type: "synced", Status: stats, LastTransitionTime: metav1.Time{Time: time.Now()}, Reason: reason, Message: message}
	meta.SetStatusCondition(&instance.Status.Conditions, condition)

	if err := r.Status().Update(ctx, instance); err != nil {
		r.Log.Info("could not update status. reconcile", "chart", instance.ObjectMeta.Name)
		return ctrl.Result{}, err
	}

	r.Log.Info("don't reconcile.", "chart", instance.ObjectMeta.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.Chart{}).
		Complete(r)
}
