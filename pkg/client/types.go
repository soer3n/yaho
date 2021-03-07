package client

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Client struct {
	RestClientGetter genericclioptions.RESTClientGetter
	Factory          Factory
}
