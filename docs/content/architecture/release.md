+++
title = "Releases"
weight = 30
+++

### Release

The release resource represents a helm release and is comparable to helm cli command "helm upgrade --install ...". The controller and its resource is work in progress like everything but in general it should map a release installation and/or upgrade process. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/release_types.go) for detailed information about the spec structure. There will be also 3 configmaps for chart templates, crds and default values created on the first installation which needs the requested chart in any version. You cannot define values directly in the release resource. This is solved by an own values resource which is explained a bit below.

![Alt text](/release.drawio.png?raw=true "Overview")

### ReleaseGroups

The releasegroup resource represents a collection of helm releases. The idea behind is to control releases which have dependencies to each other. At the moment it's just a collection without logic for managing them together. In general it deploys a collection of release resources. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/releasegroup_types.go) for detailed information about the spec structure.

![Alt text](/releasegroup.drawio.png?raw=true "Overview")
