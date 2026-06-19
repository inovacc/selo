package codegen

import (
	"fmt"
	"strings"
)

// emit_ruby_kinds.go holds the per-kind Ruby module renderers for the
// check-digit kinds (groups A, B, C). Each renders a deterministic module from
// the declarative KindPlan. The numeric kinds reuse the shared Selo::Mod11
// reducer; the irregular kinds (CNH coupled DVs, IE-SP rightmost rule, CNS
// sum-zero, CNPJ char-map) carry bespoke fragments, exactly as the TS reference.

// renderCPF emits the CPF module: two input-coupled mod-11 DVs, all-equal
// rejection, mask format, and ninth-digit origin.
func (e rubyEmitter) renderCPF(plan KindPlan) string {
	var b strings.Builder
	writeRubyHeader(&b, true)

	dv1 := rubyCheckDigitLiteral(plan.Checks[0])
	dv2 := rubyCheckDigitLiteral(plan.Checks[1])

	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module CPF
    DV1 = %s.freeze
    DV2 = %s.freeze

    # valid? reports whether value is a valid CPF (formatted or not).
    def self.valid?(value)
      d = Mod11.only_digits(value)
      return false if d.length != 11
      return false if Mod11.all_equal(d)

      digits = d.chars.map(&:to_i)
      dv1 = Mod11.compute_digit(Mod11.weighted_sum(digits[0, 9], DV1[:weights]), DV1)
      dv2 = Mod11.compute_digit(Mod11.weighted_sum(digits[0, 10], DV2[:weights]), DV2)
      dv1 == digits[9] && dv2 == digits[10]
    end

    # format renders value as XXX.XXX.XXX-XX, or raises on bad length.
    def self.format(value)
      d = Mod11.only_digits(value)
      %s if d.length != 11

      "#{d[0, 3]}.#{d[3, 3]}.#{d[6, 3]}-#{d[9, 2]}"
    end

    # origin returns the issuing region from the 9th digit, or raises.
    def self.origin(value)
      d = Mod11.only_digits(value)
      %s if d.length < 9

      region = Data::CPF_REGIONS[d[8].to_i]
      %s if region.nil?

      region
    end
  end
end
`, dv1, dv2, rubyRaise("ErrInvalidLength"),
		rubyRaise("ErrInvalidLength"), rubyRaise("ErrInvalidLength"))

	return b.String()
}

// renderSimpleNumeric emits a single-DV numeric kind (PIS): mod-11 DV over the
// first length-1 digits, all-equal rejection, and a mask format.
func (e rubyEmitter) renderSimpleNumeric(plan KindPlan, name string, length int) string {
	var b strings.Builder
	writeRubyHeader(&b, false)

	dv := rubyCheckDigitLiteral(plan.Checks[0])
	base := length - 1
	mask := rubyMaskExpr(plan.Mask, "d")

	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module %[1]s
    DV = %[2]s.freeze

    # valid? reports whether value is a valid %[1]s.
    def self.valid?(value)
      d = Mod11.only_digits(value)
      return false if d.length != %[3]d
      return false if Mod11.all_equal(d)

      digits = d.chars.map(&:to_i)
      dv = Mod11.compute_digit(Mod11.weighted_sum(digits[0, %[4]d], DV[:weights]), DV)
      dv == digits[%[4]d]
    end

    # format renders the canonical mask, or raises on bad length.
    def self.format(value)
      d = Mod11.only_digits(value)
      %[5]s if d.length != %[3]d

      %[6]s
    end
  end
end
`, name, dv, length, base, rubyRaise("ErrInvalidLength"), mask)

	return b.String()
}

