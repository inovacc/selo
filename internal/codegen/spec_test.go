package codegen_test

import (
	"testing"

	"github.com/inovacc/selo"
	"github.com/inovacc/selo/internal/codegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlans_CoverAllKinds asserts there is a codegen plan for every registered
// selo kind. This is the Task 1 gate: the plan registry must stay in lockstep
// with the document registry.
func TestPlans_CoverAllKinds(t *testing.T) {
	for _, k := range selo.Kinds() {
		_, ok := codegen.PlanFor(k)
		assert.Truef(t, ok, "no codegen plan for kind %q", k)
	}
}

// TestPlans_NoExtraKinds asserts the plan registry has no entries for kinds that
// are not registered with selo (catches typos / stale entries).
func TestPlans_NoExtraKinds(t *testing.T) {
	known := make(map[selo.Kind]bool)
	for _, k := range selo.Kinds() {
		known[k] = true
	}
	for k := range codegen.Plans {
		assert.Truef(t, known[k], "codegen plan for unknown kind %q", k)
	}
	assert.Len(t, codegen.Plans, len(selo.Kinds()), "plan count must equal kind count")
}

// TestPlans_ShapeIsValid asserts every plan is well-formed: either it has a mask
// or lengths (groups A/B/C/F), or it is a pattern kind (group D/E with Pattern
// set). Group is always one of A–F.
func TestPlans_ShapeIsValid(t *testing.T) {
	validGroups := map[string]bool{"A": true, "B": true, "C": true, "D": true, "E": true, "F": true}
	for _, k := range selo.Kinds() {
		p, ok := codegen.PlanFor(k)
		require.Truef(t, ok, "missing plan for %q", k)

		assert.Truef(t, validGroups[p.Group], "kind %q has invalid group %q", k, p.Group)
		assert.Equalf(t, k, p.Kind, "kind %q plan has mismatched Kind field %q", k, p.Kind)

		switch p.Group {
		case "D", "E":
			assert.NotEmptyf(t, p.Pattern, "pattern kind %q must set Pattern", k)
		default:
			hasShape := len(p.Lengths) > 0 || p.Mask != ""
			assert.Truef(t, hasShape, "kind %q must set Lengths or Mask", k)
		}
	}
}

// TestPlans_GroupAssignments pins the design §3 taxonomy so a regrouping is a
// deliberate, reviewed change.
func TestPlans_GroupAssignments(t *testing.T) {
	want := map[selo.Kind]string{
		selo.KindCPF:     "A",
		selo.KindPIS:     "A",
		selo.KindRenavam: "A",
		selo.KindCNH:     "A",
		selo.KindRG:      "A",
		selo.KindIE:      "A",
		selo.KindCNS:     "B",
		selo.KindCNPJ:    "C",
		selo.KindPlate:   "D",
		selo.KindPIX:     "E",
		selo.KindCEP:     "F",
		selo.KindPhone:   "F",
		selo.KindVoterID: "F",
	}
	for k, g := range want {
		p, ok := codegen.PlanFor(k)
		require.Truef(t, ok, "missing plan for %q", k)
		assert.Equalf(t, g, p.Group, "kind %q group", k)
	}
}

// TestPlans_CheckDigitWeightsTranscribed spot-checks a few high-value weight
// vectors against the known selo source values, guarding against transcription
// drift.
func TestPlans_CheckDigitWeightsTranscribed(t *testing.T) {
	cpf, _ := codegen.PlanFor(selo.KindCPF)
	require.Len(t, cpf.Checks, 2)
	assert.Equal(t, []int{10, 9, 8, 7, 6, 5, 4, 3, 2}, cpf.Checks[0].Weights)
	assert.Equal(t, []int{11, 10, 9, 8, 7, 6, 5, 4, 3, 2}, cpf.Checks[1].Weights)
	assert.True(t, cpf.Checks[0].MultiplyBy10)

	cnpj, _ := codegen.PlanFor(selo.KindCNPJ)
	assert.True(t, cnpj.CharMap)
	require.Len(t, cnpj.Checks, 2)
	assert.Equal(t, []int{2, 3, 4, 5, 6, 7, 8, 9}, cnpj.Checks[0].Weights)
	assert.Equal(t, []int{0, 1}, cnpj.Checks[0].RemainderTo0)

	rg, _ := codegen.PlanFor(selo.KindRG)
	require.Len(t, rg.Checks, 1)
	assert.Equal(t, []int{2, 3, 4, 5, 6, 7, 8, 9}, rg.Checks[0].Weights)
	assert.Equal(t, codegen.DVElevenMinus, rg.Checks[0].Rule)
	assert.Equal(t, 10, rg.Checks[0].EncodeXAt)
	assert.Equal(t, 11, rg.Checks[0].EncodeZeroAt)

	cns, _ := codegen.PlanFor(selo.KindCNS)
	require.Len(t, cns.Checks, 1)
	assert.Equal(t, codegen.DVSumZero, cns.Checks[0].Rule)
	assert.Len(t, cns.Checks[0].Weights, 15)

	ie, _ := codegen.PlanFor(selo.KindIE)
	require.Len(t, ie.Checks, 2)
	assert.Equal(t, codegen.DVRightmostDigit, ie.Checks[0].Rule)
	assert.Equal(t, []int{1, 3, 4, 5, 6, 7, 8, 10}, ie.Checks[0].Weights)
}

// TestDVRuleString keeps the rule identifiers stable (they appear in vectors and
// emitter output).
func TestDVRuleString(t *testing.T) {
	cases := map[codegen.DVRule]string{
		codegen.DVElevenMinus:    "eleven_minus",
		codegen.DVModRemainder:   "mod_remainder",
		codegen.DVRightmostDigit: "rightmost_digit",
		codegen.DVSumZero:        "sum_zero",
	}
	for r, want := range cases {
		assert.Equal(t, want, r.String())
	}
}
