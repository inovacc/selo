package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdk "github.com/inovacc/selo"
	// codegen drives generation; its language emitters self-register via init()
	// (M2 registers the TypeScript emitter in emit_ts.go), so importing the
	// package is what wires `selo gen --lang ts` end to end.
	"github.com/inovacc/selo/internal/codegen"
	"github.com/spf13/cobra"
)

// genFlags holds the bound flags for the `selo gen` command.
type genFlags struct {
	lang string
	kind string
	out  string
}

// newGenCmd builds the `selo gen` command, which generates idiomatic
// validate/format/origin code (plus golden vectors and tests) for a target
// language from the verified selo library. Emitters are registered by M2+; in
// M1 the command wiring, flags, and help are in place but generation reports
// that the requested language emitter is not yet registered (and exits non-zero).
func newGenCmd() *cobra.Command {
	f := &genFlags{}

	langs := strings.Join(codegen.SupportedLangStrings(), ", ")
	kinds := strings.Join(codegen.KindStrings(), ", ")

	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Generate validate/format/origin code for other languages from selo",
		Long: "selo gen emits idiomatic Brazilian-document validation, formatting, and " +
			"origin code — with Go-produced golden test vectors and a runnable test — " +
			"for a target language.\n\n" +
			"Supported languages: " + langs + "\n" +
			"Supported kinds: all, " + kinds,
		Example: strings.Join([]string{
			"selo gen --lang ts --kind cpf --out ./generated",
			"selo gen --lang ts --kind all",
			"selo gen --lang ruby --kind cnpj --out ./out",
		}, "\n"),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGen(cmd, f)
		},
	}

	cmd.Flags().StringVar(&f.lang, "lang", "", "Target language: "+langs+" (required)")
	cmd.Flags().StringVar(&f.kind, "kind", "all", "Document kind to generate, or 'all'")
	cmd.Flags().StringVar(&f.out, "out", "./generated", "Output directory root")

	return cmd
}

// runGen validates the flags and drives codegen.Generate for the selected
// kind(s). In M1 (no registered emitters) it surfaces the "not yet registered"
// error and returns non-nil so main() exits non-zero.
func runGen(cmd *cobra.Command, f *genFlags) error {
	if f.lang == "" {
		return fmt.Errorf("--lang is required (supported: %s)", strings.Join(codegen.SupportedLangStrings(), ", "))
	}
	if !codegen.IsSupportedLang(f.lang) {
		return fmt.Errorf("unsupported --lang %q (supported: %s)", f.lang, strings.Join(codegen.SupportedLangStrings(), ", "))
	}

	kinds, err := resolveKinds(f.kind)
	if err != nil {
		return err
	}

	lang := codegen.Lang(f.lang)
	for _, k := range kinds {
		files, gerr := codegen.Generate(lang, k)
		if gerr != nil {
			return gerr
		}
		if werr := writeFiles(f.out, files); werr != nil {
			return werr
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "generated %s/%s (%d files)\n", f.lang, k, len(files))
	}
	return nil
}

// resolveKinds expands the --kind flag into the concrete kinds to generate.
func resolveKinds(kind string) ([]sdk.Kind, error) {
	if kind == "" || kind == "all" {
		return sdk.Kinds(), nil
	}
	k := sdk.Kind(kind)
	if _, ok := codegen.PlanFor(k); !ok {
		return nil, fmt.Errorf("unknown --kind %q (use 'all' or one of: %s)", kind, strings.Join(codegen.KindStrings(), ", "))
	}
	return []sdk.Kind{k}, nil
}

// writeFiles writes a generated file set rooted at out, creating directories as
// needed.
func writeFiles(out string, files []codegen.File) error {
	for _, file := range files {
		dst := filepath.Join(out, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("gen: mkdir for %q: %w", dst, err)
		}
		if err := os.WriteFile(dst, file.Content, 0o644); err != nil {
			return fmt.Errorf("gen: write %q: %w", dst, err)
		}
	}
	return nil
}
