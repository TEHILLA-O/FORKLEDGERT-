package main

import (
	"os"

	"github.com/TEHILLA-O/fork_ledger/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
