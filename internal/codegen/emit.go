package codegen

import (
	"fmt"
	"sort"

	"github.com/inovacc/selo"
)

// emit.go holds the language-emitter registry that the `selo gen` CLI and the
// MCP generate_code tool drive. In M1 the registry is intentionally EMPTY: the
// surfaces (CLI flags, help text, MCP tool) are wired and list the supported
// languages and kinds, but no emitter is registered yet, so a generation
// request reports "emitter not yet registered" cleanly. M2+ register real
// per-language emitters via Register.

// Lang is a target language identifier for code generation.
type Lang string

// The five supported target languages (design spec §1).
const (
	LangTS     Lang = "ts"
	LangJS     Lang = "js"
	LangRuby   Lang = "ruby"
	LangJava   Lang = "java"
	LangCSharp Lang = "csharp"
)

// supportedLangs is the fixed, ordered set of languages the generator targets.
// It is independent of which emitters are registered so the CLI/MCP can always
// advertise the roadmap even before an emitter exists.
var supportedLangs = []Lang{LangTS, LangJS, LangRuby, LangJava, LangCSharp}

// File is a single generated artifact: a path relative to the output root and
// its contents.
type File struct {
	Path    string
	Content []byte
}

// Emitter renders the generated file set for one kind in one language. M2+
// implementations consume the KindPlan, the extracted data tables, and the
// golden Vector. M1 ships no implementations.
type Emitter interface {
	// Lang reports the language this emitter targets.
	Lang() Lang
	// Emit renders the module + test + vector files for kind.
	Emit(kind selo.Kind, plan KindPlan, vec Vector) ([]File, error)
}

// emitters is the registry of language emitters, keyed by Lang. Empty in M1.
var emitters = map[Lang]Emitter{}

// Register installs e as the emitter for e.Lang(). Intended to be called from a
// language emitter's init() in M2+.
func Register(e Emitter) {
	emitters[e.Lang()] = e
}

// SupportedLangs returns the ordered set of target languages (independent of
// registration).
func SupportedLangs() []Lang {
	out := make([]Lang, len(supportedLangs))
	copy(out, supportedLangs)

	return out
}

// SupportedLangStrings returns the supported languages as plain strings, for
// help text and flag validation.
func SupportedLangStrings() []string {
	out := make([]string, 0, len(supportedLangs))
	for _, l := range supportedLangs {
		out = append(out, string(l))
	}

	return out
}

// IsSupportedLang reports whether s names one of the supported target languages.
func IsSupportedLang(s string) bool {
	for _, l := range supportedLangs {
		if string(l) == s {
			return true
		}
	}

	return false
}

// KindStrings returns all plan kinds as strings in stable sorted order, for help
// text and validation.
func KindStrings() []string {
	out := make([]string, 0, len(Plans))
	for k := range Plans {
		out = append(out, k.String())
	}

	sort.Strings(out)

	return out
}

// EmitterFor returns the registered emitter for lang, or ok=false when none is
// registered (the M1 state for every language).
func EmitterFor(lang Lang) (Emitter, bool) {
	e, ok := emitters[lang]
	return e, ok
}

// Generate renders the file set for the given language and kind. It returns a
// clear, stable error when the language is unsupported or has no registered
// emitter yet (the M1 path), so the CLI and MCP tool can surface it cleanly.
func Generate(lang Lang, kind selo.Kind) ([]File, error) {
	if !IsSupportedLang(string(lang)) {
		return nil, fmt.Errorf("codegen: unsupported language %q (supported: %v)", lang, SupportedLangStrings())
	}

	plan, ok := PlanFor(kind)
	if !ok {
		return nil, fmt.Errorf("codegen: unknown kind %q", kind)
	}

	emitter, ok := EmitterFor(lang)
	if !ok {
		return nil, fmt.Errorf("codegen: emitter for %q not yet registered", lang)
	}

	vec, err := Vectors(kind)
	if err != nil {
		return nil, err
	}

	return emitter.Emit(kind, plan, vec)
}
