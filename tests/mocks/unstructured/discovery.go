package unstructured

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServerResourcesForGroupVersion represents mock func for similar runtime discovery client func
func (getter *K8SDiscoveryMock) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	args := getter.Called(groupVersion)
	v := args.Get(0).(*metav1.APIResourceList)
	err := args.Error(1)
	return v, err
}

// ServerResources represents mock func for similar runtime discovery client func
func (getter *K8SDiscoveryMock) ServerResources() ([]*metav1.APIResourceList, error) {
	args := getter.Called()
	v := args.Get(0).([]*metav1.APIResourceList)
	err := args.Error(1)
	return v, err
}

// ServerGroupsAndResources represents mock func for similar runtime discovery client func
func (getter *K8SDiscoveryMock) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	args := getter.Called()
	v := args.Get(0).([]*metav1.APIGroup)
	y := args.Get(1).([]*metav1.APIResourceList)
	err := args.Error(2)
	return v, y, err
}

// ServerPreferredResources represents mock func for similar runtime discovery client func
func (getter *K8SDiscoveryMock) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	args := getter.Called()
	v := args.Get(0).([]*metav1.APIResourceList)
	err := args.Error(1)
	return v, err
}

// ServerPreferredNamespacedResources represents mock func for similar runtime discovery client func
func (getter *K8SDiscoveryMock) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	args := getter.Called()
	v := args.Get(0).([]*metav1.APIResourceList)
	err := args.Error(1)
	return v, err
}
