// Package main is the apigen CLI entry point.
package main

import (
	"fmt"
	"os"

	"github.com/hellowzsg/api-gen/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "apigen:", err)
		os.Exit(1)
	}
}
