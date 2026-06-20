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

// golden_php_test.go is the PHP snapshot gate, mirroring golden_csharp_test.go:
// re-emitting the PHP target must reproduce the committed reference tree at
// generated/php byte-for-byte for every DETERMINISTIC file (sources, tests,
// scaffolding). The vector JSON files are excluded from the byte comparison
// because they mix non-deterministic selo.Generate() output; instead each
// committed vector is validated against the live selo library, so the snapshot
// stays honest without being brittle. If a source file drifts, regenerate with:
//
//	go run ./cmd/selo gen --lang php --kind all --out generated/php

// goldenPHPRoot is the committed reference tree, relative to this package dir.
const goldenPHPRoot = "../../generated/php"

// emitAllPHP renders the full PHP file set for every kind, keyed by slash path.
func emitAllPHP(t *testing.T) map[string][]byte {
	t.Helper()

	out := make(map[string][]byte)

	for _, k := range selo.Kinds() {
		files, err := codegen.Generate(codegen.LangPHP, k)
		require.NoErrorf(t, err, "Generate(php, %q)", k)

		for _, f := range files {
			out[filepath.ToSlash(f.Path)] = f.Content
		}
	}

	return out
}

// TestGoldenPHP_DeterministicFilesMatch asserts every re-emitted deterministic
// file equals the committed reference byte-for-byte.
func TestGoldenPHP_DeterministicFilesMatch(t *testing.T) {
	emitted := emitAllPHP(t)
	for path, content := range emitted {
		if isVectorPath(path) {
			continue
		}

		committed, err := os.ReadFile(filepath.Join(goldenPHPRoot, filepath.FromSlash(path)))
		require.NoErrorf(t, err, "reading committed %s (regenerate generated/php?)", path)
		assert.Equalf(t, normalizeEOL(committed), normalizeEOL(content),
			"generated/php/%s drifted; re-run: go run ./cmd/selo gen --lang php --kind all --out generated/php", path)
	}
}

// TestGoldenPHP_NoExtraDeterministicFiles asserts the committed tree has no
// deterministic (non-vector, non-toolchain) files beyond what is emitted.
func TestGoldenPHP_NoExtraDeterministicFiles(t *testing.T) {
	emitted := emitAllPHP(t)
	err := filepath.Walk(goldenPHPRoot, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}

		if info.IsDir() {
			// Skip Composer-installed dependencies (created by `composer install`
			// via `task gen:verify:php`); they are gitignored and not emitter
			// output, mirroring how the other gates skip node_modules/bin/obj.
			if info.Name() == "vendor" {
				return filepath.SkipDir
			}

			return nil
		}

		rel, rerr := filepath.Rel(goldenPHPRoot, path)
		require.NoError(t, rerr)

		rel = filepath.ToSlash(rel)
		if isVectorPath(rel) {
			return nil
		}
		// composer.lock and the PHPUnit result cache are toolchain artifacts
		// (gitignored), not emitter output.
		if rel == "composer.lock" || rel == ".phpunit.result.cache" {
			return nil
		}

		_, ok := emitted[rel]
		assert.Truef(t, ok, "committed file %q is not produced by the emitter (stale?)", rel)

		return nil
	})
	require.NoError(t, err)
}

// TestGoldenPHP_VectorsMatchSelo asserts every committed vector still agrees with
// the live selo library (validate/format/origin), keeping the snapshot honest.
func TestGoldenPHP_VectorsMatchSelo(t *testing.T) {
	for _, k := range selo.Kinds() {
		path := filepath.Join(goldenPHPRoot, "vectors", k.String()+".json")
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
