+++
title = "Charts"
weight = 20
+++

### Chart

The chart resource represents the meta information of an helm chart. It's nearly similar to [helm chart.Metadata](https://github.com/helm/helm/blob/main/pkg/chart/metadata.go#L43-L80). Additional there are information about chart dependencies and the url for getting content for deploying a release. See [here](https://github.com/soer3n/apps-operator/blob/master/apis/helm/v1alpha1/chart_types.go) for detailed information about the spec structure.

![Alt text](/chart.drawio.png?raw=true "Overview")
