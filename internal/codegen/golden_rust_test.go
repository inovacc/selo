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

// golden_rust_test.go is the Rust snapshot gate, mirroring golden_php_test.go:
// re-emitting the Rust target must reproduce the committed reference tree at
// generated/rust byte-for-byte for every DETERMINISTIC file (sources, tests,
// scaffolding). The vector JSON files are excluded from the byte comparison
// because they mix non-deterministic selo.Generate() output; instead each
// committed vector is validated against the live selo library, so the snapshot
// stays honest without being brittle. If a source file drifts, regenerate with:
//
//	go run ./cmd/selo gen --lang rust --kind all --out generated/rust

// goldenRustRoot is the committed reference tree, relative to this package dir.
const goldenRustRoot = "../../generated/rust"

// emitAllRust renders the full Rust file set for every kind, keyed by slash path.
func emitAllRust(t *testing.T) map[string][]byte {
	t.Helper()

	out := make(map[string][]byte)

	for _, k := range selo.Kinds() {
		files, err := codegen.Generate(codegen.LangRust, k)
		require.NoErrorf(t, err, "Generate(rust, %q)", k)

		for _, f := range files {
			out[filepath.ToSlash(f.Path)] = f.Content
		}
	}

	return out
}

// TestGoldenRust_DeterministicFilesMatch asserts every re-emitted deterministic
// file equals the committed reference byte-for-byte.
func TestGoldenRust_DeterministicFilesMatch(t *testing.T) {
	emitted := emitAllRust(t)
	for path, content := range emitted {
		if isVectorPath(path) {
			continue
		}

		committed, err := os.ReadFile(filepath.Join(goldenRustRoot, filepath.FromSlash(path)))
		require.NoErrorf(t, err, "reading committed %s (regenerate generated/rust?)", path)
		assert.Equalf(t, normalizeEOL(committed), normalizeEOL(content),
			"generated/rust/%s drifted; re-run: go run ./cmd/selo gen --lang rust --kind all --out generated/rust", path)
	}
}

// TestGoldenRust_NoExtraDeterministicFiles asserts the committed tree has no
// deterministic (non-vector, non-toolchain) files beyond what is emitted.
func TestGoldenRust_NoExtraDeterministicFiles(t *testing.T) {
	emitted := emitAllRust(t)
	err := filepath.Walk(goldenRustRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}

		if info.IsDir() {
			// Skip the cargo build output dir (created by `cargo test` via
			// `task gen:verify:rust`); it is gitignored and not emitter output,
			// mirroring how the other gates skip node_modules/bin/obj/vendor.
			if info.Name() == "target" {
				return filepath.SkipDir
			}

			return nil
		}

		rel, rerr := filepath.Rel(goldenRustRoot, path)
		require.NoError(t, rerr)

		rel = filepath.ToSlash(rel)
		if isVectorPath(rel) {
			return nil
		}
		// Cargo.lock is resolved by cargo, not emitted (a library crate does not
		// commit it); it is a toolchain artifact (gitignored), not emitter output.
		if rel == "Cargo.lock" {
			return nil
		}

		_, ok := emitted[rel]
		assert.Truef(t, ok, "committed file %q is not produced by the emitter (stale?)", rel)

		return nil
	})
	require.NoError(t, err)
}

// TestGoldenRust_VectorsMatchSelo asserts every committed vector still agrees
// with the live selo library (validate/format/origin), keeping the snapshot
// honest.
func TestGoldenRust_VectorsMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		path := filepath.Join(goldenRustRoot, "vectors", k.String()+".json")
		data, err := os.ReadFile(path)
		require.NoErrorf(t, err, "reading committed vector %s", path)

		var vec codegen.Vector
		require.NoErrorf(t, json.Unmarshal(data, &vec), "unmarshal %s", path)
		assert.Equal(t, k.String(), vec.Kind)

		for _, c := range vec.Validate {
			want, verr := selo.Validate(k, c.Input)
			require.NoErrorf(t, verr, "selo.Validate(%q, %q)", k, c.Input)
			assert.Equalf(t, want, c.Valid, "committed vector %q input %q validity drift", k, c.Input)
		}

		doc, ok := selo.Get(k)
		require.Truef(t, ok, "selo.Get(%q)", k)

		for _, c := range vec.Format {
			out, ferr := doc.Format(c.Input)
			if c.Error != "" {
				assert.Errorf(t, ferr, "committed vector %q input %q expects format error", k, c.Input)
				continue
			}

			require.NoErrorf(t, ferr, "committed vector %q input %q format", k, c.Input)
			assert.Equalf(t, out, c.Output, "committed vector %q input %q format output drift", k, c.Input)
		}

		if res, hasOrigin := doc.(selo.OriginResolver); hasOrigin {
			for _, c := range vec.Origin {
				out, oerr := res.Origin(c.Input)
				require.NoErrorf(t, oerr, "committed vector %q input %q origin", k, c.Input)
				assert.Equalf(t, out, c.Output, "committed vector %q input %q origin output drift", k, c.Input)
			}
		}
	}
}
