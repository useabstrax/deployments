package main

import (
	"os"

	"github.com/useabstrax/abstrax/plugins/deploy/internal/commands"
)

func main() {
	os.Exit(commands.Execute())
}
