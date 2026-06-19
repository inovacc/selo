package codegen_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// golden_ruby_test.go is the M4 snapshot gate: re-emitting the Ruby target must
// reproduce the committed reference tree at generated/ruby byte-for-byte for
// every DETERMINISTIC file (modules, tests, scaffolding). The vector JSON files
// are excluded from the byte comparison because they mix non-deterministic
// selo.Generate() output; instead each committed vector is validated against the
// live selo library, so the snapshot stays honest without being brittle. If a
// source file drifts, regenerate with:
//
//	go run ./cmd/selo gen --lang ruby --kind all --out generated/ruby

// goldenRubyRoot is the committed Ruby reference tree, relative to this package.
const goldenRubyRoot = "../../generated/ruby"

// isVectorPathRuby reports whether p is a non-deterministic vector JSON file.
func isVectorPathRuby(p string) bool {
	return strings.HasPrefix(filepath.ToSlash(p), "vectors/") && strings.HasSuffix(p, ".json")
}

// normalizeEOLRuby strips carriage returns so the snapshot compares content, not
// line-ending encoding (an autocrlf checkout yields CRLF while the emitter
// always writes LF).
func normalizeEOLRuby(b []byte) string {
	return strings.ReplaceAll(string(b), "\r", "")
}

// emitAllRuby renders the full Ruby file set for every kind, keyed by slash path.
func emitAllRuby(t *testing.T) map[string][]byte {
	t.Helper()

	out := make(map[string][]byte)

	for _, k := range selo.Kinds() {
		files, err := codegen.Generate(codegen.LangRuby, k)
		require.NoErrorf(t, err, "Generate(ruby, %q)", k)

		for _, f := range files {
			out[filepath.ToSlash(f.Path)] = f.Content
		}
	}

	return out
}

// TestGoldenRuby_DeterministicFilesMatch asserts every re-emitted deterministic
// file equals the committed reference byte-for-byte.
func TestGoldenRuby_DeterministicFilesMatch(t *testing.T) {
	emitted := emitAllRuby(t)
	for path, content := range emitted {
		if isVectorPathRuby(path) {
			continue
		}

		committed, err := os.ReadFile(filepath.Join(goldenRubyRoot, filepath.FromSlash(path)))
		require.NoErrorf(t, err, "reading committed %s (regenerate generated/ruby?)", path)
		assert.Equalf(t, normalizeEOLRuby(committed), normalizeEOLRuby(content),
			"generated/ruby/%s drifted; re-run: go run ./cmd/selo gen --lang ruby --kind all --out generated/ruby", path)
	}
}

// TestGoldenRuby_NoExtraDeterministicFiles asserts the committed tree has no
// deterministic (non-vector) files beyond what is emitted.
func TestGoldenRuby_NoExtraDeterministicFiles(t *testing.T) {
	emitted := emitAllRuby(t)
	err := filepath.Walk(goldenRubyRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}

		if info.IsDir() {
			return nil
		}

		rel, rerr := filepath.Rel(goldenRubyRoot, path)
		require.NoError(t, rerr)

		rel = filepath.ToSlash(rel)
		if isVectorPathRuby(rel) {
			return nil
		}

		_, ok := emitted[rel]
		assert.Truef(t, ok, "committed file %q is not produced by the emitter (stale?)", rel)

		return nil
	})
	require.NoError(t, err)
}

// TestGoldenRuby_VectorsMatchSelo asserts every committed Ruby vector still
// agrees with the live selo library (validate/format/origin).
func TestGoldenRuby_VectorsMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		path := filepath.Join(goldenRubyRoot, "vectors", k.String()+".json")
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
