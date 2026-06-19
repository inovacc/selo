package selo

import (
	"fmt"
	"slices"
	"sync"
)

// registry holds the singleton Document per Kind. Populated from each type's init().
var (
	registryMu sync.RWMutex
	registry   = map[Kind]Document{}
)

// Register installs d as the singleton implementation for d.Kind().
// It is intended to be called from a type's init() function.
func Register(d Document) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry[d.Kind()] = d
}

// Get returns the registered Document for kind, or ok=false if none is registered.
func Get(kind Kind) (Document, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	d, ok := registry[kind]

	return d, ok
}

// Kinds returns the registered kinds in stable, sorted order.
func Kinds() []Kind {
	registryMu.RLock()

	out := make([]Kind, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}

	registryMu.RUnlock()
	slices.Sort(out)

	return out
}

// Validate dispatches validation to the registered type for kind.
// It returns ErrUnknownKind (wrapped) if kind is not registered.
func Validate(kind Kind, value string) (bool, error) {
	d, ok := Get(kind)
	if !ok {
		return false, fmt.Errorf("%q: %w", kind, ErrUnknownKind)
	}

	return d.Validate(value), nil
}

// Generate dispatches generation to the registered type for kind.
func Generate(kind Kind) (string, error) {
	d, ok := Get(kind)
	if !ok {
		return "", fmt.Errorf("%q: %w", kind, ErrUnknownKind)
	}

	return d.Generate(), nil
}

// Format dispatches formatting to the registered type for kind.
func Format(kind Kind, value string) (string, error) {
	d, ok := Get(kind)
	if !ok {
		return "", fmt.Errorf("%q: %w", kind, ErrUnknownKind)
	}

	return d.Format(value)
}

// Detect attempts to identify the Kind of value by its cleaned length, then
// confirms with that type's Validate. It generalizes the legacy ValidateDocument
// auto-detect (CPF=11 digits, CNPJ=14 alphanumeric). Returns ok=false when no
// registered type both matches the length and validates.
func Detect(value string) (Kind, bool) {
	digits := onlyDigits(value)
	switch len(digits) {
	case CpfLength:
		if ok, _ := Validate(KindCPF, value); ok {
			return KindCPF, true
		}
	case CnpjLength:
		if ok, _ := Validate(KindCNPJ, value); ok {
			return KindCNPJ, true
		}
	}

	return "", false
}
