package client

import (
	mocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetClientDiscoveryMock returns kubernetes discovery client mock
func GetClientDiscoveryMock() *mocks.K8SDiscoveryMock {

	dcMock := &mocks.K8SDiscoveryMock{}

	gvr := schema.GroupVersionResource{
		Group:    "group",
		Version:  "v1",
		Resource: "resource",
	}

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
				{
					Name:       "apiresource2",
					Group:      "noApiGroup",
					Namespaced: false,
					Verbs: metav1.Verbs{
						"foo", "bar",
					},
				},
				{
					Name:       "apiresource3",
					Group:      "noApiGroup",
					Namespaced: true,
				},
				{
					Name:       "apiresource4",
					Group:      "apiGroup",
					Namespaced: true,
					Verbs: metav1.Verbs{
						"foo", "bar",
					},
				},
			},
		},
	}, nil)

	namespaceableMock := mocks.K8SNamespaceMock{}
	dcMock.On("Resource", gvr).Return(&namespaceableMock)

	resourceMock := mocks.K8SResourceMock{}
	namespaceableMock.On("Namespace", "namespace").Return(&resourceMock)

	usObj := &unstructured.UnstructuredList{}
	resourceMock.On("List", metav1.ListOptions{}).Return(usObj, nil)

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
	namespaceableMock.On("Namespace", "namespace").Return(&resourceMock)

	usObjList := &unstructured.UnstructuredList{}
	resourceMock.On("List", metav1.ListOptions{}).Return(usObjList, nil)

	usObj := &unstructured.Unstructured{}
	resourceMock.On("Get", "name").Return(usObj, nil)

	return dcMock
}
