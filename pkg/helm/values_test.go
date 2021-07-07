package helm

import (
	"encoding/json"
	"testing"

	helmv1alpha1 "github.com/soer3n/apps-operator/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/apps-operator/internal/types"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestValues(t *testing.T) {

	assert := assert.New(t)

	for _, testcase := range getTestValueSpecs() {
		vList := testcase.Input.([]*ValuesRef)
		testObj := NewValueTemplate(vList)
		_, err := testObj.ManageValues()

		assert.Equal(testcase.ReturnError, err)
	}
}

func getTestValueSpecs() []inttypes.TestCase {

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
			Input: []*ValuesRef{
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
