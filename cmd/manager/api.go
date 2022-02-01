package manager

import (
	"log"

	"github.com/soer3n/yaho/internal/api"
	"github.com/soer3n/yaho/internal/client"
	"github.com/spf13/cobra"
)

// NewAPICmd represents the api subcommand
func NewAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "runs backend for web apps",
		Long:  `restful application`,
		Run: func(cmd *cobra.Command, args []string) {
			c := client.New()
			if err := api.New("8080", c).Run(); err != nil {
				log.Fatal(err.Error())
			}
		},
	}
}
