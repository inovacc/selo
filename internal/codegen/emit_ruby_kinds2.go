package codegen

import (
	"fmt"
	"strings"

	"github.com/inovacc/selo"
)

// emit_ruby_kinds2.go holds the remaining per-kind Ruby module renderers
// (RG/IE UF-scoped, plate/pix regex, cep/phone table lookup, voter dual-DV) and
// the per-kind Minitest renderer. Every algorithm is translated verbatim from
// the TS reference.

// renderRG emits the UF-scoped RG module: 8 base digits + 1 check char
// (10->'X', 11->'0'); SP and RJ share the algorithm.
func (e rubyEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	writeRubyHeader(&b, false)

	dv := rubyCheckDigitLiteral(plan.Checks[0])
	ufs := rubyStringArray([]string{"SP", "RJ"})
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module RG
    DV = %s.freeze

    # RG_UFS lists the implemented federative units (shared SP/RJ algorithm).
    RG_UFS = %s.freeze

    # rg_parse strips formatting and returns [base_digits, check] or nil.
    def self.rg_parse(value)
      cleaned = +''
      value.each_char do |ch|
        cleaned << ch if (ch >= '0' && ch <= '9') || ch == 'X' || ch == 'x'
      end
      return nil if cleaned.length != 9

      last = cleaned[8]
      if last == 'X' || last == 'x'
        check = 10
      elsif last == '0'
        check = 11
      elsif last >= '1' && last <= '9'
        check = last.to_i
      else
        return nil
      end
      base = []
      8.times do |i|
        c = cleaned[i]
        return nil if c < '0' || c > '9'

        base << c.to_i
      end
      [base, check]
    end

    # valid_for_uf? validates value as an RG for the given UF (SP/RJ only).
    def self.valid_for_uf?(value, uf)
      return false unless RG_UFS.include?(uf)

      p = rg_parse(value)
      return false if p.nil?

      Mod11.compute_digit(Mod11.weighted_sum(p[0], DV[:weights]), DV) == p[1]
    end

    # valid? validates value under any implemented UF (first match wins).
    def self.valid?(value)
      RG_UFS.any? { |uf| valid_for_uf?(value, uf) }
    end

    # format renders an RG as XX.XXX.XXX-C (check char normalized).
    def self.format(value)
      p = rg_parse(value)
      %s if p.nil?

      check_char = Mod11.encode_digit(p[1], DV)
      d = p[0].join
      "#{d[0, 2]}.#{d[2, 3]}.#{d[5, 3]}-#{check_char}"
    end

    # generate returns a valid SP-style RG in masked form (XX.XXX.XXX-C).
    def self.generate
      base = Array.new(8) { rand(10) }
      dv = Mod11.compute_digit(Mod11.weighted_sum(base, DV[:weights]), DV)
      check_char = Mod11.encode_digit(dv, DV)
      d = base.join
      "#{d[0, 2]}.#{d[2, 3]}.#{d[5, 3]}-#{check_char}"
    end
  end
end
`, dv, ufs, rubyRaise("ErrInvalidFormat"))

	return b.String()
}

// renderIE emits the UF-scoped IE module (SP only): two rightmost-digit DVs at
// non-adjacent positions 9 and 12.
func (e rubyEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	writeRubyHeader(&b, false)

	dv1 := rubyCheckDigitLiteral(plan.Checks[0])
	dv2 := rubyCheckDigitLiteral(plan.Checks[1])
	ufs := rubyStringArray([]string{"SP"})
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module IE
    DV1 = %s.freeze
    DV2 = %s.freeze

    # IE_UFS lists the implemented federative units (SP only).
    IE_UFS = %s.freeze

    # ie_sp_validate validates a 12-digit São Paulo IE.
    def self.ie_sp_validate(d)
      return false if d.length != 12

      digits = d.chars.map(&:to_i)
      if Mod11.compute_digit(Mod11.weighted_sum(digits[0, 8], DV1[:weights]), DV1) != digits[8]
        return false
      end

      Mod11.compute_digit(Mod11.weighted_sum(digits[0, 11], DV2[:weights]), DV2) == digits[11]
    end

    # valid_for_uf? validates value as an IE for the given UF (SP only).
    def self.valid_for_uf?(value, uf)
      return false if uf != 'SP'

      d = Mod11.only_digits(value)
      return false if d.length != 12

      ie_sp_validate(d)
    end

    # valid? validates value under any implemented UF (first match wins).
    def self.valid?(value)
      IE_UFS.any? { |uf| valid_for_uf?(value, uf) }
    end

    # format renders SP IE as AAA.AAA.AAA.AAA, or raises when invalid.
    def self.format(value)
      d = Mod11.only_digits(value)
      return "#{d[0, 3]}.#{d[3, 3]}.#{d[6, 3]}.#{d[9, 3]}" if d.length == 12 && ie_sp_validate(d)

      %s
    end

    # generate returns a valid SP IE in masked form (AAA.AAA.AAA.AAA).
    def self.generate
      d = Array.new(12, '0')
      8.times { |i| d[i] = rand(10).to_s }
      digits = d.map(&:to_i)
      d[8] = Mod11.compute_digit(Mod11.weighted_sum(digits[0, 8], DV1[:weights]), DV1).to_s
      d[9] = rand(10).to_s
      d[10] = rand(10).to_s
      digits = d.map(&:to_i)
      d[11] = Mod11.compute_digit(Mod11.weighted_sum(digits[0, 11], DV2[:weights]), DV2).to_s
      s = d.join
      "#{s[0, 3]}.#{s[3, 3]}.#{s[6, 3]}.#{s[9, 3]}"
    end
  end
end
`, dv1, dv2, ufs, rubyRaise("ErrInvalidFormat"))

	return b.String()
}

