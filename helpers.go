package selo

import "math/rand/v2"

// onlyDigits returns a string containing only the ASCII digit characters from s.
// Used by Detect and other helpers that need to inspect raw numeric length.
func onlyDigits(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			out = append(out, s[i])
		}
	}

	return string(out)
}

// newRand returns a fresh *rand.Rand seeded from the global source.
func newRand() *rand.Rand {
	return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
}
