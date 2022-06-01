+++
title = "Releases"
weight = 20
chapter = false
+++

After installing needed repository and chart resources you can create a helm release.

{{% notice info %}}
Needed chart resources have to be present before creating a release resource.
{{% /notice %}}

```

---
apiVersion: helm.soer3n.info/v1alpha1
kind: Release
metadata:
  name: test-release
  namespace: helm
spec:
  name: test-release
  namespace: share
  config: helm-release-config
  repo: test-repo
  chart: testing
  version: 0.1.1

```

{{% notice info %}}
The release will be installed either into spec.namespace or if this field is not set into object.metadata.namespace.
{{% /notice %}}

```bash

$ kubectl get releases.helm.soer3n.info -n helm 
NAME           GROUP   REPO        CHART     VERSION   SYNCED   STATUS    REVISION   AGE
test-release           test-repo   testing   0.1.1     true     success   1          4m4s

$ helm list -A
NAME            NAMESPACE       REVISION        UPDATED                                         STATUS          CHART           APP VERSION
test-release    share           1               2022-05-31 22:45:08.893512498 +0200 CEST        deployed        testing-0.1.1   1.16.0

$ kubectl get all -n share 
NAME                                        READY   STATUS    RESTARTS   AGE
pod/test-release-testing-859779cc95-9q8wp   1/1     Running   0          4m4s

NAME                           TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/test-release-testing   ClusterIP   10.102.215.45   <none>        80/TCP    4m4s

NAME                                   READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/test-release-testing   1/1     1            1           4m4s

NAME                                              DESIRED   CURRENT   READY   AGE
replicaset.apps/test-release-testing-859779cc95   1         1         1       4m4s

```

{{% notice info %}}
The spec will install a release only with charts default values configured.
{{% /notice %}}

```bash

$ helm get values -n share test-release 
USER-SUPPLIED VALUES:
null

```

If we want to configure chart values for a release we need to create a value resource and reference it in the release resource.

```

---
apiVersion: helm.soer3n.info/v1alpha1
kind: Values
metadata:
  name: test-values
  namespace: helm
spec:
  json:
    foo: bar

---
apiVersion: helm.soer3n.info/v1alpha1
kind: Release
metadata:
  name: test-release
  namespace: helm
spec:
  name: test-release
  namespace: share
  config: helm-release-config
  repo: test-repo
  chart: testing
  version: 0.1.1
  values:
  - test-values

```

{{% notice info %}}
The reference of values resource in release spec should configure release values.
{{% /notice %}}

```bash

$ kubectl get releases.helm.soer3n.info -n helm 
NAME           GROUP   REPO        CHART     VERSION   SYNCED   STATUS    REVISION   AGE
test-release           test-repo   testing   0.1.1     true     success   2          5m58s

$ helm list -A
NAME            NAMESPACE       REVISION        UPDATED                                         STATUS          CHART           APP VERSION
test-release    share           2               2022-05-31 22:50:33.366392593 +0200 CEST        deployed        testing-0.1.1   1.16.0

$ helm get values -n share test-release 
USER-SUPPLIED VALUES:
foo: bar

```

{{% notice info %}}
Values resources can be chained. Equal to the reference in a release spec you can do it in a values resource except that the key has to be specified for sub values.
{{% /notice %}}

Let's add another values resource and reference it in the first value resource.

```

---
apiVersion: helm.soer3n.info/v1alpha1
kind: Values
metadata:
  name: test-values-2
  namespace: helm
spec:
  json:
    test: it

---
apiVersion: helm.soer3n.info/v1alpha1
kind: Values
metadata:
  name: test-values
  namespace: helm
spec:
  json:
    foo: bar
  refs:
    ref: test-values-2

```

{{% notice info %}}
Release revision and values should be modified after applying new values resource and updating existing resource with added reference.
{{% /notice %}}

```bash

$ kubectl get releases.helm.soer3n.info -n helm 
NAME           GROUP   REPO        CHART     VERSION   SYNCED   STATUS    REVISION   AGE
test-release           test-repo   testing   0.1.1     true     success   3          13m

$ helm list -A
NAME            NAMESPACE       REVISION        UPDATED                                         STATUS          CHART           APP VERSION
test-release    share           3               2022-05-31 22:58:53.619041213 +0200 CEST        deployed        testing-0.1.1   1.16.0

$ helm get values -n share test-release 
USER-SUPPLIED VALUES:
foo: bar
ref:
  test: it

```