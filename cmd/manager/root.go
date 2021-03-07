package manager

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manager",
		Short: "manager app",
		Long:  `manager app`,
	}

	cmd.AddCommand(NewOperatorCmd())
	cmd.AddCommand(NewAPICmd())
	return cmd
}
