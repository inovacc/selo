/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// errInvalidInput is the sentinel returned by RunE handlers when a document
// fails validation. The handler prints "invalid" to stdout first, then returns
// this error so that main() exits with code 1. Because SilenceErrors is set
// on the root command, Cobra will not print it; main() also suppresses it so
// the only output is the "invalid" line already written to stdout.
var errInvalidInput = errors.New("invalid input")

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		if !errors.Is(err, errInvalidInput) {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

// newRootCmd constructs and returns a fresh root Cobra command with all
// per-kind subcommands registered. Using a constructor (rather than a package-
// level var) makes the command tree re-entrant and safe for parallel tests.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "brdoc",
		Short: "Brazilian documents utilities",
		Long:  "brdoc is a small CLI to generate and validate Brazilian documents like CPF and CNPJ.",
	}

	root.CompletionOptions.DisableDefaultCmd = true
	root.SilenceUsage = true
	root.SilenceErrors = true

	registerKindCommands(root)

	return root
}
