package main

import (
	"log"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	helmv1alpha1 "github.com/soer3n/yaho/apis/yaho/v1alpha1"
	manager "github.com/soer3n/yaho/internal/cmd"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(helmv1alpha1.AddToScheme(scheme))
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
