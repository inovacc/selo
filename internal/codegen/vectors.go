package codegen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/inovacc/selo"
)

// vectors.go is the ONLY place in the generator that runs the real selo
// algorithms. It produces the golden JSON vectors that every generated
// language module is tested against, so a wrong port fails its vectors. Valid
// cases mix curated authoritative samples with selo.Generate output; invalid
// cases are produced by SYSTEMATIC mutation and each is re-checked with
// selo.Validate so a mutation that accidentally stays valid is dropped.

// generatedSamplesPerKind is how many selo.Generate values are added to the
// valid set for each kind (on top of curated samples).
const generatedSamplesPerKind = 8

// roundTripCount is the generateRoundTrip hint emitted per kind: how many values
// a target-language test should generate and assert valid, where generation is
// implemented.
const roundTripCount = 100

// ValidateCase is one validation vector: input and the expected validity.
type ValidateCase struct {
	Input string `json:"input"`
	Valid bool   `json:"valid"`
	// UF is set only for UF-scoped kinds (RG, IE) to record which federative
	// unit the case was validated under.
	UF string `json:"uf,omitempty"`
}

// FormatCase is one formatting vector: input and either the canonical output or
// an error sentinel name.
type FormatCase struct {
	Input  string `json:"input"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// OriginCase is one origin vector: input and the expected origin string.
type OriginCase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

// Vector is the full golden vector for one kind (design spec §4 schema).
type Vector struct {
	Kind              string         `json:"kind"`
	Group             string         `json:"group"`
	Validate          []ValidateCase `json:"validate"`
	Format            []FormatCase   `json:"format"`
	Origin            []OriginCase   `json:"origin,omitempty"`
	UFScoped          bool           `json:"ufScoped"`
	UFs               []string       `json:"ufs,omitempty"`
	GenerateRoundTrip int            `json:"generateRoundTrip"`
}

// curatedValid holds authoritative, hand-verified valid samples per kind. Every
// value here is confirmed valid by selo.Validate in TestVectors_ValidCasesMatchSelo;
// a wrong sample fails that test rather than silently corrupting the vectors.
var curatedValid = map[selo.Kind][]string{
	selo.KindCPF:  {"529.982.247-25", "52998224725", "12345678909"},
	selo.KindCNPJ: {"39.591.842/0000-10"},
	selo.KindRG:   {"24.678.131-2", "29.465.327-2", "10.000.006-X"},
	selo.KindIE:   {"110.042.490.114", "388.108.598.269"},
}

// curatedInvalid holds authoritative, hand-picked invalid samples per kind for
// kinds where systematic digit mutation is not meaningful (group D/E/F). Each is
// confirmed invalid by selo.Validate before inclusion.
var curatedInvalid = map[selo.Kind][]string{
	selo.KindPlate:   {"AB-1234", "ABC-123", "1234567", "ABCD123", "ABC12345", "abc-12e4"},
	selo.KindPIX:     {"not-an-email", "@example.com", "+5511", "12345678900", "00000000-0000-1000-8000-000000000000", "+551199999999999999"},
	selo.KindCEP:     {"00000-000", "1234567", "123456789", "abcdefgh"},
	selo.KindPhone:   {"(00) 1234-5678", "11912345", "0099999999", "1191234567a", "999999999999"},
	selo.KindVoterID: {"000000000000", "12345678", "1234567890123"},
}

// Vectors builds the golden vector for kind from the live selo library.
func Vectors(k selo.Kind) (Vector, error) {
	plan, ok := PlanFor(k)
	if !ok {
		return Vector{}, fmt.Errorf("codegen: no plan for kind %q", k)
	}

	doc, ok := selo.Get(k)
	if !ok {
		return Vector{}, fmt.Errorf("codegen: kind %q not registered with selo", k)
	}

	v := Vector{
		Kind:              k.String(),
		Group:             plan.Group,
		UFScoped:          plan.UFScoped,
		GenerateRoundTrip: roundTripCount,
	}

	if plan.UFScoped {
		if scoped, isScoped := doc.(selo.UFScoped); isScoped {
			for _, uf := range scoped.ImplementedUFs() {
				v.UFs = append(v.UFs, uf.String())
			}
		}
	}

	valids := buildValid(k)
	for _, in := range valids {
		v.Validate = append(v.Validate, ValidateCase{Input: in, Valid: true})
	}

	for _, in := range buildInvalid(k, valids) {
		v.Validate = append(v.Validate, ValidateCase{Input: in, Valid: false})
	}

	v.Format = buildFormat(k, doc, valids)
	v.Origin = buildOrigin(k, plan, doc, valids)

	return v, nil
}

// WriteVectors writes <kind>.json for every kind into dir (created if needed).
func WriteVectors(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("codegen: mkdir %q: %w", dir, err)
	}

	for _, k := range selo.Kinds() {
		vec, err := Vectors(k)
		if err != nil {
			return err
		}

		data, err := json.MarshalIndent(vec, "", "  ")
		if err != nil {
			return fmt.Errorf("codegen: marshal %q vector: %w", k, err)
		}

		data = append(data, '\n')

		path := filepath.Join(dir, k.String()+".json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("codegen: write %q: %w", path, err)
		}
	}

	return nil
}

// buildValid returns confirmed-valid inputs for a kind: curated samples plus
// selo.Generate output, deduplicated and order-stable.
func buildValid(k selo.Kind) []string {
	seen := make(map[string]bool)

	var out []string

	add := func(s string) {
		if s == "" || seen[s] {
			return
		}

		ok, err := selo.Validate(k, s)
		if err != nil || !ok {
			return
		}

		seen[s] = true
		out = append(out, s)
	}

	for _, s := range curatedValid[k] {
		add(s)
	}
	// selo.Generate is non-deterministic; loop enough to reach the target after
	// dedup/all-equal filtering without spinning forever.
	for attempts := 0; len(out) < len(curatedValid[k])+generatedSamplesPerKind && attempts < generatedSamplesPerKind*20; attempts++ {
		g, err := selo.Generate(k)
		if err != nil {
			break
		}

		add(g)
	}

	return out
}

// buildInvalid returns confirmed-invalid inputs for a kind. It combines curated
// invalids (for pattern/composite/table kinds) with systematic mutations of the
// valid samples (for check-digit kinds), keeping only inputs selo rejects.
func buildInvalid(k selo.Kind, valids []string) []string {
	seen := make(map[string]bool)

	var out []string

	add := func(s string) {
		if s == "" || seen[s] {
			return
		}

		ok, err := selo.Validate(k, s)
		if err != nil || ok { // keep only the genuinely invalid
			return
		}

		seen[s] = true
		out = append(out, s)
	}

	for _, s := range curatedInvalid[k] {
		add(s)
	}

	for _, base := range valids {
		for _, m := range mutations(base) {
			add(m)
		}
	}

	// Guarantee a generic floor so every kind has invalid coverage even if all
	// mutations happened to stay valid (they should not, but be defensive).
	for _, junk := range []string{"", "0", "abc", "!!!!!!!!!!!", "000000000000000000"} {
		add(junk)
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i] < out[j] })

	return out
}

// mutations returns systematic single-edit corruptions of a valid input: wrong
// last digit, truncated, all-equal, and an injected letter. Each is re-checked
// by the caller against selo so accidentally-valid mutations are dropped.
func mutations(in string) []string {
	var out []string

	digits := onlyDigits(in)

	// Wrong last digit: bump the final digit (mod 10) — flips the check digit.
	if len(digits) > 0 {
		runes := []rune(in)
		for i := len(runes) - 1; i >= 0; i-- {
			if runes[i] >= '0' && runes[i] <= '9' {
				orig := runes[i]
				runes[i] = '0' + (orig-'0'+1)%10
				out = append(out, string(runes))
				runes[i] = orig

				break
			}
		}
	}

	// Truncated: drop the last character.
	if len(in) > 1 {
		out = append(out, in[:len(in)-1])
	}

	// All-equal: a run of '1's the same length as the cleaned digits.
	if len(digits) > 0 {
		out = append(out, strings.Repeat("1", len(digits)))
	}

	// Injected letter: replace the first digit with 'Z' (alphanumeric kinds keep
	// the length but break the value; numeric kinds change the cleaned length).
	if idx := strings.IndexFunc(in, func(r rune) bool { return r >= '0' && r <= '9' }); idx >= 0 {
		out = append(out, in[:idx]+"Z"+in[idx+1:])
	}

	return out
}

// buildFormat returns format vectors: every valid input mapped to its canonical
// selo.Format output, plus one error case mapped to its sentinel name.
func buildFormat(_ selo.Kind, doc selo.Document, valids []string) []FormatCase {
	var out []FormatCase

	seen := make(map[string]bool)
	for _, in := range valids {
		if seen[in] {
			continue
		}

		seen[in] = true

		formatted, err := doc.Format(in)
		if err != nil {
			continue
		}

		out = append(out, FormatCase{Input: in, Output: formatted})
	}

	// One deterministic error case: an input that selo.Format rejects. "1" is
	// the wrong length / shape for every kind.
	if _, err := doc.Format("1"); err != nil {
		out = append(out, FormatCase{Input: "1", Error: sentinelName(err)})
	}

	return out
}

// buildOrigin returns origin vectors for kinds that resolve origin, mapping each
// valid input to its selo Origin output.
func buildOrigin(_ selo.Kind, plan KindPlan, doc selo.Document, valids []string) []OriginCase {
	if plan.Origin == OriginNone {
		return nil
	}

	res, ok := doc.(selo.OriginResolver)
	if !ok {
		return nil
	}

	var out []OriginCase

	seen := make(map[string]bool)
	for _, in := range valids {
		if seen[in] {
			continue
		}

		seen[in] = true

		origin, err := res.Origin(in)
		if err != nil || origin == "" {
			continue
		}

		out = append(out, OriginCase{Input: in, Output: origin})
	}

	return out
}

// sentinelName maps a selo error to a stable sentinel name that travels in the
// vector so each language emitter can map it to its own idiom.
func sentinelName(err error) string {
	switch {
	case errors.Is(err, selo.ErrInvalidLength):
		return "ErrInvalidLength"
	case errors.Is(err, selo.ErrInvalidFormat):
		return "ErrInvalidFormat"
	case errors.Is(err, selo.ErrUFNotImplemented):
		return "ErrUFNotImplemented"
	case errors.Is(err, selo.ErrUnknownKind):
		return "ErrUnknownKind"
	default:
		return "ErrInvalid"
	}
}

// onlyDigits returns the ASCII digits of s (local copy; selo's is unexported).
func onlyDigits(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}

	return b.String()
}
