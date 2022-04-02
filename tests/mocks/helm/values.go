package helm

import (
	"context"
	"encoding/json"
	"strings"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	"github.com/stretchr/testify/mock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func setValues(clientMock *unstructuredmocks.K8SClientMock, httpMock *mocks.HTTPClientMock, valueMock valueMock) {

	var err error

	valsRaw, _ := json.Marshal(valueMock.Values)
	refMap := map[string]string{}

	for _, iv := range valueMock.Refs {
		refMap[iv.Key] = iv.Mock.Name
		setValues(clientMock, httpMock, iv.Mock)
	}

	if !valueMock.IsPresent {
		err = k8serrors.NewNotFound(schema.GroupResource{
			Group:    "foo",
			Resource: "bar",
		}, "notfound")

		clientMock.On("Create", context.Background(), &helmv1alpha1.Values{
			ObjectMeta: metav1.ObjectMeta{
				Name:        valueMock.Name,
				Namespace:   valueMock.Namespace,
				Annotations: map[string]string{},
			},
			Spec: helmv1alpha1.ValuesSpec{
				ValuesMap: &runtime.RawExtension{
					Raw: valsRaw,
				},
				Refs: refMap,
			},
		}).Return(nil).Run(func(args mock.Arguments) {})
	}

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: valueMock.Name, Namespace: valueMock.Namespace}, &helmv1alpha1.Values{}).Return(err).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Values)
		c.ObjectMeta = metav1.ObjectMeta{
			Name:        valueMock.Name,
			Namespace:   valueMock.Namespace,
			Annotations: map[string]string{},
		}
		c.Spec = helmv1alpha1.ValuesSpec{
			ValuesMap: &runtime.RawExtension{
				Raw: valsRaw,
			},
			Refs: refMap,
		}
	})

	if valueMock.IsPresent {
		releases := []string{}
		releases = append(releases, valueMock.Releases...)
		patch := []byte(`{"metadata":{"annotations":{"releases": "` + strings.Join(releases, ",") + `"}}}`)
		clientMock.On("Patch", context.Background(), &helmv1alpha1.Values{
			ObjectMeta: metav1.ObjectMeta{
				Name:        valueMock.Name,
				Namespace:   valueMock.Namespace,
				Annotations: map[string]string{"releases": strings.Join(releases, ",")},
			},
			Spec: helmv1alpha1.ValuesSpec{
				ValuesMap: &runtime.RawExtension{
					Raw: valsRaw,
				},
				Refs: refMap,
			},
		}, client.RawPatch(types.MergePatchType, patch)).Return(nil).Run(func(args mock.Arguments) {

		})
	}
}
