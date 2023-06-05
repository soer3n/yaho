package helm

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func setChartVersion(clientMock *unstructuredmocks.K8SClientMock, httpMock *mocks.HTTPClientMock, chartVersionMock chartVersionMock, repo repositoryMock) {

	var e error
	op := "Update"

	if !chartVersionMock.IsPresent {
		e = k8serrors.NewNotFound(schema.GroupResource{
			Group:    "foo",
			Resource: "bar",
		}, "notfound")
		op = "Create"
	}

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-default-" + repo.Name + "-" + chartVersionMock.Chart + "-" + chartVersionMock.Version, Namespace: chartVersionMock.Namespace}, &v1.ConfigMap{}).Return(e).Run(func(args mock.Arguments) {
		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseDefaultValueConfigMap()
		c.Data = spec.Data
		c.ObjectMeta.Name = "helm-default-" + repo.Name + "-" + chartVersionMock.Chart + "-" + chartVersionMock.Version
		c.ObjectMeta.Namespace = chartVersionMock.Namespace
		// c.ObjectMeta.Labels = map[string]string{"yaho.soer3n.dev/chart": chartVersionMock.Chart + "-" + chartVersionMock.Version, "yaho.soer3n.dev/repo": repo, "yaho.soer3n.dev/type": "default"}
	})

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-tmpl-" + repo.Name + "-" + chartVersionMock.Chart + "-" + chartVersionMock.Version, Namespace: chartVersionMock.Namespace}, &v1.ConfigMap{}).Return(e).Run(func(args mock.Arguments) {
		c := args.Get(2).(*v1.ConfigMap)
		spec := testcases.GetTestReleaseTemplateConfigMap()
		c.BinaryData = spec.BinaryData
		c.ObjectMeta.Name = "helm-tmpl-" + repo.Name + "-" + chartVersionMock.Chart + "-" + chartVersionMock.Version
		c.ObjectMeta.Namespace = chartVersionMock.Namespace
		// c.ObjectMeta.Labels = map[string]string{"yaho.soer3n.dev/chart": chartVersionMock.Chart + "-" + chartVersionMock.Version, "yaho.soer3n.dev/repo": repo, "yaho.soer3n.dev/type": "tmpl"}
	})

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-crds-" + repo.Name + "-" + chartVersionMock.Chart + "-" + chartVersionMock.Version, Namespace: chartVersionMock.Namespace}, &v1.ConfigMap{}).Return(e).Run(func(args mock.Arguments) {
		c := args.Get(2).(*v1.ConfigMap)
		c.BinaryData = make(map[string][]byte)
		c.ObjectMeta.Name = "helm-crds-" + repo.Name + "-" + chartVersionMock.Chart + "-" + chartVersionMock.Version
		c.ObjectMeta.Namespace = chartVersionMock.Namespace
		// c.ObjectMeta.Labels = map[string]string{"yaho.soer3n.dev/chart": chartVersionMock.Chart + "-" + chartVersionMock.Version, "yaho.soer3n.dev/repo": repo, "yaho.soer3n.dev/type": "tmpl"}
	})

	clientMock.On(op, context.Background(), mock.MatchedBy(func(cm *v1.ConfigMap) bool {

		/*
			&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-default-" + chartVersionMock.Chart + "-" + chartVersionMock.Version,
					Namespace: chartVersionMock.Namespace,
					Labels:    map[string]string{"yaho.soer3n.dev/chart": chartVersionMock.Chart + "-" + chartVersionMock.Version, "yaho.soer3n.dev/repo": repo, "yaho.soer3n.dev/type": "default"},
				},
				Data: testcases.GetTestReleaseDefaultValueConfigMap().Data,
			}
		*/

		return true
	})).Return(e)

	clientMock.On("List", context.Background(), &v1.ConfigMapList{}, mock.MatchedBy(func(cList []client.ListOption) bool {

		opt := cList[0].(*client.ListOptions)

		if opt.LabelSelector != nil {
			res := opt.LabelSelector.String() == "yaho.soer3n.dev/chart="+chartVersionMock.Chart+"-"+chartVersionMock.Version+","+"yaho.soer3n.dev/type=tmpl"

			if !res {
				res = opt.LabelSelector.String() == "yaho.soer3n.dev/chart="+chartVersionMock.Chart+"-"+chartVersionMock.Version+","+"yaho.soer3n.dev/type=crds"
			}
			return res
		}
		return true
	})).Return(nil).Run(func(args mock.Arguments) {

		c := args.Get(1).(*v1.ConfigMapList)
		foo := map[string]string{"foo": "bar", "boo": "baz"}
		bar := map[string]string{"foo": "bar", "boo": "baz"}

		fooData, _ := yaml.Marshal(&foo)
		barData, _ := yaml.Marshal(&bar)

		c.Items = []v1.ConfigMap{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "helm-tmpl-" + repo.Name + "-" + chartVersionMock.Chart + "-" + chartVersionMock.Version,
					Namespace: chartVersionMock.Namespace,
				},
				BinaryData: map[string][]byte{
					"foo.yaml": fooData,
					"bar.yaml": barData,
				},
			},
		}
	})

	var payload []byte

	raw, _ := os.Open(chartVersionMock.Path)

	defer func() {
		if err := raw.Close(); err != nil {
			log.Printf("Error closing file: %s\n", err)
		}
	}()

	payload, _ = io.ReadAll(raw)
	httpResponse := &http.Response{
		Body: io.NopCloser(bytes.NewReader(payload)),
	}

	httpMock.On("Get",
		repo.URL+"/"+chartVersionMock.Chart+"-"+chartVersionMock.Version+".tgz").Return(httpResponse, nil)

	req, _ := http.NewRequest(http.MethodGet, repo.URL+"/"+chartVersionMock.Chart+"-"+chartVersionMock.Version+".tgz", nil)

	if chartVersionMock.Auth != nil {
		req.SetBasicAuth(chartVersionMock.Auth.User, chartVersionMock.Auth.Password)
	}

	httpMock.On("Do",
		req).Return(httpResponse, nil)
}
