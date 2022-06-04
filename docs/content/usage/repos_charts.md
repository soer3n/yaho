+++
title = "Respositories & Charts"
weight = 10
chapter = false
+++

> There are two ways for managing repositories and charts. Either by configuring charts by repository or chart resource. 

&nbsp;

### manage by repository resource

&nbsp;

Let's deploy a basic repository resource at first.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Repository
metadata:
  name: test-repo
spec:
  name: test-repo
  url: https://soer3n.github.io/charts/testing_a
  charts: []

```

{{% notice info %}}
Nothing except the resource and indices for all charts found in downloaded index should be installed.
{{% /notice %}}

```bash

$ kubectl get repositories.yaho.soer3n.dev 
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     0        14s

$ kubectl get charts.yaho.soer3n.dev 
No resources found

$ kubectl get cm -n helm 
NAME                                  DATA   AGE
helm-test-repo-testing-index          1      7s
helm-test-repo-testing-nested-index   1      7s
kube-root-ca.crt                      1      9m25s

```

Let's add a chart without specified versions.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Repository
metadata:
  name: test-repo
spec:
  name: test-repo
  url: https://soer3n.github.io/charts/testing_a
  charts:
  - name: testing

```

{{% notice info %}}
Adding just a chart without specified versions should install the chart resource without any configuration except 'create dependency' parameter is set to true.
{{% /notice %}}

```bash

$ kubectl get repositories.yaho.soer3n.dev 
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     1        97s

$ kubectl get charts.yaho.soer3n.dev
NAME                GROUP   REPO        VERSIONS   DEPS     AGE
testing-test-repo           test-repo   synced     synced   4s

$ kubectl get cm -n helm 
NAME                                  DATA   AGE
helm-test-repo-testing-index          1      97s
helm-test-repo-testing-nested-index   1      97s
kube-root-ca.crt                      1      10m

```

Let's add a version to the specified chart.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Repository
metadata:
  name: test-repo
spec:
  name: test-repo
  url: https://soer3n.github.io/charts/testing_a
  charts:
  - name: testing
    versions:
    - 0.1.1

```

{{% notice info %}}
Adding a version to specified chart should add configmaps with related information for usage of it.
{{% /notice %}}

```bash

$ kubectl get repositories.yaho.soer3n.dev 
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     1        7m

$ kubectl get charts.yaho.soer3n.dev
NAME                GROUP   REPO        VERSIONS   DEPS     AGE
testing-test-repo           test-repo   synced     synced   5m

$ kubectl get cm -n helm 
NAME                                   DATA   AGE
helm-crds-test-repo-testing-0.1.1      0      110s
helm-default-test-repo-testing-0.1.1   1      110s
helm-test-repo-testing-index           1      70s
helm-test-repo-testing-nested-index    1      70s
helm-tmpl-test-repo-testing-0.1.1      5      110s
kube-root-ca.crt                       1      16m

```

And add another version.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Repository
metadata:
  name: test-repo
spec:
  name: test-repo
  url: https://soer3n.github.io/charts/testing_a
  charts:
  - name: testing
    versions:
    - 0.1.0
    - 0.1.1

```

{{% notice info %}}
This should create configmaps related to the second version of the chart.
{{% /notice %}}

```bash

$ kubectl get repositories.yaho.soer3n.dev 
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     1        2m

$ kubectl get charts.yaho.soer3n.dev
NAME                    GROUP   REPO        VERSIONS   DEPS     AGE
testing-test-repo               test-repo   synced     synced   1m40s

$ kubectl get cm -n helm 
NAME                                   DATA   AGE
helm-crds-test-repo-testing-0.1.0      0      30s
helm-crds-test-repo-testing-0.1.1      0      50s
helm-default-test-repo-testing-0.1.0   1      30s
helm-default-test-repo-testing-0.1.1   1      50s
helm-test-repo-testing-index           1      56s
helm-test-repo-testing-nested-index    1      56s
helm-tmpl-test-repo-testing-0.1.0      5      30s
helm-tmpl-test-repo-testing-0.1.1      5      50s
kube-root-ca.crt                       1      19m

```

&nbsp;

### manage by chart resource

&nbsp;

Again let's deploy a basic repository resource at first.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Repository
metadata:
  name: test-repo
