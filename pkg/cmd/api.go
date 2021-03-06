package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// addCmd represents the add command
func NewAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "runs backend for web apps",
		Long:  `restful application`,
		Run: func(cmd *cobra.Command, args []string) {
			sum := 0
			for _, args := range args {
				num, err := strconv.Atoi(args)

				if err != nil {
					fmt.Println(err)
				}
				sum = sum + num
			}
			fmt.Println("result of addition is", sum)
		},
	}
}
