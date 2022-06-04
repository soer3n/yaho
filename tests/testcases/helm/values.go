package helm

import (
	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetTestValueSpecs returns expected spec for testing helm values parsing
func GetTestValueSpecs() []inttypes.TestCase {

	vm := map[string]interface{}{"foo": "bar", "boo": "baz"}
	embedded := map[string]interface{}{
		"ref": vm,
		"foo": "bar",
		"boo": "baz",
	}
	embedded2 := map[string]interface{}{
		"ref":  vm,
		"ref2": vm,
		"foo":  "bar",
		"boo":  "baz",
	}
	embeddedEmbedded := map[string]interface{}{
		"ref": embedded,
		"foo": "bar",
		"boo": "baz",
	}

	releaseSpec := []inttypes.TestCase{
		{
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "release",
					Namespace: "foo",
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:  "release",
					Chart: "chart",
					Repo:  "repo",
					Values: []string{
						"foo", "second",
					},
				},
			},
			ReturnError: map[string]error{
				"init":   nil,
				"manage": nil,
			},
			ReturnValue: vm,
		},
		{
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "release",
					Namespace: "foo",
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:  "release",
					Chart: "chart",
					Repo:  "repo",
					Values: []string{
						"foo", "second", "third", "fourth",
					},
				},
			},
			ReturnError: map[string]error{
				"init":   nil,
				"manage": nil,
			},
			ReturnValue: embedded,
		},
		{
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "release",
					Namespace: "foo",
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:  "release",
					Chart: "chart",
					Repo:  "repo",
					Values: []string{
						"fourth",
					},
				},
			},
			ReturnError: map[string]error{
				"init":   nil,
				"manage": nil,
			},
			ReturnValue: embedded,
		},
		{
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "release",
					Namespace: "foo",
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:  "release",
					Chart: "chart",
					Repo:  "repo",
					Values: []string{
						"fifth",
					},
				},
			},
			ReturnError: map[string]error{
				"init":   nil,
				"manage": nil,
			},
			ReturnValue: map[string]interface{}{
				"ref": embedded,
				"foo": "bar",
				"boo": "baz",
			},
		},
		{
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "release",
					Namespace: "foo",
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:  "release",
					Chart: "chart",
					Repo:  "repo",
					Values: []string{
						"fourth", "eigth",
					},
				},
			},
			ReturnError: map[string]error{
				"init":   nil,
				"manage": nil,
			},
			ReturnValue: embedded2,
		},
		{
			Input: &helmv1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "release",
					Namespace: "foo",
				},
				Spec: helmv1alpha1.ReleaseSpec{
					Name:  "release",
					Chart: "chart",
					Repo:  "repo",
					Values: []string{
						"sixth",
					},
				},
			},
			ReturnError: map[string]error{
				"init":   nil,
				"manage": nil,
			},
			ReturnValue: map[string]interface{}{
				"ref": embeddedEmbedded,
				"foo": "bar",
				"boo": "baz",
			},
		},
	}
	return releaseSpec
}
