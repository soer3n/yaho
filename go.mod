module github.com/soer3n/apps-operator

go 1.15

require (
	github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus/common v0.10.0
	github.com/soer3n/go-utils v0.0.0-20210110173340-0e3296096656
	helm.sh/helm/v3 v3.4.2
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	k8s.io/helm v2.17.0+incompatible
	sigs.k8s.io/controller-runtime v0.7.0
)