// renderPlate emits the regex-only plate module (national + Mercosul).
func (e rubyEmitter) renderPlate() string {
	var b strings.Builder
	b.WriteString(rubyHeaderComment())
	b.WriteString("\n")
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	b.WriteString(`module Selo
  module Plate
    NATIONAL = /\A[A-Z]{3}-?[0-9]{4}\z/
    MERCOSUL = /\A[A-Z]{3}[0-9][A-Z][0-9]{2}\z/

    # valid? reports whether value is a national or Mercosul plate.
    def self.valid?(value)
      v = value.strip.upcase
      NATIONAL.match?(v) || MERCOSUL.match?(v)
    end

    # format canonicalizes the plate (national gains a dash), or raises.
    def self.format(value)
      v = value.strip.upcase
      return v if MERCOSUL.match?(v)

      if NATIONAL.match?(v)
        s = v.delete('-')
        return "#{s[0, 3]}-#{s[3, 4]}"
      end

      ` + rubyRaise("ErrInvalidFormat") + `
    end

    # generate returns a random valid national-pattern plate (ABC-1234).
    def self.generate
      letters = ('A'..'Z').to_a
      digits_chars = ('0'..'9').to_a
      "#{Array.new(3) { letters.sample }.join}-#{Array.new(4) { digits_chars.sample }.join}"
    end
  end
end
`)

	return b.String()
}

