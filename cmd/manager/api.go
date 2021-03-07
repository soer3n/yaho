package manager

import (
	"github.com/soer3n/apps-operator/pkg/api"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
func NewAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "runs backend for web apps",
		Long:  `restful application`,
		Run: func(cmd *cobra.Command, args []string) {
			api.New("9090").Run()
		},
	}
}
