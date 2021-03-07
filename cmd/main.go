package main

import (
	"github.com/soer3n/apps-operator/cmd/manager"
)

func main() {
	command := manager.NewRootCmd()
	command.Execute()
}