spec:
  name: test-repo
  url: https://soer3n.github.io/charts/testing_a
  charts: []

```

{{% notice info %}}
Nothing except the resource and indices for all charts found in downloaded index should be installed.
{{% /notice %}}

```bash

$ kubectl get repositories.yaho.soer3n.dev 
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     0        14s

$ kubectl get charts.yaho.soer3n.dev 
No resources found

$ kubectl get cm -n helm 
NAME                                  DATA   AGE
helm-test-repo-testing-index          1      14s
helm-test-repo-testing-nested-index   1      14s
kube-root-ca.crt                      1      9m25s

```

Let's create a chart resource without specified versions.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Chart
metadata:
  name: test-chart
spec:
  name: testing
  repo: test-repo
  versions: []
  createDeps: false

```

{{% notice info %}}
Creating just a chart resource without specified versions should install nothing else than the resource with values configured in spec.
{{% /notice %}}

```bash

$ kubectl get repositories.yaho.soer3n.dev 
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     1        26s

$ kubectl get charts.yaho.soer3n.dev 
NAME         GROUP   REPO        VERSIONS   DEPS   AGE
test-chart           test-repo                     26s

$ kubectl get cm -n helm 
NAME                                  DATA   AGE
helm-test-repo-testing-index          1      26s
helm-test-repo-testing-nested-index   1      26s
kube-root-ca.crt                      1      26m

```
Let's add a version to the chart resource.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Chart
metadata:
  name: test-chart
spec:
  name: testing
  repository: test-repo
  versions:
  - 0.1.1
  createDeps: false

```

{{% notice info %}}
Adding a version to chart resource should add configmaps with related information for usage of it.
{{% /notice %}}

```bash

NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     1        5m53s

$ kubectl get charts.yaho.soer3n.dev 
NAME         GROUP   REPO        VERSIONS   DEPS        AGE
test-chart           test-repo   synced     synced   7m57s

$ kubectl get cm -n helm 
NAME                                   DATA   AGE
helm-crds-test-repo-testing-0.1.1      0      21s
helm-default-test-repo-testing-0.1.1   1      21s
helm-test-repo-testing-index           1      8m22s
helm-test-repo-testing-nested-index    1      8m22s
helm-tmpl-test-repo-testing-0.1.1      5      21s
kube-root-ca.crt                       1      41m

```

&nbsp;

### dependencies

&nbsp;

Again let's deploy a basic repository resource at first.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Repository
metadata:
  name: test-repo
spec:
  name: test-repo
  url: https://soer3n.github.io/charts/testing_a
  charts: []

```

{{% notice info %}}
Nothing except the resource and indices for all charts found in downloaded index should be installed.
{{% /notice %}}

```bash

$ kubectl get repositories.yaho.soer3n.dev 
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     0        14s

$ kubectl get charts.yaho.soer3n.dev 
No resources found

$ kubectl get cm -n helm 
NAME                                  DATA   AGE
helm-test-repo-testing-index          1      14s
helm-test-repo-testing-dep-index      1      14s
helm-test-repo-testing-nested-index   1      14s
kube-root-ca.crt                      1      9m25s

```

Let's create a chart resource with a specified version and dependency installation disabled.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Chart
metadata:
  name: test-chart
spec:
  name: testing-dep
  repository: test-repo
  versions:
  - 0.1.0
  createDeps: false

```

{{% notice info %}}
Adding a chart resource with specified version should add configmaps with related information for usage of it.
{{% /notice %}}

```bash

NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     1        8m

$ kubectl get charts.yaho.soer3n.dev 
NAME              GROUP   REPO        VERSIONS   DEPS        AGE
test-chart                test-repo   synced     synced      7m

$ kubectl get cm -n helm 
NAME                                      DATA   AGE
helm-crds-test-repo-testing-dep-0.1.0     0      7m
helm-default-test-repo-testing-dep-0.1.0  1      7m
helm-test-repo-testing-index              1      7m
helm-test-repo-testing-dep-index          1      7m
helm-test-repo-testing-nested-index       1      7m
helm-tmpl-test-repo-testing-dep-0.1.0     5      7m
kube-root-ca.crt                          1      41m

```

Let's enable dependency installation for the chart resource.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Chart
metadata:
  name: test-chart
spec:
  name: testing-dep
  repository: test-repo
  versions:
  - 0.1.0
  createDeps: true

```

