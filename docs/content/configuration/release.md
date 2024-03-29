+++
title = "Releases"
weight = 20
chapter = true
+++

> For configuring release installation and upgrade options like command flags, allowed namespaces and serviceAccountName which will be used for managing rendered resources there is a dedicated custom resource.

```
---
apiVersion: yaho.soer3n.dev/v1alpha1
kind: Config
metadata:
  name: example-config
  namespace: helm ### needs to be the same namespace for every release resource which should use this configuration
spec:
  serviceAccountName: account ### service account which will be used for configured releases for deploying resources
  namespace:
    install: false ### equal to `--install-namespace` flag
    allowed: ### configure a list of allowed namespaces for deploying releases
    - helm
    - share
  flags: ### keys are equal to install or upgrade flags
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
