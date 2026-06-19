package main

import (
	mcpserver "github.com/inovacc/selo/mcp"
	"github.com/spf13/cobra"
	"os/signal"
	"syscall"
)

// newMCPCmd builds the top-level "mcp" command, which runs the selo MCP
// server over stdio. It matches the M1-4 factory style (newDetectCmd /
// newVersionCmd); the version string is resolved from the shared version()
// helper defined in version.go (M1-4).
func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run selo as a Model Context Protocol server over stdio",
		Long: "Start an MCP server exposing selo's validate, generate, format, " +
			"detect, and list tools to agents over stdin/stdout. Logs go to stderr.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Cancel on SIGINT/SIGTERM so the stdio loop exits cleanly.
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			return mcpserver.Serve(ctx, version())
		},
	}
}
