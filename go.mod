module github.com/soer3n/yaho

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/containerd/containerd v1.4.11 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/gorilla/mux v1.7.3
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.26.0
	github.com/rogpeppe/go-internal v1.6.2 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/tools v0.1.6-0.20210802203754-9b21a8868e16 // indirect
	helm.sh/helm/v3 v3.6.1
	k8s.io/api v0.21.1
	k8s.io/apiextensions-apiserver v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v1.5.2
	k8s.io/kubectl v0.21.0
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/controller-runtime v0.8.2
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.12
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.3
	k8s.io/api => k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.21.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.21.0
	k8s.io/client-go => k8s.io/client-go v0.21.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.9.0
)
