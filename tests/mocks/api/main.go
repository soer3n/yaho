package api

import (
	mocks "github.com/soer3n/yaho/tests/mocks/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetClientDiscoveryMock returns kubernetes discovery client mock
func GetClientDiscoveryMock() *mocks.K8SDiscoveryMock {

	dcMock := &mocks.K8SDiscoveryMock{}
	dcMock.On("ServerPreferredResources").Return([]*metav1.APIResourceList{
		{
			GroupVersion: "apiGroup/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "apiresource",
					Group:      "apiGroup",
					Namespaced: false,
					Verbs: metav1.Verbs{
						"foo", "bar",
					},
				},
			},
		},
	}, nil)

	return dcMock
}

// GetClientDynamicMock returns kubernetes dynamic client mock
func GetClientDynamicMock() *mocks.K8SDynamicClientMock {

	dcMock := &mocks.K8SDynamicClientMock{}

	gvr := schema.GroupVersionResource{
		Group:    "group",
		Version:  "v1",
		Resource: "resource",
	}
	namespaceableMock := mocks.K8SNamespaceMock{}
	dcMock.On("Resource", gvr).Return(&namespaceableMock)

	resourceMock := mocks.K8SResourceMock{}
	namespaceableMock.On("Namespace", "").Return(&resourceMock)

	objList := []unstructured.Unstructured{
		{
			Object: map[string]interface{}{"foo": "bar"},
		},
		{
			Object: map[string]interface{}{"bar": "foo"},
		},
	}
	usObj := &unstructured.UnstructuredList{
		Items: objList,
	}
	resourceMock.On("List", metav1.ListOptions{}).Return(usObj, nil)

	return dcMock
}
