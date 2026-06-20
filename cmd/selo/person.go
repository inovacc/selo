package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	sdk "github.com/inovacc/selo"
	"github.com/spf13/cobra"
)

// newPersonCmd builds the "person" command: generate coherent synthetic
// Brazilian identities (all documents, UF-consistent) as text or JSON.
func newPersonCmd() *cobra.Command {
	var (
		count                            int
		ufFlag                           string
		asJSON, withVehicle, withCompany bool
		formatted                        bool
		seed                             int64
	)

	cmd := &cobra.Command{
		Use:   "person",
		Short: "Generate coherent synthetic Brazilian people (all documents, UF-consistent)",
		Long: "Generate fake-but-valid Brazilian identities for testing/fixtures. Every " +
			"document validates and the geolocatable ones share the same UF. Synthetic " +
			"data only — never real PII.",
		Args: cobra.NoArgs,
		Example: "selo person\n" +
			"selo person --count 5 --uf SP --json\n" +
			"selo person --uf SP --seed 42 --json\n" +
			"selo person --uf RJ --with-vehicle --with-company --formatted",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if count < 1 {
				count = 1
			}

			var opts []sdk.PersonOption

			if cmd.Flags().Changed("seed") {
				// One shared source so a --count batch is reproducible yet
				// still yields distinct people (the stream advances per draw).
				opts = append(opts, sdk.WithRand(sdk.NewSeededRand(seed)))
			}

			if ufFlag != "" {
				uf := sdk.UF(strings.ToUpper(ufFlag))
				if !uf.Valid() {
					return fmt.Errorf("invalid --uf %q", ufFlag)
				}

				opts = append(opts, sdk.WithUF(uf))
			}

			if withVehicle {
				opts = append(opts, sdk.WithVehicle())
			}

			if withCompany {
				opts = append(opts, sdk.WithCompany())
			}

			if formatted {
				opts = append(opts, sdk.Formatted())
			}

			people := make([]sdk.Person, count)
			for i := range people {
				people[i] = sdk.GeneratePerson(opts...)
			}

			out := cmd.OutOrStdout()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")

				if count == 1 {
					return enc.Encode(people[0])
				}

				return enc.Encode(people)
			}

			for i, p := range people {
				if i > 0 {
					_, _ = fmt.Fprintln(out)
				}

				printPerson(out, p)
			}

			return nil
		},
	}
	cmd.Flags().IntVarP(&count, "count", "n", 1, "number of people to generate")
	cmd.Flags().StringVar(&ufFlag, "uf", "", "pin the federative unit (e.g. SP); random if omitted")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON instead of text")
	cmd.Flags().BoolVar(&withVehicle, "with-vehicle", false, "also generate a vehicle (plate + RENAVAM)")
	cmd.Flags().BoolVar(&withCompany, "with-company", false, "also generate a company (CNPJ)")
	cmd.Flags().BoolVar(&formatted, "formatted", false, "output documents in masked form")
	cmd.Flags().Int64Var(&seed, "seed", 0, "pin the random seed for deterministic, reproducible output")

	return cmd
}

func printPerson(w io.Writer, p sdk.Person) {
	_, _ = fmt.Fprintf(w, "Name:    %s\n", p.Name)
	_, _ = fmt.Fprintf(w, "Email:   %s\n", p.Email)
	_, _ = fmt.Fprintf(w, "UF:      %s\n", p.UF)

	_, _ = fmt.Fprintf(w, "CPF:     %s\n", p.CPF)
	if p.RG != "" {
		_, _ = fmt.Fprintf(w, "RG:      %s\n", p.RG)
	}

	_, _ = fmt.Fprintf(w, "CNH:     %s\n", p.CNH)
	_, _ = fmt.Fprintf(w, "PIS:     %s\n", p.PIS)
	_, _ = fmt.Fprintf(w, "RENAVAM: %s\n", p.Renavam)
	_, _ = fmt.Fprintf(w, "VoterID: %s\n", p.VoterID)
	_, _ = fmt.Fprintf(w, "CNS:     %s\n", p.CNS)
	_, _ = fmt.Fprintf(w, "CEP:     %s\n", p.CEP)
	_, _ = fmt.Fprintf(w, "Phone:   %s\n", p.Phone)

	_, _ = fmt.Fprintf(w, "PIX:     %s\n", strings.Join(p.PIXKeys, ", "))
	if p.Vehicle != nil {
		_, _ = fmt.Fprintf(w, "Vehicle: %s / %s\n", p.Vehicle.Plate, p.Vehicle.Renavam)
	}

	if p.Company != nil {
		_, _ = fmt.Fprintf(w, "Company: %s (%s)\n", p.Company.Name, p.Company.CNPJ)
	}
}
