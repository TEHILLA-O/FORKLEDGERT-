package main

import (
	"os"

	"github.com/TEHILLA-O/forkledger/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
