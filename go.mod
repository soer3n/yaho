module github.com/soer3n/apps-operator

go 1.15

require (
	github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus/common v0.10.0
	github.com/soer3n/go-utils v0.0.0-20210109140919-45385a2c8124
	helm.sh/helm/v3 v3.4.2 // indirect
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	sigs.k8s.io/controller-runtime v0.7.0
)
