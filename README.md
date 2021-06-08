# Yet Another Helm Operator 

This operator is for managing helm repositories, releases and values in a declarative way. This project was originally created by the idea to deploy helm charts in a simple way without any binary except the kubernetes go-client. During the development more and more ideas came to my mind. The most aren't implemented until now. But this is exactly why i decided to publish this "private" project. When dozens of ideas came up when i'm thinking on it, it could be possible that i'm not the only one.


## Installation

Until now there is no docker image neither for the operator nor for the planned web backend. So you have to run it either local or you have to build an image and have to push it to your own account/repository. For both ways you need to install [golang](https://golang.org/doc/install) if not already done. Due to operator-sdk layout it's quite simple to do that.

```

# Install the CRDs
make install



# Building and pushing an image
export IMG="image_name:image_tag"
make docker-build docker-push

# Deploy the built operator
kubectl apply -f deploy/rbac.yaml
cat deploy/operator.yaml | envsubst | kubectl apply -f -

########
## OR ##
########

# Run it local simply with
make run

```


## Architecture

[Here](docs/ARCHITECTURE.md) is an explanation how the operator works and a comparison between the operator and helm usage on your workstation or somewhere else.


## Usage
