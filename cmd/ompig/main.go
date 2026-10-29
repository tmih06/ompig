package main

import (
	"os"

	"github.com/tmih06/ompig/coding/cli"
)

// Ports packages/coding-agent/src/cli.ts

func main() {
	cli.Main(os.Args[1:], nil)
}
