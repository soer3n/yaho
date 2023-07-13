package utils

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	actionlog "log"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	sa "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/klog/v2"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	conf "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func ManagerOptions(config string) (*manager.Options, error) {

	c, err := parseOperatorConfig(config)

	if err != nil {
		return nil, err
	}

	return &manager.Options{
		HealthProbeBindAddress: c.HealthProbeBindAddress,
		LeaderElection:         c.LeaderElection.Enabled,
		LeaderElectionID:       c.LeaderElection.ResourceID,
		MetricsBindAddress:     c.MetricsBindAddress,
	}, nil
}

func parseOperatorConfig(path string) (*Config, error) {
	fd, err := os.Open(filepath.Clean(filepath.Join(path)))
	if err != nil {
		return nil, fmt.Errorf("could not open the configuration file: %v", err)
	}
	defer fd.Close()

	cfg := Config{}

	if err = yaml.NewDecoder(fd).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("could not decode configuration file: %v", err)
	}

	return &cfg, nil
}

func BuildKubeconfigSecret(path, address, name, namespace string, scheme *runtime.Scheme) (*v1.Secret, error) {

	klog.V(0).Infof("creating client from config path %s", path)
	config, err := clientcmd.BuildConfigFromFlags("", path)

	if err != nil {
		return nil, err
	}

	c, err := client.New(config, client.Options{Scheme: scheme})

	if err != nil {
		return nil, err
	}

	serviceAccount := &v1.ServiceAccount{}
	klog.V(0).Infof("looking for service account 'yaho-agent in namespace %s", namespace)
	err = c.Get(context.TODO(), types.NamespacedName{Name: "yaho-agent", Namespace: namespace}, serviceAccount, &client.GetOptions{})

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
		klog.V(0).Info("creating service account")
		serviceAccount = &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "yaho-agent",
				Namespace: namespace,
			},
		}

		if err := c.Create(context.TODO(), serviceAccount, &client.CreateOptions{}); err != nil {
			return nil, err
		}
	}

	var secret *v1.Secret

	// for kubernetes >= 1.24 we need to create and connect the secret token by ourself
	if serviceAccount.Secrets == nil {

		klog.V(0).Info("creating service account secret")
		rc, err := conf.GetConfig()

		if err != nil {
			return nil, err
		}

		sac, err := sa.NewForConfig(rc)

		if err != nil {
			return nil, err
		}

		tokenSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "account-secret-" + "yaho-agent",
				Namespace: namespace,
				Annotations: map[string]string{
					"kubernetes.io/service-account.name": "yaho-agent",
				},
			},
			Type: v1.SecretTypeServiceAccountToken,
		}

		err = c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: tokenSecret.Name}, tokenSecret)

		if err != nil {
			klog.V(0).Infof("error on getting token secret. msg: %v\n", err.Error())
		}

		if k8serrors.IsNotFound(err) {

			klog.V(0).Info("create secret...")

			if err := c.Create(context.Background(), tokenSecret); err != nil {
				klog.V(0).Infof("error on creating token secret. msg: %v\n", err.Error())
				return nil, err
			}

			request := &authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					Audiences: []string{},
					BoundObjectRef: &authenticationv1.BoundObjectReference{
						Kind:       "Secret",
						APIVersion: "v1",
						Name:       "account-secret-" + serviceAccount.Name,
					},
				},
			}

			klog.V(0).Info("create token...")

			tr, err := sac.ServiceAccounts(namespace).CreateToken(context.TODO(), serviceAccount.Name, request, metav1.CreateOptions{})

			if err != nil {
				klog.V(0).Infof("failed to create token: %v\n", err)
			}
			if len(tr.Status.Token) == 0 {
				klog.V(0).Info("failed to create token: no token in server response")
			}

			klog.V(0).Info("get updated secret...")

			if err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: tokenSecret.Name}, tokenSecret); err != nil {
				klog.V(0).Infof("error on getting token secret. msg: %v\n", err.Error())
			}
		}

		secret = tokenSecret
	}

	if secret == nil {
		klog.V(0).Info("getting existing service account secret")
		tokenSecret := &v1.Secret{}
		for _, s := range serviceAccount.Secrets {
			if err := c.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: s.Name}, tokenSecret); err != nil {
				klog.V(0).Infof("error on getting token secret. msg: %v", err.Error())
				continue
			}
			break
		}

		secret = tokenSecret
	}

	rawToken := secret.Data["token"]

	klog.V(0).Info("set cluster in config")
	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default-cluster"] = &clientcmdapi.Cluster{
		Server:                   address,
		CertificateAuthorityData: secret.Data["ca.crt"],
	}

	klog.V(0).Info("set context in config")
	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default-context"] = &clientcmdapi.Context{
		Cluster:   "default-cluster",
		Namespace: namespace,
		AuthInfo:  "yaho-agent",
	}

	klog.V(0).Info("set auth info in config")
	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos["yaho-agent"] = &clientcmdapi.AuthInfo{
		Token: string(rawToken),
	}

	klog.V(0).Info("set config struct")
	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default-context",
		AuthInfos:      authinfos,
	}

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	overrides.Context.Namespace = namespace

	klog.V(0).Info("encode config")
	rawConfig, err := runtime.Encode(clientcmdlatest.Codec, &clientConfig)

	if err != nil {
		return nil, err
	}

	klog.V(0).Info("try to get deployment role role")
	role := &rbacv1.Role{}

	err = c.Get(context.TODO(), types.NamespacedName{Name: "agent-role", Namespace: namespace}, role, &client.GetOptions{})

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
		klog.V(0).Info("build deployment role")
		role = &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "agent-role",
				Namespace: namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "serviceaccounts"},
					Verbs:     []string{"create", "update", "get", "list", "watch"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"serviceaccounts"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}

		klog.V(0).Info("deploy deployment role")
		if err := c.Create(context.TODO(), role, &client.CreateOptions{}); err != nil {
			return nil, err
		}
	}

	klog.V(0).Info("try to get cluster role")
	clusterRole := &rbacv1.ClusterRole{}

	err = c.Get(context.TODO(), types.NamespacedName{Name: "agent-role"}, clusterRole, &client.GetOptions{})

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
		klog.V(0).Info("build cluster role")
		clusterRole = &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: "agent-role",
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
					Verbs:     []string{"create", "update", "get", "list", "watch"},
				},
				{
					APIGroups: []string{"yaho.soer3n.dev"},
					Resources: []string{"releases/finalizers", "values/status"},
					Verbs:     []string{"get", "list", "watch", "update"},
				},
				{
					APIGroups: []string{"yaho.soer3n.dev"},
					Resources: []string{"releases", "releases/status", "values"},
					Verbs:     []string{"get", "list", "watch", "update", "patch"},
				},
			},
		}

		klog.V(0).Info("deploy cluster role")
		if err := c.Create(context.TODO(), clusterRole, &client.CreateOptions{}); err != nil {
			return nil, err
		}
	}

	klog.V(0).Info("try to get deployment role binding")
	roleBinding := &rbacv1.RoleBinding{}

	err = c.Get(context.TODO(), types.NamespacedName{Name: "yaho-agent-binding", Namespace: namespace}, roleBinding, &client.GetOptions{})

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
		klog.V(0).Info("build deployment role binding")
		roleBinding = &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "yaho-agent-binding",
				Namespace: namespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "yaho-agent",
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "agent-role",
			},
		}

		klog.V(0).Info("deploy cluster role binding")
		if err := c.Create(context.TODO(), roleBinding, &client.CreateOptions{}); err != nil {
			klog.V(0).Info(err.Error())
			return nil, err
		}
	}

	klog.V(0).Info("try to get cluster role binding")
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err = c.Get(context.TODO(), types.NamespacedName{Name: "yaho-agent-binding"}, clusterRoleBinding, &client.GetOptions{})

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
		klog.V(0).Info("build cluster role binding")
		clusterRoleBinding = &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "yaho-agent-binding",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "yaho-agent",
					Namespace: namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "agent-role",
			},
		}

		klog.V(0).Info("deploy cluster role binding")
		if err := c.Create(context.TODO(), clusterRoleBinding, &client.CreateOptions{}); err != nil {
			klog.V(0).Info(err.Error())
			return nil, err
		}
	}

	str := base64.StdEncoding.EncodeToString(rawConfig)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
	}

	agentSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"kubeconfig": []byte(string(data)),
		},
	}

	return agentSecret, nil
}

