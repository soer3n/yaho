package helm

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	pointer "k8s.io/utils/ptr"

	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func setRepository(clientMock *unstructuredmocks.K8SClientMock, httpMock *mocks.HTTPClientMock, repositoryMock repositoryMock) {

	var err, aee error

	if !repositoryMock.IsPresent {
		err = k8serrors.NewNotFound(schema.GroupResource{
			Group:    "foo",
			Resource: "bar",
		}, "notfound")
	} else {
		aee = k8serrors.NewAlreadyExists(schema.GroupResource{
			Group:    "foo",
			Resource: "bar",
		}, "notfound")
	}

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: repositoryMock.Name}, &helmv1alpha1.Repository{}).Return(err).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Repository)
		c.ObjectMeta = metav1.ObjectMeta{
			Name: repositoryMock.Name,
		}

		if len(repositoryMock.Labels) > 0 {
			c.ObjectMeta.Labels = repositoryMock.Labels
		}

		c.Spec = helmv1alpha1.RepositorySpec{
			Name: repositoryMock.Name,
			URL:  repositoryMock.URL,
		}

		if repositoryMock.Auth != nil {
			c.Spec.AuthSecret = "secret"
		}
	})

	for _, cmock := range repositoryMock.Charts {
		v := make([]*repo.ChartVersion, 0)
		cm := &v1.ConfigMap{}

		indexFile := testcases.GetTestRepoIndexFile(cmock.Name)
		rawIndexFile, _ := json.Marshal(indexFile)
		rawVersions, _ := json.Marshal(indexFile.Entries[cmock.Name])

		for _, vmock := range cmock.Versions {
			i := &repo.ChartVersion{
				Metadata: &chart.Metadata{
					Name:       cmock.Name,
					Version:    vmock.Version,
					APIVersion: "v2",
				},
				URLs: []string{vmock.URL},
			}

			for _, ix := range vmock.Dependencies {
				d := &chart.Dependency{
					Name:       ix.Chart,
					Version:    ix.Version,
					Repository: cmock.Repository,
				}
				i.Metadata.Dependencies = append(i.Metadata.Dependencies, d)
			}

			v = append(v, i)
		}

		_, _ = json.Marshal(v)
		b := true
		ab := &b
		cm.BinaryData = map[string][]byte{
			"versions": rawVersions,
		}
		cm.ObjectMeta = metav1.ObjectMeta{
			Name:      "helm-" + cmock.Repository + "-" + cmock.Name + "-index",
			Namespace: cmock.Namespace,
			Labels: map[string]string{
				"yaho.soer3n.dev/chart": cmock.Name,
				"yaho.soer3n.dev/repo":  repositoryMock.Name,
				"yaho.soer3n.dev/type":  "index",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "yaho.soer3n.dev/v1alpha1",
					Kind:               "Repository",
					Name:               repositoryMock.Name,
					Controller:         pointer.To[bool](true),
					BlockOwnerDeletion: pointer.To[bool](true),
				},
			},
		}

		clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-" + cmock.Repository + "-" + cmock.Name + "-index", Namespace: cmock.Namespace}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {
			c := args.Get(2).(*v1.ConfigMap)
			c.ObjectMeta = cm.ObjectMeta
			c.BinaryData = cm.BinaryData

		})

		clientMock.On("Create", context.Background(), cm).Return(aee).Run(func(args mock.Arguments) {
			c := args.Get(1).(*v1.ConfigMap)
			c.ObjectMeta = cm.ObjectMeta
			c.BinaryData = cm.BinaryData

		})

		setChart(clientMock, httpMock, cmock, repositoryMock, ab)

		httpResponse := &http.Response{
			Body: io.NopCloser(bytes.NewReader(rawIndexFile)),
		}

		req, _ := http.NewRequest(http.MethodGet, repositoryMock.URL+"/index.yaml", nil)

		if repositoryMock.Auth != nil {
			req.SetBasicAuth(repositoryMock.Auth.User, repositoryMock.Auth.Password)

			clientMock.On("Get", context.Background(), types.NamespacedName{Name: "secret", Namespace: repositoryMock.Namespace}, &v1.Secret{}).Return(nil).Run(func(args mock.Arguments) {
				c := args.Get(2).(*v1.Secret)
				c.ObjectMeta = metav1.ObjectMeta{
					Name:      "secret",
					Namespace: repositoryMock.Namespace,
				}
				pwRaw := []byte(repositoryMock.Auth.Password)
				userRaw := []byte(repositoryMock.Auth.User)
				destPw := make([]byte, b64.StdEncoding.EncodedLen(len(pwRaw)))
				destUser := make([]byte, b64.StdEncoding.EncodedLen(len(userRaw)))
				b64.StdEncoding.Encode(destPw, pwRaw)
				b64.StdEncoding.Encode(destUser, userRaw)
				c.Data = map[string][]byte{
					"user":     destUser,
					"password": destPw,
				}
			})
		}

		httpMock.On("Do",
			req).Return(httpResponse, nil)
	}
}
