package helm

import (
	"context"

	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	"github.com/stretchr/testify/mock"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func setConfig(clientMock *unstructuredmocks.K8SClientMock, httpMock *mocks.HTTPClientMock, name, namespace string, IsPresent bool) {

	var e error

	if !IsPresent {
		e = k8serrors.NewNotFound(schema.GroupResource{
			Group:    "foo",
			Resource: "bar",
		}, "notfound")
	}

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, &yahov1alpha2.Config{}).Return(e).Run(func(args mock.Arguments) {
		c := args.Get(2).(*yahov1alpha2.Config)
		c.ObjectMeta.Name = name
		c.ObjectMeta.Namespace = namespace
		c.Spec.Flags = &yahov1alpha2.Flags{
			DryRun:        false,
			DisableHooks:  false,
			CleanupOnFail: true,
		}
	})
}
