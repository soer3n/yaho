## Overview

In general the procedure is nearly similar to well-known helm cli. At first you've to deploy a repository or repositories depending on the charts you want to use for your planned releases. After that you can use this prepared environment for deploying resources. It's just not local at your workstation but in your cluster. Sure there are some different behaviours when we come to the details. Also caused by the number of involved people.

![Alt text](img/overview.png?raw=true "Overview")


## Resources

### Repos

The repo resource represents an initialization of an helm repository. It is similar to helm cli command "helm repo add ..." and downloads the file for parsing charts which are part of requested repository. It is also parsing the chart resources. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/repo_types.go) for detailed information about the spec structure.

### RepoGroups

The repogroup resource represents a collection of helm repositories. This is needed if you want to deploy an helm release which has dependency charts which are part of different repositories. If dependencies are part of the same repository you don't need this. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/repogroup_types.go) for detailed information about the spec structure.

### Charts

The chart resource represents the meta information of an helm chart. It's nearly similar to [helm chart.Metadata](https://github.com/helm/helm/blob/main/pkg/chart/metadata.go#L43-L80). Additional there are information about chart dependencies and the url for getting content for deploying a release. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/chart_types.go) for detailed information about the spec structure.

### Releases

The release resource represents a helm release and is comparable to helm cli command "helm upgrade --install ...". The controller and its resource is work in progress like everything but in general it should map a release installation and/or upgrade process. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/release_types.go) for detailed information about the spec structure. There will be also 3 configmaps for chart templates, crds and default values created on the first installation which needs the requested chart in any version. You cannot define values directly in the release resource. This is solved by an own values resource which is explained a bit below.

### ReleaseGroups

The releasegroup resource represents a collection of helm releases. The idea behind is to control releases which have dependencies to each other. At the moment it's just a collection without logic for managing them together. In general it deploys a collection of release resources. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/releasegroup_types.go) for detailed information about the spec structure.

### Values

The values resource represents in general a values file for a release. There is some own logic there. The resource is splitted into two parts. The values and references to another values spec. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/values_types.go) for detailed information about the spec structure. The idea here is that these resources are managed like a construction kit for handling values for different releases. The main benefits are that you can stretch your values structure for a single release and that you can connect similar configurations for different releases. An example would be the definition of resource requests and limits.

## Comparsion to helm as binary
