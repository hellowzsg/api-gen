// Package main is the apigen CLI entry point.
package main

import (
	"fmt"
	"os"

	"github.com/acme/apigen/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "apigen:", err)
		os.Exit(1)
	}
}