// renderPIX emits the composite PIX module: dispatch EVP -> email -> phone ->
// CPF -> CNPJ, reusing the CPF/CNPJ validators.
func (e rubyEmitter) renderPIX() string {
	var b strings.Builder
	b.WriteString(rubyHeaderComment())
	b.WriteString("\n")
	b.WriteString("require_relative 'mod11'\n")
	b.WriteString("require_relative 'cpf'\n")
	b.WriteString("require_relative 'cnpj'\n\n")
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	b.WriteString(`module Selo
  module PIX
    EVP = /\A[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}\z/
    PHONE = /\A\+55\d{10,11}\z/
    EMAIL = /\A[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)+\z/

    # detect_kind reports the PIX key kind, or nil when value is not a key.
    def self.detect_kind(value)
      v = value.strip
      return 'evp' if EVP.match?(v)
      return EMAIL.match?(v) ? 'email' : nil if v.include?('@')
      return PHONE.match?(v) ? 'phone' : nil if v.start_with?('+')

      digits = Mod11.only_digits(v).length
      return 'cpf' if digits == 11 && Selo::CPF.valid?(v)
      return 'cnpj' if digits == 14 && Selo::CNPJ.valid?(v)

      nil
    end

    # valid? reports whether value is a well-formed PIX key of any kind.
    def self.valid?(value)
      !detect_kind(value).nil?
    end

    # format returns the trimmed key verbatim, or raises when invalid.
    def self.format(value)
      v = value.strip
      ` + rubyRaise("ErrInvalidLength") + ` if detect_kind(v).nil?

      v
    end

    # generate returns a random UUIDv4 EVP PIX key.
    def self.generate
      b = Array.new(16) { rand(256) }
      b[6] = (b[6] & 0x0f) | 0x40
      b[8] = (b[8] & 0x3f) | 0x80
      hex = b.map { |x| x.to_s(16).rjust(2, '0') }
      "#{hex[0..3].join}-#{hex[4..5].join}-#{hex[6..7].join}-#{hex[8..9].join}-#{hex[10..15].join}"
    end
  end
end
`)

	return b.String()
}

// renderCEP emits the table-lookup CEP module: prefix-range validation, mask
// format, and UF origin from the embedded CEP_RANGES table.
func (e rubyEmitter) renderCEP() string {
	var b strings.Builder
	writeRubyHeader(&b, true)
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	b.WriteString(`module Selo
  module CEP
    # cep_range_for returns the UF whose prefix range contains prefix, or nil.
    def self.cep_range_for(prefix)
      Data::CEP_RANGES.each do |r|
        return r[:uf] if prefix >= r[:from] && prefix <= r[:to]
      end
      nil
    end

    # valid? reports whether value is a CEP whose prefix maps to a UF.
    def self.valid?(value)
      d = Mod11.only_digits(value)
      return false if d.length != 8

      prefix = d[0, 3].to_i
      !cep_range_for(prefix).nil?
    end

    # format masks a CEP as #####-###, or raises on bad length.
    def self.format(value)
      d = Mod11.only_digits(value)
      ` + rubyRaise("ErrInvalidLength") + ` if d.length != 8

      "#{d[0, 5]}-#{d[5, 3]}"
    end

    # origin returns the UF whose prefix range contains value, or raises.
    def self.origin(value)
      d = Mod11.only_digits(value)
      ` + rubyRaise("ErrInvalidLength") + ` if d.length != 8

      uf = cep_range_for(d[0, 3].to_i)
      ` + rubyRaise("ErrInvalidFormat") + ` if uf.nil?

      uf
    end

    # generate returns a random, valid 8-digit CEP (unformatted).
    def self.generate
      r = Data::CEP_RANGES.sample
      prefix = r[:from] + rand(r[:to] - r[:from] + 1)
      suffix = rand(100_000)
      sprintf('%03d%05d', prefix, suffix)
    end
  end
end
`)

	return b.String()
}

