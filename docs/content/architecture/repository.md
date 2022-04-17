+++
title = "Repositories"
weight = 10
+++

### Repository

The repo resource represents an initialization of an helm repository. It is similar to helm cli command "helm repo add ..." and downloads the file for parsing charts which are part of requested repository. It is also parsing the chart resources. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/repo_types.go) for detailed information about the spec structure.

![Alt text](/repository.drawio.png?raw=true "Overview")

### RepoGroups

The repogroup resource represents a collection of helm repositories. This is needed if you want to deploy an helm release which has dependency charts which are part of different repositories. If dependencies are part of the same repository you don't need this. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/repogroup_types.go) for detailed information about the spec structure.

![Alt text](/repogroup.drawio.png?raw=true "Overview")
