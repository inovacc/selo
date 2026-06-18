package brdoc

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors_DistinctAndWrappable(t *testing.T) {
	all := []error{
		ErrInvalidLength,
		ErrInvalidFormat,
		ErrUnknownKind,
		ErrUnsupported,
		ErrUFNotImplemented,
	}
	for i, e := range all {
		if e == nil {
			t.Fatalf("sentinel at index %d is nil", i)
		}
		// each sentinel must remain identifiable through %w wrapping
		wrapped := fmt.Errorf("ctx: %w", e)
		if !errors.Is(wrapped, e) {
			t.Fatalf("errors.Is failed to unwrap sentinel index %d", i)
		}
	}
	// distinctness: no two sentinels are Is-equal to each other
	for i := range all {
		for j := range all {
			if i != j && errors.Is(all[i], all[j]) {
				t.Fatalf("sentinels %d and %d are not distinct", i, j)
			}
		}
	}
}
