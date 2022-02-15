## Usage

There is an more complex configuration sample in [these](../examples/) directory. You should deploy all needed repositories before deploying a release or releasegroup. Kubectl can be used for filtering repos, charts and releases due to set labels by controllers.

```

# as an example if you deployed repo and repogroup resource from sample directory
# you see an output like this:

$ kubectl apply -f config/samples/helm_v1alpha1_repo.yaml
$ kubectl apply -f config/samples/helm_v1alpha1_repogroup.yaml

$ kubectl get repoes.helm.soer3n.info -n helm

NAME         GROUP   CREATED_AT
bitnami      foo     2021-06-16T13:38:52Z
nextcloud    foo     2021-06-16T13:38:52Z
submariner           2021-06-16T13:38:57Z

# you can also filter by group label

$ kubectl get repoes.helm.soer3n.info -n helm -l repoGroup=foo

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


$ kubectl get releases.helm.soer3n.info -n helm -l chart=submariner-operator,repo=submariner

NAME              GROUP   REPO         CHART                 CREATED_AT
release-sample2           submariner   submariner-operator   2021-06-16T13:57:58Z

```
