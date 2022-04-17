+++
title = "Installation"
weight = 10
chapter = true
+++

## Remote

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
