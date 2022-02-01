package helm

import (
	"encoding/json"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/internal/helm"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GetTestValueSpecs returns expected spec for testing helm values parsing
func GetTestValueSpecs() []inttypes.TestCase {
	firstVals := map[string]string{"foo": "bar"}
	secVals := map[string]string{"foo": "bar"}
	thirdVals := map[string]interface{}{"baf": "muh", "boo": map[string]string{
		"fuz": "xyz",
	}, "mah": map[string]interface{}{
		"bah": map[string]string{
			"aah": "wah",
		},
	}}
	fourthVals := map[string]string{"foo": "bar"}

	firstValsRaw, _ := json.Marshal(firstVals)
	secValsRaw, _ := json.Marshal(secVals)
	thirdValsRaw, _ := json.Marshal(thirdVals)
	fourthValsRaw, _ := json.Marshal(fourthVals)

	return []inttypes.TestCase{
		{
			Input: []*helm.ValuesRef{
				{
					Ref: &helmv1alpha1.Values{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "",
						},
						Spec: helmv1alpha1.ValuesSpec{
							ValuesMap: &runtime.RawExtension{
								Raw: firstValsRaw,
							},
							Refs: map[string]string{
								"bar": "second",
								"boo": "third",
							},
						},
					},
					Parent: "base",
				},
				{
					Ref: &helmv1alpha1.Values{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "second",
							Namespace: "",
						},
						Spec: helmv1alpha1.ValuesSpec{
							ValuesMap: &runtime.RawExtension{
								Raw: secValsRaw,
							},
							Refs: map[string]string{
								"boo": "fourth",
							},
						},
					},
					Parent: "foo",
				},
				{
					Ref: &helmv1alpha1.Values{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "third",
							Namespace: "",
						},
						Spec: helmv1alpha1.ValuesSpec{
							ValuesMap: &runtime.RawExtension{
								Raw: thirdValsRaw,
							},
						},
					},
					Parent: "foo",
				},
				{
					Ref: &helmv1alpha1.Values{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fourth",
							Namespace: "",
						},
						Spec: helmv1alpha1.ValuesSpec{
							ValuesMap: &runtime.RawExtension{
								Raw: fourthValsRaw,
							},
						},
					},
					Parent: "second",
				},
			},
			ReturnError: nil,
		},
	}
}
