package main

import (
	"log"

	"github.com/soer3n/yaho/cmd/manager"
)

func main() {
	command := manager.NewRootCmd()

	if err := command.Execute(); err != nil {
		log.Fatal(err.Error())
	}
}