// renderRenavam emits RENAVAM: single (sum*10)%11 DV, all-equal rejection, and a
// left-pad-to-11 format (no separator mask).
func (e rubyEmitter) renderRenavam(plan KindPlan) string {
	var b strings.Builder
	writeRubyHeader(&b, false)

	dv := rubyCheckDigitLiteral(plan.Checks[0])
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module Renavam
    DV = %s.freeze

    # valid? reports whether value is a valid 11-digit RENAVAM.
    def self.valid?(value)
      d = Mod11.only_digits(value)
      return false if d.length != 11
      return false if Mod11.all_equal(d)

      digits = d.chars.map(&:to_i)
      dv = Mod11.compute_digit(Mod11.weighted_sum(digits[0, 10], DV[:weights]), DV)
      dv == digits[10]
    end

    # format left-pads shorter inputs to 11 digits (no separator mask).
    def self.format(value)
      d = Mod11.only_digits(value)
      d = ('0' * (11 - d.length)) + d if d.length < 11

      d
    end
  end
end
`, dv)

	return b.String()
}

// renderCNH emits the coupled-DV CNH module (bespoke fragment per the spec Note):
// DV1 descending 9..1 (raw remainder >=10 -> DV1=0, carry offset 2); DV2
// ascending 1..9 with the offset subtracted before the mod-11 fold.
func (e rubyEmitter) renderCNH() string {
	var b strings.Builder
	writeRubyHeader(&b, false)

	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module CNH
    # cnh_check_digits computes both coupled CNH check digits over the 9-digit base.
    def self.cnh_check_digits(base)
      dsc = 0
      sum = 0
      9.times { |i| sum += base[i].to_i * (9 - i) }
      r = sum %% 11
      if r >= 10
        dv1 = 0
        dsc = 2
      else
        dv1 = r
      end
      sum = 0
      9.times { |i| sum += base[i].to_i * (1 + i) }
      r = (sum %% 11) - dsc
      r += 11 if r < 0
      dv2 = r >= 10 ? 0 : r
      [dv1, dv2]
    end

    # valid? reports whether value is a valid 11-digit CNH.
    def self.valid?(value)
      d = Mod11.only_digits(value)
      return false if d.length != 11
      return false if Mod11.all_equal(d)

      dv1, dv2 = cnh_check_digits(d[0, 9])
      dv1 == d[9].to_i && dv2 == d[10].to_i
    end

    # format returns the cleaned 11-digit CNH (no separator mask).
    def self.format(value)
      d = Mod11.only_digits(value)
      %s if d.length != 11

      d
    end
  end
end
`, rubyRaise("ErrInvalidLength"))

	return b.String()
}

// renderCNS emits the verify-only sum-zero module with prefix constraint.
func (e rubyEmitter) renderCNS(plan KindPlan) string {
	var b strings.Builder
	writeRubyHeader(&b, false)

	dv := rubyCheckDigitLiteral(plan.Checks[0])
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module CNS
    DV = %s.freeze

    # valid? reports whether value is a well-formed CNS (sum %% 11 == 0).
    def self.valid?(value)
      d = Mod11.only_digits(value)
      return false if d.length != 15
      return false if Mod11.all_equal(d)

      lead = d[0]
      return false unless %%w[1 2 7 8 9].include?(lead)

      digits = d.chars.map(&:to_i)
      Mod11.compute_digit(Mod11.weighted_sum(digits, DV[:weights]), DV) == 0
    end

    # format returns the cleaned 15-digit CNS (no separator mask).
    def self.format(value)
      d = Mod11.only_digits(value)
      %s if d.length != 15

      d
    end
  end
end
`, dv, rubyRaise("ErrInvalidLength"))

	return b.String()
}

// renderCNPJ emits the alphanumeric CNPJ module (bespoke char-map + RL-cycling
// weights per the spec Note): two DVs, last two chars numeric, all-equal reject.
func (e rubyEmitter) renderCNPJ(plan KindPlan) string {
	var b strings.Builder
	writeRubyHeader(&b, false)

	dv := rubyCheckDigitLiteral(plan.Checks[0])
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module CNPJ
    DV = %s.freeze

    # cnpj_clean uppercases and keeps only [0-9A-Z], capped at 14 chars.
    def self.cnpj_clean(value)
      out = +''
      value.each_char do |ch|
        up = ch.upcase
        if (up >= '0' && up <= '9') || (up >= 'A' && up <= 'Z')
          out << up
          break if out.length == 14
        end
      end
      out
    end

    # cnpj_dv computes one check digit over the base string (RL-cycling weights).
    def self.cnpj_dv(base)
      vals = base.chars.map { |c| Mod11.char_value(c) }
      Mod11.compute_digit(Mod11.weighted_sum(vals, DV[:weights], true), DV)
    end

    # valid? reports whether value is a valid alphanumeric CNPJ.
    def self.valid?(value)
      c = cnpj_clean(value)
      return false if c.length != 14
      return false if Mod11.all_equal(c)
      return false if c[12] < '0' || c[12] > '9'
      return false if c[13] < '0' || c[13] > '9'

      base = c[0, 12]
      dv1 = cnpj_dv(base)
      dv2 = cnpj_dv(base + dv1.to_s)
      dv1 == c[12].to_i && dv2 == c[13].to_i
    end

    # format renders value as XX.XXX.XXX/XXXX-XX, or raises on bad length.
    def self.format(value)
      c = cnpj_clean(value)
      %s if c.length != 14

      "#{c[0, 2]}.#{c[2, 3]}.#{c[5, 3]}/#{c[8, 4]}-#{c[12, 2]}"
    end
  end
end
`, dv, rubyRaise("ErrInvalidLength"))

	return b.String()
}
