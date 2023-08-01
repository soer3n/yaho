package main

import (
	"log"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	yahov1alpha2 "github.com/soer3n/yaho/apis/yaho/v1alpha2"
	manager "github.com/soer3n/yaho/internal/cmd"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1beta1.AddToScheme(scheme))

	utilruntime.Must(helmv1alpha1.AddToScheme(scheme))
	utilruntime.Must(yahov1alpha2.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	command := newRootCmd()

	if err := command.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manager",
		Short: "manager app",
		Long:  `manager app`,
	}

	cmd.AddCommand(manager.NewOperatorCmd(scheme))
	cmd.AddCommand(manager.NewAgentCmd(scheme))
	return cmd
}
