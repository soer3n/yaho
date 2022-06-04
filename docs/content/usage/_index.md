+++
title = "Usage"
weight = 30
chapter = true
+++

#### General usage

> There is a more complex configuration sample in [these](https://github.com/soer3n/apps-operator/blob/master/examples) directory. 

##### Preparation

All you need is a kubernetes cluster and a valid kubeconfig for deploying resources to it. You can also setup a kind cluster for testing.

```

# install kind cluster
kind create cluster --config testutils/kind.yaml --image kindest/node:v1.23.5

```

Now you can follow the [install instructions](/installation) to deploy the operator and rbac belonging to it. For installing releases you need to setup a service account with a cluster-role-binding or role-binding like this.

```
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: helm-releases
  namespace: helm

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: helm-releases-role
rules:
- apiGroups:
  - ''
  - 'apps'
  resources:
  - '*'
  verbs:
  - '*'

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: helm-releases-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: helm-releases-role
subjects:
- kind: ServiceAccount
  name: helm-releases
  namespace: helm

```

As a last step you need to setup a config resource. There you need to specify the name of the created service account. Installation and upgrade flags as well as allowed namespaces can also be configured there. The config needs to be in the same namespace as the release custom resource which should use it.

```

---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Config
metadata:
  name: helm-release-config
  namespace: helm
spec:
  serviceAccountName: helm-releases
  namespace:
    install: false
    allowed:
    - share
    - helm
  flags:
    atomic: false
    skipCRDs: false
    subNotes: true
    disableOpenAPIValidation: false
    dryRun: false
    disableHooks: false
    wait: false
    cleanupOnFail: false
    recreate: false
    timeout: 3600
    force: false
    description: "test description"

```

Now everything is ready for deploying your first repositories and charts in the next step.
