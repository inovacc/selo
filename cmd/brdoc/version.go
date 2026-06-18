package main

import (
	"fmt"
	"runtime/debug"

	sdk "github.com/inovacc/brdoc"
	"github.com/spf13/cobra"
)

// version resolves the build version from the module build info, defaulting to
// "dev" for `go run` / un-versioned builds.
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}

	return "dev"
}

// newVersionCmd builds the top-level "version" command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the " + sdk.AppName + " version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", sdk.AppName, version())
			return nil
		},
	}
}
