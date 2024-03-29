+++
title = "Development"
weight = 25
chapter = false
+++

#### Local

To start the operator locally there are only two commands nessecarry.

```

# generating crds if needed (when something changed in 'apis' folder)
make generate

# installing crds
make install

# starting the operator
WATCH_NAMESPACE=helm make run

```

#### Testing

Tests are splitted into two kinds. Unit tests implementent with the builtin testing and ginkgo framework package and integration tests build with operator-sdk and kubebuilder.

##### Unit

```

# running all unit tests
go test -v ./tests/unit/... 

# running only filtered unit tests (e.g. run only tests related to values model)
go test -v ./tests/unit/... -test.run TestValues

```

##### Integration

```

# running integration tests
go test -v ./tests/e2e/...

```

{{% notice info %}}
To run only specific tests you need to focus them with an 'F' before 'It' keyword.
{{% /notice %}}

```

# ./tests/e2e/controller/..._test.go

...
var _ = Context("...", func() {

    ...

	Describe("...", func() {

        ...
		FIt(...

```

#### Debugging

When you are using vscode there is an launch.json already present in the project. It can be used for everything needed. There are three configurations. For debugging integration and unit tests and also for the operator. Environment variables can be set as usual in vscode.

{{% notice note %}}
If you want to debug a failed integration test you only need to change the WATCH_NAMESPACE variable in launch.json to the namespace which was generated while running the failed test.
{{% /notice %}}

#### Remote

You can use every tool for golang remote development. There are no special things to care about. If you choose for example okteto, only a normal configuration file is needed. For example like this:

```
name: yaho
namespace: dev
autocreate: true
image: okteto/golang:1.18
command: bash
securityContext:
  capabilities:
    add:
    - SYS_PTRACE
volumes:
- /go/pkg/
- /root/.cache/go-build/
sync:
- .:/usr/src/app
environment:
  PATH: $PATH:/usr/local/go/bin:/go/bin
persistentVolume:
  enabled: true
  storageClass: local-path-storage
  size: 1Gi
resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "1024Mi"
    cpu: "500m"
initContainer:
  resources:
    requests:
      cpu: 125m
      memory: 128Mi
    limits:
      cpu: 250m
      memory: 256Mi
interface: 0.0.0.0

```