{{% notice info %}}
Enable dependency installation for the chart resource should add configmaps with related information for usage of dependency charts.
{{% /notice %}}

```bash

NAME        GROUP   SYNCED   CHARTS   AGE
test-repo           true     2        8m

$ kubectl get charts.yaho.soer3n.dev 
NAME              GROUP   REPO        VERSIONS   DEPS        AGE
test-chart                test-repo   synced     synced      7m
testing-test-repo         test-repo   synced     synced      1m

$ kubectl get cm -n helm 
NAME                                      DATA   AGE
helm-crds-test-repo-testing-dep-0.1.0     0      7m
helm-crds-test-repo-testing-0.1.1         0      1m
helm-default-test-repo-testing-dep-0.1.0  1      7m
helm-default-test-repo-testing-0.1.1      1      1m
helm-test-repo-testing-index              1      7m
helm-test-repo-testing-dep-index          1      7m
helm-test-repo-testing-nested-index       1      7m
helm-tmpl-test-repo-testing-dep-0.1.0     5      7m
helm-tmpl-test-repo-testing-0.1.1         5      1m
kube-root-ca.crt                          1      41m

```

{{% notice note %}}
Currently charts can only find dependency charts when they are in the same repository or both repositories are in the same repogroup which means that the repositories share the label value "yaho.soer3n.dev/repoGroup"
{{% /notice %}}

&nbsp;

### repository groups

Now let's deploy a repository group where only one chart is specified. 

{{% notice info %}}
Remember that the 'create dependencies' option is automatically set to true. The specified chart has a chart from the first repository as a dependency.
{{% /notice %}}

```

apiVersion: yaho.soer3n.dev/v1alpha1
kind: RepoGroup
metadata:
  name: repogroup-sample
spec:
  labelSelector: foo
  repos:
    - name: test-repo-a
      url: https://soer3n.github.io/charts/testing_a
    - name: test-repo-b
      url: https://soer3n.github.io/charts/testing_b
      charts:
      - name: testing-dep
        versions:
        - 0.1.1

```

{{% notice info %}}
After applying the repository group resource there are two charts with related configmaps are deployed. The specified and its dependency.
{{% /notice %}}

```bash

$ kubectl get repogroups.yaho.soer3n.dev
NAME             AGE
repogroup-sample 8m

$ kubectl get repositories.yaho.soer3n.dev
NAME        GROUP   SYNCED   CHARTS   AGE
test-repo-a foo     true     1        8m
test-repo-b foo     true     1        8m

$ kubectl get charts.yaho.soer3n.dev 
NAME                    GROUP   REPO        VERSIONS   DEPS        AGE
testing-test-repo-a     foo     test-repo-a synced     synced      8m
testing-dep-test-repo-b foo     test-repo-b synced     synced      8m

$ kubectl get cm -n helm 
NAME                                        DATA   AGE
helm-crds-test-repo-b-testing-dep-0.1.1     0      8m
helm-crds-test-repo-a-testing-0.1.1         0      8m
helm-default-test-repo-b-testing-dep-0.1.0  1      8m
helm-default-test-repo-a-testing-0.1.1      1      8m
helm-test-repo-a-testing-index              1      8m
helm-test-repo-a-testing-dep-index          1      8m
helm-test-repo-a-testing-nested-index       1      8m
helm-test-repo-b-testing-dep-index          1      8m
helm-tmpl-test-b-repo-testing-dep-0.1.1     5      8m
helm-tmpl-test-a-repo-testing-0.1.1         5      8m
kube-root-ca.crt                            1      41m

```

&nbsp;

### filter by labels

The custom resources and related configmaps can be filtered by labels.

| Label                      | Resource                       | Value                                   |
|----------------------------|--------------------------------|-----------------------------------------|
| yaho.soer3n.dev/chart     | configmaps, chart              | chart name                              |
| yaho.soer3n.dev/repo      | configmaps, repository, chart  | repo name                               |
| yaho.soer3n.dev/type      | configmaps                     | index,tmpl,default,crds                 |
| yaho.soer3n.dev/repoGroup | repository, chart              | repo group name                         |
| yaho.soer3n.dev/unmanaged | chart                          | if chart is managed by another resource |