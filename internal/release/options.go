package release

import (
	"context"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (hc *Release) setOptions(instance *helmv1alpha1.Config, namespace *string) error {

	var rn string
	hc.Flags = instance.Spec.Flags

	if namespace == nil {
		rn = instance.ObjectMeta.Namespace
	} else {
		rn = *namespace
	}

	for _, v := range instance.Spec.Namespace.Allowed {
		if v == rn {
			return nil
		}
	}

	return errors.NewBadRequest("namespace not in allowed list")
}

func (hc *Release) getConfig(name *string) (*helmv1alpha1.Config, error) {
	instance := &helmv1alpha1.Config{}

	hc.logger.Info(hc.Namespace.Name)

	err := hc.K8sClient.Get(context.Background(), types.NamespacedName{
		Name:      *name,
		Namespace: hc.Namespace.Name,
	}, instance)

	return instance, err
}

func (hc *Release) setInstallFlags(client *action.Install) {
	if hc.Flags == nil {
		hc.logger.Info("no flags set for release", "name", hc.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
		return
	}

	client.Namespace = hc.releaseNamespace
	client.Atomic = hc.Flags.Atomic
	client.DisableHooks = hc.Flags.DisableHooks
	client.DisableOpenAPIValidation = hc.Flags.DisableOpenAPIValidation
	client.DryRun = hc.Flags.DryRun
	client.SkipCRDs = hc.Flags.SkipCRDs
	client.SubNotes = hc.Flags.SubNotes
	client.Timeout = hc.Flags.Timeout
	client.Wait = hc.Flags.Wait
}

func (hc *Release) setUpgradeFlags(client *action.Upgrade) {
	if hc.Flags == nil {
		hc.logger.Info("no flags set for release", "name", hc.Name, "chart", hc.Chart.Name(), "repo", hc.Repo)
		return
	}

	client.Atomic = hc.Flags.Atomic
	client.DisableHooks = hc.Flags.DisableHooks
	client.DisableOpenAPIValidation = hc.Flags.DisableOpenAPIValidation
	client.DryRun = hc.Flags.DryRun
	client.SkipCRDs = hc.Flags.SkipCRDs
	client.SubNotes = hc.Flags.SubNotes
	client.Timeout = hc.Flags.Timeout
	client.Wait = hc.Flags.Wait
	client.Force = hc.Flags.Force
	client.Recreate = hc.Flags.Recreate
	client.CleanupOnFail = hc.Flags.CleanupOnFail
}
