package main

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	sdk "github.com/inovacc/selo"
	"github.com/spf13/cobra"
)

// kindFlags holds the bound flag values for one per-kind subcommand.
type kindFlags struct {
	generate bool
	validate string
	format   string
	origin   string
	from     string
	uf       string
	count    int
}

// newKindCmd builds the Cobra subcommand for a single registered document kind.
// Capability flags (--origin, --uf) are wired only when the underlying type
// implements OriginResolver / UFScoped respectively.
func newKindCmd(kind sdk.Kind) *cobra.Command {
	doc, ok := sdk.Get(kind)
	if !ok {
		// Defensive: registerKindCommands only iterates registered kinds.
		return &cobra.Command{Use: kind.String(), Hidden: true}
	}

	name := kind.String()
	upper := strings.ToUpper(name)
	f := &kindFlags{}

	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Generate, validate, or format %s", upper),
		Example: strings.Join([]string{
			fmt.Sprintf("selo %s --generate", name),
			fmt.Sprintf("selo %s --generate --count 10", name),
			fmt.Sprintf("selo %s --validate <value>", name),
			fmt.Sprintf("selo %s --format <value>", name),
			fmt.Sprintf("selo %s --from values.txt", name),
			fmt.Sprintf("type values.txt | selo %s --from -", name),
		}, "\n"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runKind(cmd, doc, f)
		},
	}

	cmd.Flags().BoolVarP(&f.generate, "generate", "g", false, "Generate a valid "+upper)
	cmd.Flags().StringVarP(&f.validate, "validate", "v", "", "Validate a single "+upper+" value")
	cmd.Flags().StringVar(&f.format, "format", "", "Format a single "+upper+" value")
	cmd.Flags().StringVarP(&f.from, "from", "f", "", "Validate many values from file or '-' for stdin")
	cmd.Flags().IntVarP(&f.count, "count", "n", 0, "When generating, how many values to output")

	if _, ok := doc.(sdk.OriginResolver); ok {
		cmd.Flags().StringVar(&f.origin, "origin", "", "Resolve origin/region of a single "+upper+" value")
	}

	if _, ok := doc.(sdk.UFScoped); ok {
		cmd.Flags().StringVar(&f.uf, "uf", "", "Federative unit (e.g. SP) — required for "+upper)
	}

	return cmd
}

// registerKindCommands adds one subcommand per registered kind to root.
func registerKindCommands(root *cobra.Command) {
	for _, k := range sdk.Kinds() {
		root.AddCommand(newKindCmd(k))
	}
}

// runKind dispatches a single per-kind invocation based on the bound flags.
func runKind(cmd *cobra.Command, doc sdk.Document, f *kindFlags) error {
	if err := f.validateCombo(); err != nil {
		return err
	}

	switch {
	case f.generate:
		return runGenerate(cmd, doc, f.count)
	case f.from != "":
		return runFrom(cmd, doc, f.from)
	case f.format != "":
		return runFormat(cmd, doc, f.format)
	case f.origin != "":
		return runOrigin(cmd, doc, f.origin)
	default:
		return runValidate(cmd, doc, f.validate, f.uf)
	}
}

// validateCombo enforces mutually exclusive / required flag combinations,
// preserving the original CLI's error messages and UX.
func (f *kindFlags) validateCombo() error {
	if f.generate && (f.validate != "" || f.from != "" || f.format != "" || f.origin != "") {
		return errors.New("--generate cannot be used with --validate, --format, --origin, or --from")
	}

	actions := 0

	for _, on := range []bool{f.generate, f.validate != "", f.format != "", f.origin != "", f.from != ""} {
		if on {
			actions++
		}
	}

	if actions == 0 {
		return errors.New("either --generate, --validate, --format, --origin, or --from must be provided")
	}

	if f.from != "" && f.validate != "" {
		return errors.New("--from and --validate are mutually exclusive")
	}

	return nil
}

func runGenerate(cmd *cobra.Command, doc sdk.Document, count int) error {
	if count <= 0 {
		count = 1
	}

	w := bufio.NewWriter(cmd.OutOrStdout())

	defer func() { _ = w.Flush() }()

	for i := 0; i < count; i++ {
		value := doc.Generate()
		if formatted, err := doc.Format(value); err == nil {
			_, _ = fmt.Fprintln(w, formatted)
		} else {
			_, _ = fmt.Fprintln(w, value)
		}
	}

	return nil
}

func runFrom(cmd *cobra.Command, doc sdk.Document, from string) error {
	r, closeFn, err := openReader(from)
	if err != nil {
		return err
	}

	if closeFn != nil {
		defer closeFn()
	}

	fn := func(value string) (string, bool) {
		if !doc.Validate(value) {
			return "", false
		}

		formatted, ferr := doc.Format(value)
		if ferr != nil {
			return "", true
		}

		return formatted, true
	}

	anyInvalid, err := streamValidate(r, cmd.OutOrStdout(), fn)
	if err != nil {
		return err
	}

	if anyInvalid {
		return errInvalidInput
	}

	return nil
}

func runFormat(cmd *cobra.Command, doc sdk.Document, value string) error {
	formatted, err := doc.Format(value)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), formatted)

	return nil
}

func runOrigin(cmd *cobra.Command, doc sdk.Document, value string) error {
	r, ok := doc.(sdk.OriginResolver)
	if !ok {
		return fmt.Errorf("--origin is not supported for %s", doc.Kind())
	}

	origin, err := r.Origin(value)
	if err != nil {
		cmd.SilenceUsage = true
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), origin)

	return nil
}

func runValidate(cmd *cobra.Command, doc sdk.Document, value, uf string) error {
	valid := docValidate(doc, value, uf)
	if !valid {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "invalid")
		return errInvalidInput
	}

	if formatted, err := doc.Format(value); err == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "valid\t%s\n", formatted)
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "valid")
	}

	return nil
}

// docValidate runs UF-scoped validation when a --uf is supplied and the type
// supports it; otherwise it runs plain Validate.
func docValidate(doc sdk.Document, value, uf string) bool {
	if uf != "" {
		if s, ok := doc.(sdk.UFScoped); ok {
			ok2, err := s.ValidateUF(value, sdk.UF(strings.ToUpper(uf)))
			return err == nil && ok2
		}
	}

	return doc.Validate(value)
}
