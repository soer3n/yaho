package values

import (
	"context"
	"strings"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (hv *ValueTemplate) getValuesByReference(refs []string, namespace string) []*helmv1alpha1.Values {
	var list []*helmv1alpha1.Values

	for _, ref := range refs {

		helmRef := &helmv1alpha1.Values{}

		err := hv.k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: namespace,
			Name:      ref,
		}, helmRef)
		if err != nil {
			if errors.IsNotFound(err) {
				helmRef.ObjectMeta.Namespace = namespace
				helmRef.ObjectMeta.Name = ref
				err = hv.k8sClient.Create(context.TODO(), helmRef)

				if err != nil {
					hv.logger.Error(err, "error on create")
					continue
				}
			}

			hv.logger.Error(err, "not found")
			continue
		}

		hv.logger.Info("add value reference", "name", helmRef.ObjectMeta.Name)
		list = append(list, helmRef)
	}

	return list
}

func (hv *ValueTemplate) getRefList(valuesList []*helmv1alpha1.Values, instance *helmv1alpha1.Release) ([]*ValuesRef, error) {
	var refList, subRefList []*ValuesRef
	var err error
	for _, valueObj := range valuesList {

		if subRefList, err = hv.collectValues(valueObj, 0, instance); err != nil {
			return refList, err
		}

		if err = hv.updateValuesAnnotations(valueObj, instance); err != nil {
			return refList, err
		}

		refList = append(refList, subRefList...)
	}

	return refList, nil
}

func (hv *ValueTemplate) collectValues(specValues *helmv1alpha1.Values, count int32, release *helmv1alpha1.Release) ([]*ValuesRef, error) {
	var list []*ValuesRef

	// secure against infinite loop
	if count > 10 {
		return list, nil
	}

	entry := &ValuesRef{
		Ref:    specValues,
		Parent: "base",
	}

	list = append(list, entry)

	for _, ref := range specValues.Spec.Refs {

		helmRef := &helmv1alpha1.Values{}

		if err := hv.k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: specValues.ObjectMeta.Namespace,
			Name:      ref,
		}, helmRef); err != nil {
			return list, err
		}

		if err := hv.updateValuesAnnotations(helmRef, release); err != nil {
			hv.logger.Info("annotations error: %v", err)
			return list, err
		}

		if helmRef.Spec.Refs != nil {
			nestedRef, err := hv.collectValues(helmRef, (count + 1), release)
			if err != nil {
				return list, err
			}

			list = append(list, nestedRef...)
		}

		entry := &ValuesRef{
			Ref:    helmRef,
			Parent: specValues.ObjectMeta.Name,
		}

		list = append(list, entry)
	}

	return list, nil
}

func (hv *ValueTemplate) updateValuesAnnotations(obj *helmv1alpha1.Values, release *helmv1alpha1.Release) error {
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
		return hv.k8sClient.Patch(context.TODO(), obj, client.RawPatch(types.MergePatchType, patch))
	}

	if !utils.Contains(strings.Split(value, ","), release.ObjectMeta.Name) {
		obj.ObjectMeta.Annotations["releases"] = currentAnnotations["releases"] + "," + release.ObjectMeta.Name
		patch = []byte(`{"metadata":{"annotations":{"releases": "` + obj.ObjectMeta.Annotations["releases"] + `"}}}`)
		return hv.k8sClient.Patch(context.TODO(), obj, client.RawPatch(types.MergePatchType, patch))
	}

	return nil
}
