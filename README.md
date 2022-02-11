[![codecov](https://codecov.io/gh/soer3n/yaho/branch/master/graph/badge.svg?token=DCPVNPSIFF)](https://codecov.io/gh/soer3n/yaho)
[![Go Report Card](https://goreportcard.com/badge/soer3n/yaho)](https://goreportcard.com/report/soer3n/yaho)

# Yet Another Helm Operator 

This operator is for managing helm repositories, releases and values in a declarative way. This project tries to picture helm as an kubernetes api extension. Through a custom resource for values reusing of them in different releases with same sub specifications is one feature. Another is to use kubernetes rbac for restricting helm usage for specific cluster configs. And there no local files which could differ from each other.


## Installation

```

kubectl apply -f https://raw.githubusercontent.com/soer3n/yaho/master/deploy/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/soer3n/yaho/master/deploy/operator.yaml


```

##  Local

If you want to run it local you need to install [golang](https://golang.org/doc/install) if not already done and [operator-sdk](https://sdk.operatorframework.io/docs/installation/).

```

# Install the CRDs
make install

# Run it local
make run

```

## Architecture

[Here](docs/ARCHITECTURE.md) is an explanation how the operator works and a comparison between the operator and helm usage on your workstation or somewhere else.

## Usage

[Click Here](docs/USAGE.md) for seeing an example.

## TODOs

- add assertions for tests; currently there are more or less only the normal cases covered
- handle func calls with context.Context if actually needed
- syncing state of releases continiously (check if there changes due to manual actions)
- switching to previous revision and back
- auto-sync for new chart versions from repository
- black- and whitelisting for charts when auto-sync for repository is enabled

## Contributing

Pull requests are welcome. As these project is in a very early stage there is currently no traditional contribution guideline due to the fact that actually every issue is a bigger change which can bring incompatibility on update processes of this operator.

But everyone can feel welcome to mention ideas and adding features which makes sense what could be actually everything what you can do with helm. More than view is needed for a proper further development.

## License
[LICENSE](LICENSE)
