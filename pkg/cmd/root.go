package cmd

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "demo app to demonstrate cobra",
		Long:  `demo app to demonstrate cobra by addition`,
	}

	cmd.AddCommand(NewOperatorCmd())
	cmd.AddCommand(NewAPICmd())
	return cmd
}
