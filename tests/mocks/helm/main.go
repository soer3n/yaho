package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
)

func GetChartMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {

	clientMock := unstructuredmocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "foo", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := testcases.GetTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)
	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.2.tgz").Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)

	req, _ := http.NewRequest(http.MethodGet, "https://foo.bar/charts/foo-0.0.1.tgz", nil)

	httpMock.On("Do",
		req).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)

	reqEmpty, _ := http.NewRequest(http.MethodGet, "https://foo.bar/charts/foo-0.0.2.tgz", nil)
	httpMock.On("Do",
		reqEmpty).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}, nil)

	return &clientMock, &httpMock
}

func GetClientMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {

	clientMock := unstructuredmocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "selector",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "notpresent",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	indexFile := testcases.GetTestClientIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/index.yaml").Return(httpResponse, nil)

	return &clientMock, &httpMock
}

func GetFinalizerMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {

	clientMock := unstructuredmocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "selector",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "notpresent",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	indexFile := testcases.GetTestFinalizerIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/index.yaml").Return(httpResponse, nil)

	return &clientMock, &httpMock
}

func GetReleaseMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {

	clientMock := unstructuredmocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "repo", Namespace: ""}, &helmv1alpha1.Repo{}).Return(nil)
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "notfound", Namespace: ""}, &helmv1alpha1.Repo{}).Return(errors.New("repo not found"))
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "chart", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := testcases.GetTestChartSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "notfound", Namespace: ""}, &helmv1alpha1.Chart{}).Return(errors.New("chart not found"))

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-tmpl-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseTemplateConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-crds-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseCRDConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-default-chart-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseDefaultValueConfigMap()
		c.Data = spec.Data
		c.ObjectMeta = spec.ObjectMeta
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.MatchingLabels{"repoGroup": "group"}, client.InNamespace("")}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(1).(*helmv1alpha1.ChartList)
		spec := helmv1alpha1.ChartList{
			Items: []helmv1alpha1.Chart{
				{
					Spec: helmv1alpha1.ChartSpec{
						Name: "dep",
					},
				},
				testcases.GetTestChartSpec(),
			},
		}
		c.Items = spec.Items

	})

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "dep", Namespace: ""}, &helmv1alpha1.Chart{}).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		spec := testcases.GetTestChartDepSpec()
		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-tmpl-dep-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseTemplateConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-crds-dep-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseCRDConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta = spec.ObjectMeta
	})
	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-default-dep-0.0.1", Namespace: ""}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseDefaultValueConfigMap()
		c.Data = spec.Data
		c.ObjectMeta = spec.ObjectMeta
	})

	var payload []byte

	raw, _ := os.Open("../../testutils/busybox-0.1.0.tgz")
	defer raw.Close()
	payload, _ = ioutil.ReadAll(raw)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(payload)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/foo-0.0.1.tgz").Return(httpResponse, nil)

	req, _ := http.NewRequest(http.MethodGet, "https://foo.bar/charts/foo-0.0.1.tgz", nil)

	httpMock.On("Do",
		req).Return(httpResponse, nil)

	httpMock.On("Get",
		"").Return(&http.Response{}, errors.New("no valid url"))

	return &clientMock, &httpMock
}

func GetRepoMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {

	clientMock := unstructuredmocks.K8SClientMock{}
	httpMock := mocks.HTTPClientMock{}

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "selector",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, []client.ListOption{client.InNamespace(""), client.MatchingLabels{
		"label": "notpresent",
	}}).Return(nil).Run(func(args mock.Arguments) {

		_ = args.Get(1).(*helmv1alpha1.ChartList)
	})

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "secret", Namespace: ""}, &v1.Secret{}).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(2).(*v1.Secret)
		c.ObjectMeta =
			metav1.ObjectMeta{
				Name: "secret",
			}
		c.Data = map[string][]byte{
			"user":     []byte("Zm9vCg=="),
			"password": []byte("ZW5jcnlwdGVkCg=="),
		}
	})

	/*expected :=  getExpectedTestCharts(clientMock)*/

	indexFile := testcases.GetTestRepoIndexFile()
	rawIndexFile, _ := json.Marshal(indexFile)
	httpResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpAuthResponse := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(rawIndexFile)),
	}

	httpMock.On("Get",
		"https://foo.bar/charts/index.yaml").Return(httpResponse, nil)

	req, _ := http.NewRequest(http.MethodGet, "https://foo.bar/charts/index.yaml", nil)

	httpMock.On("Do",
		req).Return(httpResponse, nil)

	reqAuth, _ := http.NewRequest(http.MethodGet, "https://bar.foo/charts/index.yaml", nil)
	reqAuth.SetBasicAuth("foo", "encrypted")

	httpMock.On("Do",
		reqAuth).Return(httpAuthResponse, nil)

	return &clientMock, &httpMock
}
