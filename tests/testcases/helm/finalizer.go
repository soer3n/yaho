package helm

import (
	"flag"
	"io/ioutil"
	"testing"

	helmv1alpha1 "github.com/soer3n/yaho/apis/helm/v1alpha1"
	inttypes "github.com/soer3n/yaho/tests/mocks/types"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubefake "helm.sh/helm/v3/pkg/kube/fake"
)

var verbose = flag.Bool("test.log", false, "enable test logging")

func GetTestFinalizerRepo() *helmv1alpha1.Repo {
	return &helmv1alpha1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "",
		},
		Spec: helmv1alpha1.RepoSpec{
			Name: "repo",
			URL:  "https://foo.bar/charts",
		},
	}
}

func GetTestFinalizerRelease() *helmv1alpha1.Release {
	return &helmv1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "release",
			Namespace: "",
		},
		Spec: helmv1alpha1.ReleaseSpec{
			Name: "release", Repo: "repo",
			Chart: "chart",
		},
	}
}

func GetTestFinalizerFakeActionConfig(t *testing.T) *action.Configuration {
	return &action.Configuration{
		Releases:     storage.Init(driver.NewMemory()),
		KubeClient:   &kubefake.FailingKubeClient{PrintingKubeClient: kubefake.PrintingKubeClient{Out: ioutil.Discard}},
		Capabilities: chartutil.DefaultCapabilities,
		Log: func(format string, v ...interface{}) {
			t.Helper()
			if *verbose {
				t.Logf(format, v...)
			}
		},
	}
}

func GetTestFinalizerDeployedReleaseObj() *release.Release {
	return &release.Release{
		Name:  "release",
		Chart: GetTestHelmChart(),
		Info: &release.Info{
			Status: release.StatusDeployed,
		},
	}
}

func GetTestFinalizerIndexFile() *repo.IndexFile {
	return &repo.IndexFile{
		Entries: map[string]repo.ChartVersions{
			"doo": []*repo.ChartVersion{},
		},
	}
}

func GetTestFinalizerSpecsRelease() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnError: nil,
			ReturnValue: true,
			Input:       GetTestClientRelease(),
		},
		{
			ReturnError: nil,
			ReturnValue: true,
			Input:       GetTestClientRepo(),
		},
	}
}

func GetTestFinalizerSpecsRepo() []inttypes.TestCase {
	return []inttypes.TestCase{
		{
			ReturnError: nil,
			ReturnValue: true,
			Input:       GetTestClientRepo(),
		},
	}
}