// renderPhone emits the table-lookup phone module: optional +55/0055 prefix,
// DDD->UF validation, mobile/landline mask, and DDD origin.
func (e rubyEmitter) renderPhone() string {
	var b strings.Builder
	writeRubyHeader(&b, true)
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	b.WriteString(`module Selo
  module Phone
    # national_number strips a +55/0055 country prefix, returning the rest or nil.
    def self.national_number(d)
      if d.start_with?('0055')
        d = d[4..]
      elsif d.start_with?('55') && d.length > 11
        d = d[2..]
      end
      return nil if d == ''

      d
    end

    # valid? reports whether value is a valid phone whose DDD maps to a UF.
    def self.valid?(value)
      n = national_number(Mod11.only_digits(value))
      return false if n.nil?
      return false if n.length != 10 && n.length != 11

      ddd = n[0, 2]
      return false unless Data::DDD_TO_UF.key?(ddd)
      return false if n.length == 11 && n[2] != '9'

      true
    end

    # format masks as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or raises.
    def self.format(value)
      n = national_number(Mod11.only_digits(value))
      ` + rubyRaise("ErrInvalidLength") + ` if n.nil? || (n.length != 10 && n.length != 11)

      ddd = n[0, 2]
      ` + rubyRaise("ErrInvalidFormat") + ` unless Data::DDD_TO_UF.key?(ddd)

      sub = n[2..]
      return "(#{ddd}) #{sub[0, 5]}-#{sub[5, 4]}" if sub.length == 9

      "(#{ddd}) #{sub[0, 4]}-#{sub[4, 4]}"
    end

    # origin returns the UF for the phone's DDD, or raises.
    def self.origin(value)
      n = national_number(Mod11.only_digits(value))
      ` + rubyRaise("ErrInvalidLength") + ` if n.nil? || (n.length != 10 && n.length != 11)

      ddd = n[0, 2]
      uf = Data::DDD_TO_UF[ddd]
      ` + rubyRaise("ErrInvalidFormat") + ` if uf.nil?

      uf
    end

    # generate returns a random valid Brazilian phone (unformatted national digits).
    def self.generate
      ddds = Data::DDD_TO_UF.keys
      ddd = ddds.sample
      if rand(2) == 0
        ddd + '9' + Array.new(8) { rand(10).to_s }.join
      else
        ddd + (2 + rand(4)).to_s + Array.new(7) { rand(10).to_s }.join
      end
    end
  end
end
`)

	return b.String()
}

// renderVoterID emits the dual-DV voter module (bespoke per the spec Note): DV1
// over the 8 sequence digits; DV2 over [ufDigit0, ufDigit1, dv1]; UF code 01..28.
func (e rubyEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	writeRubyHeader(&b, true)

	dv1 := rubyCheckDigitLiteral(plan.Checks[0])
	dv2 := rubyCheckDigitLiteral(plan.Checks[1])
	//nolint:dupword // generated Ruby nests two consecutive `end` keywords; not prose
	fmt.Fprintf(&b, `module Selo
  module VoterId
    DV1 = %s.freeze
    DV2 = %s.freeze

    # voter_dv1 computes the first check digit over the 8 sequence digits.
    def self.voter_dv1(d)
      seq = d[0, 8].chars.map(&:to_i)
      Mod11.compute_digit(Mod11.weighted_sum(seq, DV1[:weights]), DV1)
    end

    # voter_dv2 computes the second check digit over [uf0, uf1, dv1].
    def self.voter_dv2(d, dv1)
      vals = [d[8].to_i, d[9].to_i, dv1]
      Mod11.compute_digit(Mod11.weighted_sum(vals, DV2[:weights]), DV2)
    end

    # valid? reports whether value is a well-formed Título Eleitoral.
    def self.valid?(value)
      d = Mod11.only_digits(value)
      return false if d.length != 12
      return false if Mod11.all_equal(d)

      uf_code = (d[8].to_i * 10) + d[9].to_i
      return false if uf_code < 1 || uf_code > 28

      dv1 = voter_dv1(d)
      dv2 = voter_dv2(d, dv1)
      dv1 == d[10].to_i && dv2 == d[11].to_i
    end

    # format groups the voter ID as "SSSS SSSS UUDD", or raises.
    def self.format(value)
      d = Mod11.only_digits(value)
      %s if d.length != 12

      "#{d[0, 4]} #{d[4, 4]} #{d[8, 4]}"
    end

    # origin returns the region encoded in the UF code, or raises.
    def self.origin(value)
      d = Mod11.only_digits(value)
      %s if d.length != 12

      uf_code = (d[8].to_i * 10) + d[9].to_i
      name = Data::VOTER_UF_NAMES[uf_code]
      %s if name.nil?

      name
    end

    # generate returns a random, valid Título Eleitoral (12 digits, unformatted).
    def self.generate
      loop do
        d = Array.new(12, '0')
        8.times { |i| d[i] = rand(10).to_s }
        uf = 1 + rand(28)
        d[8] = (uf / 10).to_s
        d[9] = (uf %% 10).to_s
        s = d.join
        dv1 = voter_dv1(s)
        d[10] = dv1.to_s
        d[11] = voter_dv2(s, dv1).to_s
        out = d.join
        next if Mod11.all_equal(out)
        return out
      end
    end
  end
end
`, dv1, dv2, rubyRaise("ErrInvalidLength"),
		rubyRaise("ErrInvalidLength"), rubyRaise("ErrInvalidFormat"))

	return b.String()
}

