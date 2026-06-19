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

// golden_test.go is the Task 8 snapshot gate: re-emitting the TypeScript target
// must reproduce the committed reference tree at generated/typescript byte-for-
// byte for every DETERMINISTIC file (sources, tests, scaffolding). The vector
// JSON files are intentionally excluded from the byte comparison because they
// mix non-deterministic selo.Generate() output; instead each committed vector is
// validated against the live selo library, so the snapshot stays honest without
// being brittle. If a source file drifts, regenerate with:
//
//	go run ./cmd/selo gen --lang ts --kind all --out generated/typescript

// goldenRoot is the committed reference tree, relative to this package dir.
const goldenRoot = "../../generated/typescript"

// isVectorPath reports whether p is a non-deterministic vector JSON file.
func isVectorPath(p string) bool {
	return strings.HasPrefix(filepath.ToSlash(p), "vectors/") && strings.HasSuffix(p, ".json")
}

// normalizeEOL strips carriage returns so the snapshot compares content, not
// line-ending encoding. git stores the committed reference as LF, but an
// autocrlf=true checkout (Windows) yields CRLF in the working tree while the
// emitter always writes LF; without this, the test would falsely report drift.
func normalizeEOL(b []byte) string {
	return strings.ReplaceAll(string(b), "\r", "")
}

// emitAllTS renders the full TS file set for every kind, keyed by slash path.
func emitAllTS(t *testing.T) map[string][]byte {
	t.Helper()

	out := make(map[string][]byte)

	for _, k := range selo.Kinds() {
		files, err := codegen.Generate(codegen.LangTS, k)
		require.NoErrorf(t, err, "Generate(ts, %q)", k)

		for _, f := range files {
			out[filepath.ToSlash(f.Path)] = f.Content
		}
	}

	return out
}

// TestGoldenTS_DeterministicFilesMatch asserts every re-emitted deterministic
// file equals the committed reference byte-for-byte.
func TestGoldenTS_DeterministicFilesMatch(t *testing.T) {
	emitted := emitAllTS(t)
	for path, content := range emitted {
		if isVectorPath(path) {
			continue
		}

		committed, err := os.ReadFile(filepath.Join(goldenRoot, filepath.FromSlash(path)))
		require.NoErrorf(t, err, "reading committed %s (regenerate generated/typescript?)", path)
		assert.Equalf(t, normalizeEOL(committed), normalizeEOL(content),
			"generated/typescript/%s drifted; re-run: go run ./cmd/selo gen --lang ts --kind all --out generated/typescript", path)
	}
}

// TestGoldenTS_NoExtraDeterministicFiles asserts the committed tree has no
// deterministic (non-vector, non-node_modules) files beyond what is emitted.
func TestGoldenTS_NoExtraDeterministicFiles(t *testing.T) {
	emitted := emitAllTS(t)
	err := filepath.Walk(goldenRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}

		if info.IsDir() {
			if info.Name() == "node_modules" {
				return filepath.SkipDir
			}

			return nil
		}

		rel, rerr := filepath.Rel(goldenRoot, path)
		require.NoError(t, rerr)

		rel = filepath.ToSlash(rel)
		if isVectorPath(rel) || rel == "package-lock.json" {
			return nil
		}

		_, ok := emitted[rel]
		assert.Truef(t, ok, "committed file %q is not produced by the emitter (stale?)", rel)

		return nil
	})
	require.NoError(t, err)
}

// TestGoldenTS_VectorsMatchSelo asserts every committed vector still agrees with
// the live selo library (validate/format/origin), keeping the snapshot honest.
func TestGoldenTS_VectorsMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		path := filepath.Join(goldenRoot, "vectors", k.String()+".json")
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

// goldenRootJS is the committed reference tree for JavaScript, relative to this package dir.
const goldenRootJS = "../../generated/javascript"

// emitAllJS renders the full JS file set for every kind, keyed by slash path.
func emitAllJS(t *testing.T) map[string][]byte {
	t.Helper()

	out := make(map[string][]byte)

	for _, k := range selo.Kinds() {
		files, err := codegen.Generate(codegen.LangJS, k)
		require.NoErrorf(t, err, "Generate(js, %q)", k)

		for _, f := range files {
			out[filepath.ToSlash(f.Path)] = f.Content
		}
	}

	return out
}

// TestGoldenJS_DeterministicFilesMatch asserts every re-emitted deterministic
// JS file equals the committed reference byte-for-byte.
func TestGoldenJS_DeterministicFilesMatch(t *testing.T) {
	emitted := emitAllJS(t)
	for path, content := range emitted {
		if isVectorPath(path) {
			continue
		}

		committed, err := os.ReadFile(filepath.Join(goldenRootJS, filepath.FromSlash(path)))
		require.NoErrorf(t, err, "reading committed %s (regenerate generated/javascript?)", path)
		assert.Equalf(t, normalizeEOL(committed), normalizeEOL(content),
			"generated/javascript/%s drifted; re-run: go run ./cmd/selo gen --lang js --kind all --out generated/javascript", path)
	}
}

// TestGoldenJS_NoExtraDeterministicFiles asserts the committed JS tree has no
// extra deterministic files beyond what is emitted.
func TestGoldenJS_NoExtraDeterministicFiles(t *testing.T) {
	emitted := emitAllJS(t)
	err := filepath.Walk(goldenRootJS, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}

		if info.IsDir() {
			if info.Name() == "node_modules" {
				return filepath.SkipDir
			}

			return nil
		}

		rel, rerr := filepath.Rel(goldenRootJS, path)
		require.NoError(t, rerr)

		rel = filepath.ToSlash(rel)
		if isVectorPath(rel) || rel == "package-lock.json" {
			return nil
		}

		_, ok := emitted[rel]
		assert.Truef(t, ok, "committed file %q is not produced by the JS emitter (stale?)", rel)

		return nil
	})
	require.NoError(t, err)
}

// TestGoldenJS_VectorsMatchSelo asserts every committed JS vector still agrees
// with the live selo library.
func TestGoldenJS_VectorsMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		path := filepath.Join(goldenRootJS, "vectors", k.String()+".json")
		data, err := os.ReadFile(path)
		require.NoErrorf(t, err, "reading committed vector %s", path)

		var vec codegen.Vector
		require.NoErrorf(t, json.Unmarshal(data, &vec), "unmarshal %s", path)
		assert.Equal(t, k.String(), vec.Kind)

		for _, c := range vec.Validate {
			want, verr := selo.Validate(k, c.Input)
			require.NoErrorf(t, verr, "selo.Validate(%q, %q)", k, c.Input)
			assert.Equalf(t, want, c.Valid, "committed JS vector %q input %q validity drift", k, c.Input)
		}

		doc, ok := selo.Get(k)
		require.Truef(t, ok, "selo.Get(%q)", k)

		for _, c := range vec.Format {
			out, ferr := doc.Format(c.Input)
			if c.Error != "" {
				assert.Errorf(t, ferr, "committed JS vector %q input %q expects format error", k, c.Input)
				continue
			}

			require.NoErrorf(t, ferr, "committed JS vector %q input %q format", k, c.Input)
			assert.Equalf(t, out, c.Output, "committed JS vector %q input %q format output drift", k, c.Input)
		}

		if res, hasOrigin := doc.(selo.OriginResolver); hasOrigin {
			for _, c := range vec.Origin {
				out, oerr := res.Origin(c.Input)
				require.NoErrorf(t, oerr, "committed JS vector %q input %q origin", k, c.Input)
				assert.Equalf(t, out, c.Output, "committed JS vector %q input %q origin output drift", k, c.Input)
			}
		}
	}
}
