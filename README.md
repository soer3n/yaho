[![codecov](https://codecov.io/gh/soer3n/yaho/branch/master/graph/badge.svg?token=DCPVNPSIFF)](https://codecov.io/gh/soer3n/yaho)

# Yet Another Helm Operator 

This operator is for managing helm repositories, releases and values in a declarative way. This project was originally created by the idea to deploy helm charts in a simple way without any binary except the kubernetes go-client, to avoid problem caused by local dependencies (e.g. missing repo pull, usage of wrong repository, not synced values for a release, ...), reusing of values in different releases with same sub specifications and to learn how helm and golang actually works. During the development more and more ideas came to my mind. The most aren't implemented until now. But this huge number of ideas brought me to publishing this project. 


## Installation

For now there is no docker image neither for the operator nor for the planned web backend. So you have to run it either local or you have to build an image and push it to your own account/repository. For the second way only docker is needed. If you want to run it local you need to install [golang](https://golang.org/doc/install) if not already done and [operator-sdk](https://sdk.operatorframework.io/docs/installation/).

```

# Install the CRDs
make install


# Building and pushing as an image to private registry
export IMG="image_name:image_tag"
make docker-build docker-push

# create image pull secret if needed
kubectl create secret generic harbor-registry-secret -n helm --from-file=.dockerconfigjson=harbor.json --type=kubernetes.io/dockerconfigjson

# Deploy the built operator
kubectl apply -f deploy/rbac.yaml
cat deploy/operator.yaml | envsubst | kubectl apply -f -

########
## OR ##
########

# Run it local
make run

```


## Architecture

[Here](docs/ARCHITECTURE.md) is an explanation how the operator works and a comparison between the operator and helm usage on your workstation or somewhere else.


## Usage

There are samples in [these](config/samples) directory. You should deploy all needed repositories before deploying a release or releasegroup. Kubectl can be used for filtering repos, charts and releases due to set labels by controllers.

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

- add assertions for tests; currently there are more or less only the normal cases covered by qa
- running e2e tests with kind in kubernetes( [related issue](https://github.com/actions-runner-controller/actions-runner-controller/issues/640))
- handle func calls with context.Context if actually needed
- evaluate where to use concurrency makes sense
- add contribution guideline
- implement web user interface with backend (the [frontend skeleton](web/) and start of [backend server implementation](pkg/api/) is already present)
- syncing state of releases from helm cli and other tools which are using the binary
- switching to previous revision and back
- translate cli flags to release spec
- auto-sync for new chart versions from repository
- black- and whitelisting for charts when auto-sync for repository is enabled
- loading charts from volume or git

## Known Issues / Troubleshooting

- charts with subfolders in templates are failing due to configmap rendering (slashes are not allowed as charactes in keys)
- non public repositories cannot be downloaded currently due to a replacement if integrated http client with the client delivered by "net/http" package
- fix infinite reconciling in e2e tests (tests with release resource deployed)
- fix local e2e test runs (currently there is a fix needed due to limitations of envtest; [garbage collection of owned resources is not working due to missing kubelet](https://book.kubebuilder.io/reference/envtest.html#testing-considerations) and a [caching problem related to go-client](https://github.com/kubernetes-sigs/controller-runtime/issues/343))

## Contributing

Pull requests are welcome. As these project is in a very early stage there is currently no traditional contribution guideline due to the fact that actually every issue is a bigger change which can bring incompatibility on update processes of this operator.

But everyone can feel welcome to mention ideas and adding features which makes sense what could be actually everything what you can do with helm. The reason why i'm open sourced this project is that different views are needed for a proper further development.


## License
[LICENSE](LICENSE)

