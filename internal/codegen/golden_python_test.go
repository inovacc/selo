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

// golden_python_test.go is the M6 snapshot gate: re-emitting the Python target
// must reproduce the committed reference tree at generated/python byte-for-byte
// for every DETERMINISTIC file (modules, tests, scaffolding). The vector JSON
// files are excluded from the byte comparison because they mix non-deterministic
// selo.Generate() output; instead each committed vector is validated against the
// live selo library, so the snapshot stays honest without being brittle. If a
// source file drifts, regenerate with:
//
//	go run ./cmd/selo gen --lang python --kind all --out generated/python

// goldenPythonRoot is the committed Python reference tree, relative to this package.
const goldenPythonRoot = "../../generated/python"

// isVectorPathPython reports whether p is a non-deterministic vector JSON file.
func isVectorPathPython(p string) bool {
	return strings.HasPrefix(filepath.ToSlash(p), "vectors/") && strings.HasSuffix(p, ".json")
}

// normalizeEOLPython strips carriage returns so the snapshot compares content,
// not line-ending encoding (an autocrlf checkout yields CRLF while the emitter
// always writes LF).
func normalizeEOLPython(b []byte) string {
	return strings.ReplaceAll(string(b), "\r", "")
}

// emitAllPython renders the full Python file set for every kind, keyed by slash path.
func emitAllPython(t *testing.T) map[string][]byte {
	t.Helper()

	out := make(map[string][]byte)

	for _, k := range selo.Kinds() {
		files, err := codegen.Generate(codegen.LangPython, k)
		require.NoErrorf(t, err, "Generate(python, %q)", k)

		for _, f := range files {
			out[filepath.ToSlash(f.Path)] = f.Content
		}
	}

	return out
}

// TestGoldenPython_DeterministicFilesMatch asserts every re-emitted deterministic
// file equals the committed reference byte-for-byte.
func TestGoldenPython_DeterministicFilesMatch(t *testing.T) {
	emitted := emitAllPython(t)
	for path, content := range emitted {
		if isVectorPathPython(path) {
			continue
		}

		committed, err := os.ReadFile(filepath.Join(goldenPythonRoot, filepath.FromSlash(path)))
		require.NoErrorf(t, err, "reading committed %s (regenerate generated/python?)", path)
		assert.Equalf(t, normalizeEOLPython(committed), normalizeEOLPython(content),
			"generated/python/%s drifted; re-run: go run ./cmd/selo gen --lang python --kind all --out generated/python", path)
	}
}

// TestGoldenPython_NoExtraDeterministicFiles asserts the committed tree has no
// deterministic (non-vector) files beyond what is emitted.
func TestGoldenPython_NoExtraDeterministicFiles(t *testing.T) {
	emitted := emitAllPython(t)
	err := filepath.Walk(goldenPythonRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}

		if info.IsDir() {
			// Skip Python toolchain artifacts (created by `pip install -e .` /
			// pytest, e.g. via `task gen:verify:python`); they are gitignored and
			// not emitter output, mirroring how the TS/JS gates skip node_modules.
			name := info.Name()
			if name == "__pycache__" || name == ".pytest_cache" || name == ".venv" ||
				name == "build" || strings.HasSuffix(name, ".egg-info") {
				return filepath.SkipDir
			}

			return nil
		}

		rel, rerr := filepath.Rel(goldenPythonRoot, path)
		require.NoError(t, rerr)

		rel = filepath.ToSlash(rel)
		if isVectorPathPython(rel) {
			return nil
		}

		_, ok := emitted[rel]
		assert.Truef(t, ok, "committed file %q is not produced by the emitter (stale?)", rel)

		return nil
	})
	require.NoError(t, err)
}

// TestGoldenPython_VectorsMatchSelo asserts every committed Python vector still
// agrees with the live selo library (validate/format/origin).
func TestGoldenPython_VectorsMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		path := filepath.Join(goldenPythonRoot, "vectors", k.String()+".json")
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
