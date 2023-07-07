package utils

import (
	"context"
	"errors"
	"fmt"

	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	sa "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	conf "sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/go-logr/logr"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
)

func NewRESTClientGetter(config *yahov1alpha2.Config, namespace, releaseNamespace string, isLocal bool, c client.Client, logger logr.Logger) (*HelmRESTClientGetter, error) {

	getter := &HelmRESTClientGetter{
		Namespace:        namespace,
		ReleaseNamespace: releaseNamespace,
		HelmConfig:       config,
		Client:           c,
		logger:           logger,
		IsLocal:          isLocal,
	}

	if err := getter.setKubeconfig(); err != nil {
		return nil, err
	}

	return getter, nil
}

func (h *HelmRESTClientGetter) setKubeconfig() error {

	clienCmdConfig := h.ToRawKubeConfigLoader()
	clientConfig, err := clienCmdConfig.RawConfig()

	if err != nil {
		return err
	}

	rawConfig, err := runtime.Encode(clientcmdlatest.Codec, &clientConfig)

	if err != nil {
		return err
	}

	h.KubeConfig = string(rawConfig)

	return nil

}

func (c *HelmRESTClientGetter) ToRESTConfig() (*rest.Config, error) {

	if len(c.KubeConfig) == 0 {
		return nil, errors.New("kubeconfig not generated")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(c.KubeConfig))

	if err != nil {
		c.logger.Info(err.Error(), "key", "torest")
		return nil, err
	}
	return config, nil
}

func (c *HelmRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := c.ToRESTConfig()
	if err != nil {
		c.logger.Info(err.Error(), "key", "discover")
		return nil, err
	}

	// The more groups you have, the more discovery requests you need to make.
	// given 25 groups (our groups + a few custom conf) with one-ish version each, discovery needs to make 50 requests
	// double it just so we don't end up here again for a while.  This config is only used for discovery.
	config.Burst = 100

	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(config)
	return memory.NewMemCacheClient(discoveryClient), nil
}

func (c *HelmRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := c.ToDiscoveryClient()
	if err != nil {
		c.logger.Info(err.Error(), "key", "tomapper")
		return nil, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	expander := restmapper.NewShortcutExpander(mapper, discoveryClient)
	return expander, nil
}

func (c *HelmRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {

	serviceAccount := &v1.ServiceAccount{}
	serviceAccountName := "default"

	if c.HelmConfig != nil {
		serviceAccountName = c.HelmConfig.Spec.ServiceAccountName
	}

	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: c.Namespace, Name: serviceAccountName}, serviceAccount); err != nil {
		fmt.Printf("error on getting service account. msg: %v", err.Error())
		return nil
	}

	var secret *v1.Secret

	// for kubernetes >= 1.24 we need to create and connect the secret token by ourself
	if serviceAccount.Secrets == nil {
		rc, err := conf.GetConfig()

		if err != nil {
			fmt.Println(err.Error())
			return nil
		}

		sac, err := sa.NewForConfig(rc)

		if err != nil {
			fmt.Println(err.Error())
			return nil
		}

		tokenSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "account-secret-" + serviceAccountName,
				Namespace: c.Namespace,
				Annotations: map[string]string{
					"kubernetes.io/service-account.name": serviceAccountName,
				},
			},
			Type: v1.SecretTypeServiceAccountToken,
		}

		err = c.Client.Get(context.Background(), types.NamespacedName{Namespace: c.Namespace, Name: tokenSecret.Name}, tokenSecret)

		if err != nil {
			fmt.Printf("error on getting token secret. msg: %v\n", err.Error())
		}

		if k8serrors.IsNotFound(err) {

			fmt.Println("create secret...")

			if err := c.Client.Create(context.Background(), tokenSecret); err != nil {
				fmt.Printf("error on creating token secret. msg: %v\n", err.Error())
				return nil
			}

			request := &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences: []string{},
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						Kind:       "Secret",
						APIVersion: "v1",
						Name:       "account-secret-" + serviceAccountName,
					},
				},
			}

			fmt.Println("create token...")

			tr, err := sac.ServiceAccounts(c.Namespace).CreateToken(context.TODO(), serviceAccountName, request, metav1.CreateOptions{})

			if err != nil {
				fmt.Printf("failed to create token: %v\n", err)
			}
			if len(tr.Status.Token) == 0 {
				fmt.Println("failed to create token: no token in server response")
			}

			fmt.Println("get updated secret...")

			if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: c.Namespace, Name: tokenSecret.Name}, tokenSecret); err != nil {
				fmt.Printf("error on getting token secret. msg: %v\n", err.Error())
			}
		}

		secret = tokenSecret
	}

	if secret == nil {
		tokenSecret := &v1.Secret{}
		for _, s := range serviceAccount.Secrets {
			if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: c.Namespace, Name: s.Name}, tokenSecret); err != nil {
				fmt.Printf("error on getting token secret. msg: %v", err.Error())
				continue
			}
			break
		}

		secret = tokenSecret
	}

	rawToken := secret.Data["token"]

	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default-cluster"] = &clientcmdapi.Cluster{
		Server:                   "https://kubernetes.svc.default.cluster.local",
		CertificateAuthorityData: secret.Data["ca.crt"],
	}

	// for testing purposes rewrite cluster apiserver address
	if c.IsLocal {
		clusters["default-cluster"].Server = "https://127.0.0.1:6443"
	}

	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default-context"] = &clientcmdapi.Context{
		Cluster:   "default-cluster",
		Namespace: c.ReleaseNamespace,
		AuthInfo:  c.ReleaseNamespace,
	}

	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos[c.ReleaseNamespace] = &clientcmdapi.AuthInfo{
		Token: string(rawToken),
	}

	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default-context",
		AuthInfos:      authinfos,
	}

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	overrides.Context.Namespace = c.ReleaseNamespace

	rawConfig, err := runtime.Encode(clientcmdlatest.Codec, &clientConfig)

	if err != nil {
		return nil
	}

	returnClient, _ := clientcmd.NewClientConfigFromBytes(rawConfig)

	return returnClient
}