// InitActionConfig represents the initialization of an helm configuration
func InitActionConfig(getter genericclioptions.RESTClientGetter, kubeconfig []byte, logger logr.Logger) (*action.Configuration, error) {
	/*
		/ we cannot use helm init func here due to data race issues on concurrent execution (helm's kube client tries to update the namespace field on each initialization)

		// actionConfig := new(action.Configuration)
		err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), actionlog.Printf)
	*/

	if getter == nil {
		logger.Info("getter is nil")
		return nil, errors.New("getter is nil")
	}

	f := cmdutil.NewFactory(getter)
	set, err := f.KubernetesClientSet()

	if err != nil {
		fmt.Print(err.Error())
		return nil, err
	}

	casted, ok := getter.(*HelmRESTClientGetter)
	namespace := "default"

	if ok {
		namespace = casted.ReleaseNamespace
	}

	c := &kube.Client{
		Factory:   f,
		Log:       actionlog.Printf,
		Namespace: namespace,
	}

	conf := &action.Configuration{
		RESTClientGetter: getter,
		KubeClient:       c,
		Log:              actionlog.Printf,
		Releases:         storage.Init(driver.NewSecrets(set.CoreV1().Secrets(namespace))),
	}

	return conf, nil
}

// MergeMaps returns distinct map of two as input
// have to be called as a goroutine to avoid memory leaks
func MergeMaps(source, dest map[string]interface{}) map[string]interface{} {
	if source == nil || dest == nil {
		return dest
	}

	copy := make(map[string]interface{})

	for k, v := range dest {
		copy[k] = v
	}

	for k, v := range source {
		// when key already exists we have to compare also sub values
		if temp, ok := v.(map[string]interface{}); ok {
			merge, _ := copy[k].(map[string]interface{})
			copy[k] = MergeMaps(merge, temp)
			continue
		}

		copy[k] = v
	}

	return copy
}

