+++
title = "Values"
weight = 40
+++

### Values

The values resource represents in general a values file for a release. There is some own logic there. The resource is splitted into two parts. The values and references to another values spec. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/values_types.go) for detailed information about the spec structure. The idea here is that these resources are managed like a construction kit for handling values for different releases. The main benefits are that you can stretch your values structure for a single release and that you can connect similar configurations for different releases. An example would be the definition of resource requests and limits.

![Alt text](/values.drawio.png?raw=true "Overview")
