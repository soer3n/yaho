package helm

import (
	"github.com/soer3n/yaho/tests/mocks"
	unstructuredmocks "github.com/soer3n/yaho/tests/mocks/unstructured"
)

// GetChartMock returns kubernetes typed client mock and http client mock for testing chart functions
func GetChartMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {
	clientMock := &unstructuredmocks.K8SClientMock{}
	httpMock := &mocks.HTTPClientMock{}

	// testcase 1
	repo := repositoryMock{Name: "one", Namespace: "one", IsPresent: true, Labels: map[string]string{"repo": "one"}, URL: "https://foo.bar/charts"}

	charts := []chartMock{}
	versions := []chartVersionMock{}

	c := chartMock{Name: "one", Namespace: "one", Repository: "one", IsPresent: true, Labels: map[string]string{"repo": "one"}}
	cv := chartVersionMock{Chart: "one", Version: "0.0.1", Namespace: "one", IsPresent: true, URL: "https://foo.bar/charts/bar-0.0.1.tgz", Path: "../../../testutils/busybox-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)

	// testcase 2
	repo = repositoryMock{Name: "two", Namespace: "two", IsPresent: true, Labels: map[string]string{"repo": "two"}, URL: "https://foo.bar/charts"}

	charts = []chartMock{}
	versions = []chartVersionMock{}

	c = chartMock{Name: "two", Namespace: "two", Repository: "two", IsPresent: true, Labels: map[string]string{"repo": "two"}}
	dep := chartVersionMock{Chart: "testing-dep", Version: "0.1.0", Namespace: "two", IsPresent: true, URL: "https://foo.bar/charts/testing-dep-0.1.0.tgz", Path: "../../../testutils/busybox-0.1.0.tgz"}
	cv = chartVersionMock{Chart: "two", Version: "0.0.2", Namespace: "two", IsPresent: true, Dependencies: []chartVersionMock{dep}, URL: "https://foo.bar/charts/baz-0.0.2.tgz", Path: "../../../testutils/testing-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)

	// testcase 3
	repo = repositoryMock{Name: "three", Namespace: "three", IsPresent: true, Labels: map[string]string{"repo": "three"}, URL: "https://foo.bar/charts"}

	charts = []chartMock{}
	versions = []chartVersionMock{}

	c = chartMock{Name: "three", Namespace: "three", Repository: "three", IsPresent: false, Labels: map[string]string{"repo": "three"}}
	cv = chartVersionMock{Chart: "three", Version: "0.0.3", Namespace: "three", IsPresent: false, Dependencies: []chartVersionMock{}, URL: "https://foo.bar/charts/bar-0.0.3.tgz", Path: "../../../testutils/busybox-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)

	// testcase 4
	repo = repositoryMock{Name: "four", Namespace: "four", IsPresent: true, Labels: map[string]string{"repo": "four"}, URL: "https://foo.bar/charts"}

	charts = []chartMock{}
	versions = []chartVersionMock{}

	c = chartMock{Name: "four", Namespace: "four", Repository: "four", IsPresent: false, Labels: map[string]string{"repo": "four"}}
	dep = chartVersionMock{Chart: "testing-dep", Version: "0.1.0", Namespace: "four", IsPresent: false, URL: "https://foo.bar/charts/testing-dep-0.1.0.tgz", Path: "../../../testutils/testing-dep-0.1.1.tgz"}
	cv = chartVersionMock{Chart: "four", Version: "0.0.4", Namespace: "four", IsPresent: false, Dependencies: []chartVersionMock{dep}, URL: "https://foo.bar/charts/foo-0.0.4.tgz", Path: "../../../testutils/testing-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)

	// testcase 5
	repo = repositoryMock{Name: "five", Namespace: "five", IsPresent: true, Auth: &credentialsMock{User: "foo", Password: "bar"}, Labels: map[string]string{"repo": "five"}, URL: "https://bar.foo/charts"}

	charts = []chartMock{}
	versions = []chartVersionMock{}

	c = chartMock{Name: "five", Namespace: "five", Repository: "five", IsPresent: false, Labels: map[string]string{"repo": "five"}}
	dep = chartVersionMock{Chart: "testing-dep", Version: "0.1.0", Namespace: "five", IsPresent: false, Auth: &credentialsMock{User: "foo", Password: "bar"}, URL: "https://bar.foo/charts/testing-dep-0.1.0.tgz", Path: "../../../testutils/testing-dep-0.1.1.tgz"}
	cv = chartVersionMock{Chart: "five", Version: "0.0.5", Namespace: "five", IsPresent: false, Auth: &credentialsMock{User: "foo", Password: "bar"}, Dependencies: []chartVersionMock{dep}, URL: "https://bar.foo/charts/foo-0.0.5.tgz", Path: "../../../testutils/testing-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)

	return clientMock, httpMock
}

// GetValueMock returns kubernetes typed client mock and http client mock for testing values related functions
func GetValueMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {

	clientMock := &unstructuredmocks.K8SClientMock{}
	httpMock := &mocks.HTTPClientMock{}

	values := valueMock{Name: "foo", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: false, Releases: []string{"release"}}
	setValues(clientMock, httpMock, values)

	valuesSecond := valueMock{Name: "second", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: true, Releases: []string{"release"}}
	setValues(clientMock, httpMock, valuesSecond)

	valuesThird := valueMock{Name: "third", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: false, Releases: []string{"release"}}
	setValues(clientMock, httpMock, valuesThird)

	valuesFourth := valueMock{Name: "fourth", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: true, Releases: []string{"release"}, Refs: []valueRefMock{{
		Key:  "ref",
		Mock: valuesSecond,
	}}}
	setValues(clientMock, httpMock, valuesFourth)

	valuesFifth := valueMock{Name: "fifth", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: true, Releases: []string{"release"}, Refs: []valueRefMock{{
		Key:  "ref",
		Mock: valuesFourth,
	}}}
	setValues(clientMock, httpMock, valuesFifth)

	valuesSixth := valueMock{Name: "sixth", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: true, Releases: []string{"release"}, Refs: []valueRefMock{{
		Key:  "ref",
		Mock: valuesFifth,
	}}}
	setValues(clientMock, httpMock, valuesSixth)

	valuesSeventh := valueMock{Name: "seventh", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: true, Releases: []string{"release"}}
	setValues(clientMock, httpMock, valuesSeventh)

	valuesEigth := valueMock{Name: "eigth", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: true, Releases: []string{"release"}, Refs: []valueRefMock{{
		Key:  "ref2",
		Mock: valuesSeventh,
	}}}
	setValues(clientMock, httpMock, valuesEigth)

	return clientMock, httpMock
}

// GetReleaseMock returns kubernetes typed client mock and http client mock for testing release functions
func GetReleaseMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {
	clientMock := &unstructuredmocks.K8SClientMock{}
	httpMock := &mocks.HTTPClientMock{}

	// general setup
	repo := repositoryMock{Name: "repo", Namespace: "foo", IsPresent: true, Labels: map[string]string{"repo": "repo"}, URL: "https://foo.bar/charts"}

	charts := []chartMock{}
	versions := []chartVersionMock{}

	c := chartMock{Name: "chart", Namespace: "foo", Repository: "repo", IsPresent: true, Labels: map[string]string{"repo": "repo"}}
	cv := chartVersionMock{Chart: "chart", Version: "0.0.1", Namespace: "foo", IsPresent: true, URL: "https://foo.bar/charts/bar-0.0.1.tgz", Path: "../../../testutils/busybox-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)
	setConfig(clientMock, httpMock, "config", "foo", true)

	// testcase 1
	values := valueMock{Name: "notpresent", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: false, Releases: []string{"release"}}
	setValues(clientMock, httpMock, values)

	// testcase 2
	values = valueMock{Name: "present", Namespace: "foo", Values: map[string]interface{}{"foo": "bar", "boo": "baz"}, IsPresent: true, Releases: []string{"test"}}
	setValues(clientMock, httpMock, values)

	return clientMock, httpMock
}

// GetRepoMock returns kubernetes typed client mock and http client mock for testing repository functions
func GetRepoMock() (*unstructuredmocks.K8SClientMock, *mocks.HTTPClientMock) {
	clientMock := &unstructuredmocks.K8SClientMock{}
	httpMock := &mocks.HTTPClientMock{}

	// testcase 1
	repo := repositoryMock{Name: "one", Namespace: "one", IsPresent: false, Labels: map[string]string{"repo": "one"}, Auth: &credentialsMock{User: "foo", Password: "encrypted"}, URL: "https://foo.bar/charts"}

	charts := []chartMock{}
	versions := []chartVersionMock{}

	c := chartMock{Name: "foo", Namespace: "one", Repository: "one", IsPresent: true, Labels: map[string]string{"repo": "one"}}
	cv := chartVersionMock{Chart: "foo", Version: "0.0.1", Namespace: "one", IsPresent: true, URL: "https://foo.bar/charts/foo-0.0.1.tgz", Path: "../../../testutils/busybox-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)

	// testcase 2
	repo = repositoryMock{Name: "two", Namespace: "two", IsPresent: true, Labels: map[string]string{"repo": "two"}, URL: "https://bar.foo/charts"}

	charts = []chartMock{}
	versions = []chartVersionMock{}

	c = chartMock{Name: "bar", Namespace: "two", Repository: "two", IsPresent: true, Labels: map[string]string{"repo": "two"}}
	cv = chartVersionMock{Chart: "bar", Version: "0.0.2", Namespace: "two", IsPresent: true, URL: "https://bar.foo/charts/bar-0.0.2.tgz", Path: "../../../testutils/busybox-0.1.0.tgz"}
	versions = append(versions, cv)
	c.Versions = versions
	charts = append(charts, c)
	repo.Charts = charts

	setRepository(clientMock, httpMock, repo)

	return clientMock, httpMock
}
