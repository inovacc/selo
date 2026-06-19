/*
Copyright © 2025 Dyam Marcano

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

	sdk "github.com/inovacc/selo"
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

// newRootCmd assembles the Cobra root command: registry-driven per-kind
// subcommands plus the top-level detect and version commands. UX niceties from
// the original CLI are preserved (SilenceUsage/SilenceErrors, no default
// completion command).
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   sdk.CLIUse,
		Short: sdk.CLIShort,
		Long:  "selo generates, validates, formats, and inspects Brazilian documents. Subcommands are derived from the document registry.",
	}

	root.CompletionOptions.DisableDefaultCmd = true
	// Errors are printed (or suppressed) by main(); avoid duplicate usage/error output.
	root.SilenceUsage = true
	root.SilenceErrors = true

	registerKindCommands(root)
	root.AddCommand(newDetectCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newMCPCmd())
	root.AddCommand(newPersonCmd())
	root.AddCommand(newGenCmd())

	return root
}
