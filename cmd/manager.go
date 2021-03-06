package main

import (
	"github.com/soer3n/apps-operator/pkg/cmd"
)

func main() {
	command := cmd.NewRootCmd()
	command.Execute()
}
