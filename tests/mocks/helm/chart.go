package helm

import (
	"context"
	"encoding/json"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	// "k8s.io/utils/pointer"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
	testcases "github.com/soer3n/yaho/tests/testcases/helm"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func setChart(clientMock *unstructuredmocks.K8SClientMock, httpMock *mocks.HTTPClientMock, chartMock chartMock, repoMock repositoryMock, b *bool) {

	var e error
	var ce error

	if !chartMock.IsPresent {
		e = k8serrors.NewNotFound(schema.GroupResource{
			Group:    "foo",
			Resource: "bar",
		}, "notfound")
	} else {
		ce = k8serrors.NewAlreadyExists(schema.GroupResource{
			Group:    "foo",
			Resource: "bar",
		}, "notfound")
	}

	clientMock.On("Create", context.Background(), mock.MatchedBy(func(c *helmv1alpha1.Chart) bool {
		return c.ObjectMeta.Name == chartMock.Name && c.ObjectMeta.Namespace == chartMock.Namespace
	})).Return(ce)

	clientMock.On("Update", context.Background(), mock.MatchedBy(func(c *helmv1alpha1.Chart) bool {
		return c.ObjectMeta.Name == chartMock.Name && c.ObjectMeta.Namespace == chartMock.Namespace
	})).Return(e)

	/*
		&helmv1alpha1.Chart{
			ObjectMeta: metav1.ObjectMeta{
				Name:      chartMock.Name,
				Namespace: chartMock.Namespace,
				Labels:    chartMock.Labels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "helm.soer3n.info/v1alpha1",
						Kind:               "Repository",
						Name:               chartMock.Repository,
						Controller:         pointer.BoolPtr(true),
						BlockOwnerDeletion: pointer.BoolPtr(true),
					},
				},
			},
			Spec: helmv1alpha1.ChartSpec{
				Name:       chartMock.Name,
				Repository: chartMock.Repository,
				CreateDeps: true,
				Versions:   []string{},
			},
		},*/

	clientMock.On("Get", context.Background(), types.NamespacedName{Name: chartMock.Name, Namespace: chartMock.Namespace}, &helmv1alpha1.Chart{}).Return(e).Run(func(args mock.Arguments) {
		c := args.Get(2).(*helmv1alpha1.Chart)
		v := []string{}

		for _, e := range chartMock.Versions {
			v = append(v, e.Version)
		}

		spec := helmv1alpha1.Chart{
			ObjectMeta: metav1.ObjectMeta{
				Name:      chartMock.Name,
				Namespace: chartMock.Namespace,
				Labels:    chartMock.Labels,
			},
			Spec: helmv1alpha1.ChartSpec{
				Name:       chartMock.Name,
				Repository: chartMock.Repository,
				Versions:   []string{},
			},
		}

		if chartMock.IsPresent {
			spec.Spec.Versions = v
		}

		c.Spec = spec.Spec
		c.ObjectMeta = spec.ObjectMeta
	})

	cl := &helmv1alpha1.ChartList{}

	for _, v := range chartMock.Versions {
		setChartVersion(clientMock, httpMock, v, repoMock)

		cl.Items = append(cl.Items, helmv1alpha1.Chart{
			ObjectMeta: metav1.ObjectMeta{
				Name: v.Chart, Namespace: v.Namespace,
			},
			Spec: helmv1alpha1.ChartSpec{
				Name:       v.Chart,
				Repository: chartMock.Repository,
				Versions: []string{
					v.Version,
				},
			},
		})

		for _, d := range v.Dependencies {

			clientMock.On("Get", context.Background(), types.NamespacedName{Name: "helm-" + chartMock.Repository + "-" + d.Chart + "-index", Namespace: d.Namespace}, &v1.ConfigMap{}).Return(nil).Run(func(args mock.Arguments) {
				c := args.Get(2).(*v1.ConfigMap)
				v := make([]*repo.ChartVersion, 0)
				i := &repo.ChartVersion{
					Metadata: &chart.Metadata{
						Name:       d.Chart,
						Version:    d.Version,
						APIVersion: "v2",
					},
					URLs: []string{"https://foo.bar/charts"},
				}

				v = append(v, i)

				b, _ := json.Marshal(v)
				c.BinaryData = map[string][]byte{
					"versions": b,
				}
				c.ObjectMeta = metav1.ObjectMeta{
					Name:      d.Chart,
					Namespace: d.Namespace,
				}

				for ix, iv := range chartMock.Labels {
					if c.ObjectMeta.Labels == nil {
						c.ObjectMeta.Labels = map[string]string{}
					}

					c.ObjectMeta.Labels[ix] = iv
				}

			})

			it := helmv1alpha1.Chart{
				ObjectMeta: metav1.ObjectMeta{
					Name: d.Chart, Namespace: v.Namespace,
				},
				Spec: helmv1alpha1.ChartSpec{
					Name:       d.Chart,
					Repository: chartMock.Repository,
					Versions:   []string{},
				},
			}

			if d.IsPresent {
				it.Spec.Versions = []string{d.Version}
			}

			cl.Items = append(cl.Items, it)

			var e error
			op := "Update"

			if !d.IsPresent {
				e = k8serrors.NewNotFound(schema.GroupResource{
					Group:    "foo",
					Resource: "bar",
				}, "notfound")
				op = "Create"
			}

			clientMock.On("Get", context.Background(), types.NamespacedName{Name: d.Chart, Namespace: d.Namespace}, &helmv1alpha1.Chart{}).Return(e).Run(func(args mock.Arguments) {
				c := args.Get(2).(*helmv1alpha1.Chart)

				spec := helmv1alpha1.Chart{
					ObjectMeta: metav1.ObjectMeta{
						Name:      d.Chart,
						Namespace: d.Namespace,
					},
					Spec: helmv1alpha1.ChartSpec{
						Name:       d.Chart,
						Repository: chartMock.Repository,
						Versions:   []string{},
					},
				}

				if d.IsPresent {
					spec.Spec.Versions = []string{d.Version}
				}

				c.Spec = spec.Spec
				c.ObjectMeta = spec.ObjectMeta
			})

			clientMock.On(op, context.Background(), mock.MatchedBy(func(c *helmv1alpha1.Chart) bool {
				return d.Chart == c.Name && d.Namespace == c.Namespace
			})).Return(nil)

			setChartVersion(clientMock, httpMock, d, repoMock)
		}
	}

	clientMock.On("List", context.Background(), &helmv1alpha1.ChartList{}, mock.MatchedBy(func(cList []client.ListOption) bool {

		opt := cList[0].(*client.ListOptions)

		if opt.LabelSelector != nil {
			if chartMock.Group != nil {
				return opt.LabelSelector.String() == "repoGroup="+*chartMock.Group
			}

			if opt.Namespace == chartMock.Namespace {
				return true
			}
		}

		return false
	})).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(1).(*helmv1alpha1.ChartList)
		c.Items = cl.Items
	})

	clientMock.On("List", context.Background(), &helmv1alpha1.RepositoryList{}, mock.MatchedBy(func(cList []client.ListOption) bool {

		opt := cList[0].(*client.ListOptions)

		if opt.LabelSelector != nil {
			if chartMock.Group != nil {
				return opt.LabelSelector.String() == "repoGroup="+*chartMock.Group
			}
		}
		return true
	})).Return(nil).Run(func(args mock.Arguments) {
		c := args.Get(1).(*helmv1alpha1.RepositoryList)
		spec := testcases.GetTestRepoRepoListSpec()
		c.Items = spec.Items
	})
}
