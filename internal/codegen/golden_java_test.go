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

// golden_java_test.go is the M5 snapshot gate (mirror of golden_test.go for the
// TypeScript target): re-emitting the Java target must reproduce the committed
// reference tree at generated/java byte-for-byte for every DETERMINISTIC file
// (sources, tests, scaffolding). The vector JSON files are intentionally
// excluded from the byte comparison because they mix non-deterministic
// selo.Generate() output; instead each committed vector is validated against the
// live selo library, so the snapshot stays honest without being brittle. If a
// source file drifts, regenerate with:
//
//	go run ./cmd/selo gen --lang java --kind all --out generated/java

// goldenJavaRoot is the committed reference tree, relative to this package dir.
const goldenJavaRoot = "../../generated/java"

// isJavaVectorPath reports whether p is a non-deterministic vector JSON file.
func isJavaVectorPath(p string) bool {
	return strings.HasPrefix(filepath.ToSlash(p), "vectors/") && strings.HasSuffix(p, ".json")
}

// normalizeJavaEOL strips carriage returns so the snapshot compares content, not
// line-ending encoding (autocrlf=true checkouts yield CRLF working-tree files
// while the emitter always writes LF).
func normalizeJavaEOL(b []byte) string {
	return strings.ReplaceAll(string(b), "\r", "")
}

// emitAllJava renders the full Java file set for every kind, keyed by slash path.
func emitAllJava(t *testing.T) map[string][]byte {
	t.Helper()

	out := make(map[string][]byte)

	for _, k := range selo.Kinds() {
		files, err := codegen.Generate(codegen.LangJava, k)
		require.NoErrorf(t, err, "Generate(java, %q)", k)

		for _, f := range files {
			out[filepath.ToSlash(f.Path)] = f.Content
		}
	}

	return out
}

// TestGoldenJava_DeterministicFilesMatch asserts every re-emitted deterministic
// file equals the committed reference byte-for-byte.
func TestGoldenJava_DeterministicFilesMatch(t *testing.T) {
	emitted := emitAllJava(t)
	for path, content := range emitted {
		if isJavaVectorPath(path) {
			continue
		}

		committed, err := os.ReadFile(filepath.Join(goldenJavaRoot, filepath.FromSlash(path)))
		require.NoErrorf(t, err, "reading committed %s (regenerate generated/java?)", path)
		assert.Equalf(t, normalizeJavaEOL(committed), normalizeJavaEOL(content),
			"generated/java/%s drifted; re-run: go run ./cmd/selo gen --lang java --kind all --out generated/java", path)
	}
}

// TestGoldenJava_NoExtraDeterministicFiles asserts the committed tree has no
// deterministic (non-vector, non-build-output) files beyond what is emitted.
func TestGoldenJava_NoExtraDeterministicFiles(t *testing.T) {
	emitted := emitAllJava(t)
	err := filepath.Walk(goldenJavaRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}

		if info.IsDir() {
			if info.Name() == "target" {
				return filepath.SkipDir
			}

			return nil
		}

		rel, rerr := filepath.Rel(goldenJavaRoot, path)
		require.NoError(t, rerr)

		rel = filepath.ToSlash(rel)
		if isJavaVectorPath(rel) {
			return nil
		}

		_, ok := emitted[rel]
		assert.Truef(t, ok, "committed file %q is not produced by the emitter (stale?)", rel)

		return nil
	})
	require.NoError(t, err)
}

// TestGoldenJava_VectorsMatchSelo asserts every committed vector still agrees
// with the live selo library (validate/format/origin), keeping the snapshot
// honest.
func TestGoldenJava_VectorsMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		path := filepath.Join(goldenJavaRoot, "vectors", k.String()+".json")
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
