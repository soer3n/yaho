# Yet Another Helm Operator 

This operator is for managing helm repositories, releases and values in a declarative way. This project was originally created by the idea to deploy helm charts in a simple way without any binary except the kubernetes go-client. During the development more and more ideas came to my mind. The most aren't implemented until now. But this is exactly why i decided to publish this "private" project. When dozens of ideas came up when i'm thinking on it, it could be possible that i'm not the only one.


## Installation

For now there is no docker image neither for the operator nor for the planned web backend. So you have to run it either local or you have to build an image and have to push it to your own account/repository. For both ways you need to install [golang](https://golang.org/doc/install) if not already done. Due to [operator-sdk](https://sdk.operatorframework.io/docs/installation/) layout it's quite simple to do that.

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

There are samples in [these](https://github.com/soer3n/apps-operator/blob/master/config/samples) directory. You should deploy all needed repositories before deploying a release or releasegroup. Kubectl can be used for filtering repos, charts and releases due to set labels by controllers.

```

# as an example if you deployed repo and repogroup resource from sample directory
# you see an output like this:

$ kubectl apply -f config/samples/helm_v1alpha1_repo.yaml
$ kubectl apply -f config/samples/helm_v1alpha1_repogroup.yaml

$ kubectl get repos.helm.soer3n.info -n helm

NAME         GROUP   CREATED_AT
bitnami      foo     2021-06-16T13:38:52Z
nextcloud    foo     2021-06-16T13:38:52Z
submariner           2021-06-16T13:38:57Z

# you can also filter by group label

$ kubectl get repos.helm.soer3n.info -n helm -l repoGroup=foo

NAME        GROUP   CREATED_AT
bitnami     foo     2021-06-16T13:38:52Z
nextcloud   foo     2021-06-16T13:38:52Z

# this is also possible for created charts:

$ kubectl get charts.helm.soer3n.info -n helm -l repo=submariner

NAME                    GROUP   REPO         CREATED_AT
submariner                      submariner   2021-06-16T13:39:05Z
submariner-k8s-broker           submariner   2021-06-16T13:39:06Z
submariner-operator             submariner   2021-06-16T13:39:06Z

$ kubectl get charts.helm.soer3n.info -n helm -l repoGroup=foo,repo=nextcloud

NAME        GROUP   REPO        CREATED_AT
nextcloud   foo     nextcloud   2021-06-16T13:38:52Z

# and also for releases:

$ kubectl apply -f config/samples/helm_v1alpha1_release.yaml
$ kubectl apply -n helm -f config/samples/helm_v1alpha1_release2.yaml


$ kubectl get releases.helm.soer3n.info -n helm -l chart=submariner-operator,repo=submariner

NAME              GROUP   REPO         CHART                 CREATED_AT
release-sample2           submariner   submariner-operator   2021-06-16T13:57:58Z

```



## Roadmap

- syncing state of releases from helm cli and other tools which are using the binary
- switching to previous revision and back
- translate cli flags to release spec
- auto-sync for new chart versions from repository
- black- and whitelisting for charts when auto-sync for repository is enabled
- loading charts from volume or git

## Known Issues / Troubleshooting

- charts with subfolders in templates are failing due to configmap rendering (slashes are not allowed as charactes in keys)
- non public repositories cannot be downloaded currently due to a replacement if integrated http client with the client delivered by "net/http" package

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
Please make sure to update tests as appropriate.


## License
[MIT](https://choosealicense.com/licenses/mit/)

