package main

import (
	"fmt"

	sdk "github.com/inovacc/selo"
	"github.com/spf13/cobra"
)

// newDetectCmd builds the top-level "detect <value>" command, which reports the
// document kind inferred by the registry's length-based Detect.
func newDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "detect <value>",
		Short:   "Auto-detect the document kind of a value",
		Args:    cobra.ExactArgs(1),
		Example: "selo detect 529.982.247-25\nselo detect 39.591.842/0001-10",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := sdk.Detect(args[0])
			if !ok {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "unknown")
				cmd.SilenceUsage = true

				return errInvalidInput
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), kind.String())

			return nil
		},
	}
}
