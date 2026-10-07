package main

import (
	"os"

	"github.com/forkledger/forkledger/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
