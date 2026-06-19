package codegen_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVectors_ValidateCasesMatchSelo is the Task 2 gate: every emitted validate
// case's `valid` flag must equal selo.Validate(kind, input). This guarantees the
// vectors never disagree with the source of truth.
func TestVectors_ValidateCasesMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		vec, err := codegen.Vectors(k)
		require.NoErrorf(t, err, "Vectors(%q)", k)

		for _, c := range vec.Validate {
			want, verr := selo.Validate(k, c.Input)
			require.NoErrorf(t, verr, "selo.Validate(%q, %q)", k, c.Input)
			assert.Equalf(t, want, c.Valid,
				"kind %q input %q: vector says valid=%v but selo says %v", k, c.Input, c.Valid, want)
		}
	}
}

// TestVectors_MinimumCoverage asserts at least 4 valid and 4 invalid cases per
// kind, per the Task 2 requirement.
func TestVectors_MinimumCoverage(t *testing.T) {
	for _, k := range selo.Kinds() {
		vec, err := codegen.Vectors(k)
		require.NoErrorf(t, err, "Vectors(%q)", k)

		var valid, invalid int

		for _, c := range vec.Validate {
			if c.Valid {
				valid++
			} else {
				invalid++
			}
		}

		assert.GreaterOrEqualf(t, valid, 4, "kind %q needs >=4 valid cases, got %d", k, valid)
		assert.GreaterOrEqualf(t, invalid, 4, "kind %q needs >=4 invalid cases, got %d", k, invalid)
	}
}

// TestVectors_FormatCasesMatchSelo asserts every format vector's output (or error
// presence) matches selo.Format.
func TestVectors_FormatCasesMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		vec, err := codegen.Vectors(k)
		require.NoErrorf(t, err, "Vectors(%q)", k)
		doc, ok := selo.Get(k)
		require.Truef(t, ok, "selo.Get(%q)", k)

		for _, c := range vec.Format {
			out, ferr := doc.Format(c.Input)
			if c.Error != "" {
				assert.Errorf(t, ferr, "kind %q input %q: vector expects error %q", k, c.Input, c.Error)
				continue
			}

			require.NoErrorf(t, ferr, "kind %q input %q: vector has output but selo errored", k, c.Input)
			assert.Equalf(t, out, c.Output, "kind %q input %q format output", k, c.Input)
		}
	}
}

// TestVectors_OriginCasesMatchSelo asserts origin vectors match selo's resolver
// and that origin-bearing kinds actually emit origin cases.
func TestVectors_OriginCasesMatchSelo(t *testing.T) {
	originKinds := map[selo.Kind]bool{
		selo.KindCPF: true, selo.KindCEP: true, selo.KindPhone: true, selo.KindVoterID: true,
	}

	for _, k := range selo.Kinds() {
		vec, err := codegen.Vectors(k)
		require.NoErrorf(t, err, "Vectors(%q)", k)
		doc, _ := selo.Get(k)

		if originKinds[k] {
			assert.NotEmptyf(t, vec.Origin, "kind %q should emit origin cases", k)
		}

		res, hasOrigin := doc.(selo.OriginResolver)
		for _, c := range vec.Origin {
			require.Truef(t, hasOrigin, "kind %q has origin cases but no resolver", k)

			out, oerr := res.Origin(c.Input)
			require.NoErrorf(t, oerr, "kind %q input %q origin", k, c.Input)
			assert.Equalf(t, out, c.Output, "kind %q input %q origin output", k, c.Input)
		}
	}
}

// TestVectors_AllKindsProduced asserts a vector is produced for every kind and
// is non-trivially populated.
func TestVectors_AllKindsProduced(t *testing.T) {
	require.Len(t, selo.Kinds(), 13, "expected 13 registered kinds")

	for _, k := range selo.Kinds() {
		vec, err := codegen.Vectors(k)
		require.NoErrorf(t, err, "Vectors(%q)", k)
		assert.Equal(t, k.String(), vec.Kind)
		assert.NotEmpty(t, vec.Validate, "kind %q has no validate cases", k)
		assert.Equal(t, 100, vec.GenerateRoundTrip)
	}
}

// TestVectors_UFScopedFlagged asserts RG and IE carry the ufScoped flag and list
// their implemented UFs.
func TestVectors_UFScopedFlagged(t *testing.T) {
	for _, k := range []selo.Kind{selo.KindRG, selo.KindIE} {
		vec, err := codegen.Vectors(k)
		require.NoErrorf(t, err, "Vectors(%q)", k)
		assert.Truef(t, vec.UFScoped, "kind %q should be ufScoped", k)
		assert.NotEmptyf(t, vec.UFs, "kind %q should list implemented UFs", k)
	}
}

// TestWriteVectors_AllKinds asserts WriteVectors emits one parseable JSON file
// per kind, whose validate cases still agree with selo.
func TestWriteVectors_AllKinds(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, codegen.WriteVectors(dir))

	for _, k := range selo.Kinds() {
		path := filepath.Join(dir, k.String()+".json")
		data, err := os.ReadFile(path)
		require.NoErrorf(t, err, "reading %s", path)

		var vec codegen.Vector
		require.NoErrorf(t, json.Unmarshal(data, &vec), "unmarshal %s", path)
		assert.Equal(t, k.String(), vec.Kind)

		for _, c := range vec.Validate {
			want, verr := selo.Validate(k, c.Input)
			require.NoError(t, verr)
			assert.Equalf(t, want, c.Valid, "written %q input %q", k, c.Input)
		}
	}
}
