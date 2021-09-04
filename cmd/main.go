package main

import (
	"github.com/soer3n/yaho/cmd/manager"
)

func main() {
	command := manager.NewRootCmd()
	command.Execute()
}