// CopyUntypedMap return a copy of a map with strings as keys and empty interface as value
func CopyUntypedMap(source map[string]interface{}) map[string]interface{} {

	vals := make(map[string]interface{})

	for k, v := range source {
		vals[k] = v
	}

	return vals
}

// MergeUntypedMaps returns distinct map of two as input
func MergeUntypedMaps(dest, source map[string]interface{}, keys ...string) map[string]interface{} {

	trimedKeys := []string{}
	copy := make(map[string]interface{})

	for k, v := range dest {
		copy[k] = v
	}

	for _, v := range keys {
		if v == "" {
			continue
		}
		trimedKeys = append(trimedKeys, v)
	}

	for l, k := range trimedKeys {
		if l == 0 {
			_, ok := copy[k].(map[string]interface{})

			if !ok {
				copy[k] = make(map[string]interface{})
			}

			if len(trimedKeys) == 1 {
				helper := copy[k].(map[string]interface{})
				for kv, v := range source {
					helper[kv] = v
				}
				copy[k] = helper
			}

			continue
		} else {
			if l > 1 {
				break
			}
			if _, ok := copy[k].(map[string]interface{}); ok {
				helper := copy[trimedKeys[0]].(map[string]interface{})
				if l == len(trimedKeys)-1 {
					subHelper, ok := helper[k].(map[string]interface{})

					if ok {
						for sk, sv := range source {
							subHelper[sk] = sv
						}
						helper[k] = subHelper
					} else {
						helper[k] = source
					}
					copy[trimedKeys[0]] = helper
				} else {
					sub := MergeUntypedMaps(helper, source, trimedKeys[2:]...)
					helper[k] = sub
					copy[trimedKeys[0]] = helper
				}
			}
		}
	}

	if len(trimedKeys) == 0 {
		for k, v := range source {
			copy[k] = v
		}
	}

	return copy
}

// GetEnvSettings represents func for returning helm cli settings which are needed for helm actions
func GetEnvSettings(env map[string]string) *cli.EnvSettings {
	settings := cli.New()

	if env == nil {
		return settings
	}

	// overwrite default settings if requested
	for k, v := range env {
		switch k {
		case "KubeConfig":
			settings.KubeConfig = v
		case "KubeContext":
			settings.KubeContext = v
		case "KubeToken":
			settings.KubeToken = v
		case "KubeAsUser":
			settings.KubeAsUser = v
		case "KubeAsGroups":
			settings.KubeAsGroups = []string{v}
		case "KubeAPIServer":
			settings.KubeAPIServer = v
		// case "KubeCaFile":
		//	settings.KubeCaFile = v
		// case "Debug":
		//	settings.Debug = v
		case "RegistryConfig":
			settings.RegistryConfig = v
		case "RepositoryConfig":
			settings.RepositoryConfig = v
		case "RepositoryCache":
			settings.RepositoryCache = v
		case "PluginsDirectory":
			settings.PluginsDirectory = v
			// case "MaxHistory":
			//	settings.MaxHistory = v
		}
	}

	return settings
}
