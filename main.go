package main

import (
	"os"

	"github.com/mergermarket/cdflow2/command"
)

func main() {
	os.Exit(command.RunCommand())
}
