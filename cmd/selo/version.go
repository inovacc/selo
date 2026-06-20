package main

import (
	"fmt"
	"runtime/debug"

	sdk "github.com/inovacc/selo"
	"github.com/spf13/cobra"
)

// Build-time metadata injected via -ldflags -X by GoReleaser. They stay empty
// for `go run` / `go install`, in which case version() falls back to the module
// build info (and then to "dev").
var (
	buildVersion = ""
	buildCommit  = ""
	buildDate    = ""
)

// version resolves the build version: the ldflags-injected value when present,
// otherwise the module build info, defaulting to "dev" for `go run` / un-versioned
// builds.
func version() string {
	if buildVersion != "" {
		return buildVersion
	}

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

			if buildCommit != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n  built:  %s\n", buildCommit, buildDate)
			}

			return nil
		},
	}
}
