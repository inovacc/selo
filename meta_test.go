package brdoc

import "testing"

func TestMetaConstants(t *testing.T) {
	if AppName == "" {
		t.Fatal("AppName must not be empty")
	}
	if CLIUse != "brdoc" {
		t.Fatalf("CLIUse = %q, want \"brdoc\"", CLIUse)
	}
	if MCPServerName == "" {
		t.Fatal("MCPServerName must not be empty")
	}
	if CLIShort == "" {
		t.Fatal("CLIShort must not be empty")
	}
}
