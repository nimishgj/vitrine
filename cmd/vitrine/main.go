package main

import (
	"os"

	"github.com/nimishgj/vitrine/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[0], os.Args[1:]))
}