// rubyHasOrigin reports whether kind has an origin resolver in the generated
// Ruby module (mirrors originFnName in the TS test renderer).
func rubyHasOrigin(kind selo.Kind) bool {
	switch kind { //nolint:exhaustive // only origin-capable kinds return true; all others fall through
	case selo.KindCPF, selo.KindCEP, selo.KindPhone, selo.KindVoterID:
		return true
	default:
		return false
	}
}

// renderTest emits test/<kind>_test.rb driven by vectors/<kind>.json: it asserts
// validate (and format/origin) against the emitted module.
func (e rubyEmitter) renderTest(kind selo.Kind) string {
	name := rubyName(kind)
	className := strings.ToUpper(kind.String()[:1]) + kind.String()[1:]

	var b strings.Builder
	b.WriteString(rubyHeaderComment())
	b.WriteString("\n")
	b.WriteString("require 'minitest/autorun'\n")
	b.WriteString("require 'json'\n")
	fmt.Fprintf(&b, "require_relative '../lib/selo/%s'\n\n", kind.String())

	fmt.Fprintf(&b, "class %sTest < Minitest::Test\n", className)
	fmt.Fprintf(&b, "  VECTOR = JSON.parse(File.read(File.join(__dir__, '../vectors/%s.json'))).freeze\n\n", kind.String())

	// validate
	b.WriteString("  def test_validate\n")
	b.WriteString("    VECTOR['validate'].each do |c|\n")
	fmt.Fprintf(&b, "      assert_equal c['valid'], Selo::%s.valid?(c['input']), \"validate #{c['input'].inspect}\"\n", name)
	b.WriteString("    end\n")
	b.WriteString("  end\n\n")

	// format
	b.WriteString("  def test_format\n")
	b.WriteString("    VECTOR['format'].each do |c|\n")
	b.WriteString("      if c.key?('error')\n")
	fmt.Fprintf(&b, "        assert_raises(ArgumentError) { Selo::%s.format(c['input']) }\n", name)
	b.WriteString("      else\n")
	fmt.Fprintf(&b, "        assert_equal c['output'], Selo::%s.format(c['input']), \"format #{c['input'].inspect}\"\n", name)
	b.WriteString("      end\n")
	b.WriteString("    end\n")
	b.WriteString("  end\n")

	// origin
	if rubyHasOrigin(kind) {
		b.WriteString("\n  def test_origin\n")
		b.WriteString("    (VECTOR['origin'] || []).each do |c|\n")
		fmt.Fprintf(&b, "      assert_equal c['output'], Selo::%s.origin(c['input']), \"origin #{c['input'].inspect}\"\n", name)
		b.WriteString("    end\n")
		b.WriteString("  end\n")
	}

	// generate round-trip
	b.WriteString("\n  def test_generate\n")
	b.WriteString("    100.times do\n")
	fmt.Fprintf(&b, "      val = Selo::%s.generate\n", name)
	fmt.Fprintf(&b, "      assert Selo::%s.valid?(val), \"generate produced invalid: #{val.inspect}\"\n", name)
	b.WriteString("    end\n")
	b.WriteString("  end\n")

	b.WriteString("end\n")

	return b.String()
}
