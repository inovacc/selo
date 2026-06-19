package codegen

import (
	"fmt"
	"strings"
)

// emit_csharp_kinds2.go holds the remaining per-kind C# class renderers:
// RG/IE (UF-scoped), plate/pix (regex/composite), cep/phone (table lookup), and
// voter_id (dual-DV). Each mirrors its TypeScript counterpart in
// emit_ts_kinds2.go, translated faithfully to idiomatic C#.

// renderRG emits the UF-scoped RG class: 8 base digits + 1 check char
// (10->'X', 11->'0'); SP and RJ share the algorithm.
func (e csharpEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Rg", "RG validation, formatting, and UF scoping (SP/RJ).")

	dv := csCheckDigitLiteral(plan.Checks[0])
	ufs := csStringArray(plan, []string{"SP", "RJ"})
	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv = %s;

        /// <summary>RgUfs lists the implemented federative units (shared SP/RJ algorithm).</summary>
        public static readonly string[] RgUfs = { %s };

        /// <summary>Parse strips formatting and returns the 8 base digits + check value, or null.</summary>
        private static (int[] Base, int Check)? Parse(string value)
        {
            var sb = new System.Text.StringBuilder();
            foreach (var ch in value)
            {
                if ((ch >= '0' && ch <= '9') || ch == 'X' || ch == 'x')
                {
                    sb.Append(ch);
                }
            }

            var cleaned = sb.ToString();
            if (cleaned.Length != 9)
            {
                return null;
            }

            var last = cleaned[8];
            int check;
            if (last == 'X' || last == 'x')
            {
                check = 10;
            }
            else if (last == '0')
            {
                check = 11;
            }
            else if (last >= '1' && last <= '9')
            {
                check = last - '0';
            }
            else
            {
                return null;
            }

            var baseDigits = new int[8];
            for (var i = 0; i < 8; i++)
            {
                var c = cleaned[i];
                if (c < '0' || c > '9')
                {
                    return null;
                }

                baseDigits[i] = c - '0';
            }

            return (baseDigits, check);
        }

        /// <summary>ValidateForUf validates value as an RG for the given UF (SP/RJ only).</summary>
        public static bool ValidateForUf(string value, string uf)
        {
            if (Array.IndexOf(RgUfs, uf) < 0)
            {
                return false;
            }

            var p = Parse(value);
            if (p == null)
            {
                return false;
            }

            return Mod11.ComputeDigit(Mod11.WeightedSum(p.Value.Base, Dv.Weights), Dv) == p.Value.Check;
        }

        /// <summary>Validate validates value under any implemented UF (first match wins).</summary>
        public static bool Validate(string value)
        {
            foreach (var uf in RgUfs)
            {
                if (ValidateForUf(value, uf))
                {
                    return true;
                }
            }

            return false;
        }

        /// <summary>Format renders an RG as XX.XXX.XXX-C (check char normalized).</summary>
        public static string Format(string value)
        {
            var p = Parse(value);
            if (p == null)
            {
                %s
            }

            var checkChar = Mod11.EncodeDigit(p.Value.Check, Dv);
            var d = string.Concat(Array.ConvertAll(p.Value.Base, x => x.ToString(System.Globalization.CultureInfo.InvariantCulture)));
            return $"{d.Substring(0, 2)}.{d.Substring(2, 3)}.{d.Substring(5, 3)}-{checkChar}";
        }
`, dv, ufs, csFormatThrow("ErrInvalidFormat"))

	csClassClose(&b)

	return b.String()
}

// renderIE emits the UF-scoped IE class (SP only): two rightmost-digit DVs at
// non-adjacent positions 9 and 12.
func (e csharpEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Ie", "IE (Inscrição Estadual) validation and formatting (SP only).")

	dv1 := csCheckDigitLiteral(plan.Checks[0])
	dv2 := csCheckDigitLiteral(plan.Checks[1])
	ufs := csStringArray(plan, []string{"SP"})
	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv1 = %s;
        private static readonly CheckDigit Dv2 = %s;

        /// <summary>IeUfs lists the implemented federative units (SP only).</summary>
        public static readonly string[] IeUfs = { %s };

        /// <summary>SpValidate validates a 12-digit São Paulo IE.</summary>
        private static bool SpValidate(string d)
        {
            if (d.Length != 12)
            {
                return false;
            }

            var digits = Mod11.DigitsOf(d);
            if (Mod11.ComputeDigit(Mod11.WeightedSum(Slice(digits, 0, 8), Dv1.Weights), Dv1) != digits[8])
            {
                return false;
            }

            return Mod11.ComputeDigit(Mod11.WeightedSum(Slice(digits, 0, 11), Dv2.Weights), Dv2) == digits[11];
        }

        /// <summary>ValidateForUf validates value as an IE for the given UF (SP only).</summary>
        public static bool ValidateForUf(string value, string uf)
        {
            if (uf != "SP")
            {
                return false;
            }

            var d = Mod11.OnlyDigits(value);
            if (d.Length != 12)
            {
                return false;
            }

            return SpValidate(d);
        }

        /// <summary>Validate validates value under any implemented UF (first match wins).</summary>
        public static bool Validate(string value)
        {
            foreach (var uf in IeUfs)
            {
                if (ValidateForUf(value, uf))
                {
                    return true;
                }
            }

            return false;
        }

        /// <summary>Format renders SP IE as AAA.AAA.AAA.AAA, or throws when invalid.</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length == 12 && SpValidate(d))
            {
                return $"{d.Substring(0, 3)}.{d.Substring(3, 3)}.{d.Substring(6, 3)}.{d.Substring(9, 3)}";
            }

            %s
        }

        private static int[] Slice(int[] xs, int from, int to)
        {
            var n = to - from;
            var outv = new int[n];
            Array.Copy(xs, from, outv, 0, n);
            return outv;
        }
`, dv1, dv2, ufs, csFormatThrow("ErrInvalidFormat"))

	csClassClose(&b)

	return b.String()
}

// renderPlate emits the regex-only plate class (national + Mercosul).
func (e csharpEmitter) renderPlate(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(csHeaderComment())
	b.WriteString("\n")
	b.WriteString("using System;\n")
	b.WriteString("using System.Text.RegularExpressions;\n\n")
	b.WriteString("namespace Inovacc.Selo\n{\n")
	b.WriteString("    /// <summary>Plate validation and formatting (national + Mercosul).</summary>\n")
	b.WriteString("    public static class Plate\n    {\n")
	b.WriteString("        private static readonly Regex National = new Regex(\"^[A-Z]{3}-?[0-9]{4}$\", RegexOptions.Compiled);\n")
	b.WriteString("        private static readonly Regex Mercosul = new Regex(\"^[A-Z]{3}[0-9][A-Z][0-9]{2}$\", RegexOptions.Compiled);\n\n")
	b.WriteString(`        /// <summary>Validate reports whether value is a national or Mercosul plate.</summary>
        public static bool Validate(string value)
        {
            var v = value.Trim().ToUpperInvariant();
            return National.IsMatch(v) || Mercosul.IsMatch(v);
        }

        /// <summary>Format canonicalizes the plate (national gains a dash), or throws.</summary>
        public static string Format(string value)
        {
            var v = value.Trim().ToUpperInvariant();
            if (Mercosul.IsMatch(v))
            {
                return v;
            }

            if (National.IsMatch(v))
            {
                var s = v.Replace("-", string.Empty);
                return $"{s.Substring(0, 3)}-{s.Substring(3, 4)}";
            }

            ` + csFormatThrow("ErrInvalidFormat") + `
        }
`)

	b.WriteString("    }\n}\n")

	return b.String()
}

// renderPIX emits the composite PIX class: dispatch EVP -> email -> phone ->
// CPF -> CNPJ, reusing the Cpf/Cnpj validators.
func (e csharpEmitter) renderPIX(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(csHeaderComment())
	b.WriteString("\n")
	b.WriteString("using System;\n")
	b.WriteString("using System.Text.RegularExpressions;\n\n")
	b.WriteString("namespace Inovacc.Selo\n{\n")
	b.WriteString("    /// <summary>PIX composite key validation and formatting.</summary>\n")
	b.WriteString("    public static class Pix\n    {\n")
	b.WriteString("        private static readonly Regex Evp = new Regex(\"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$\", RegexOptions.Compiled);\n")
	b.WriteString("        private static readonly Regex PhoneRe = new Regex(@\"^\\+55\\d{10,11}$\", RegexOptions.Compiled);\n")
	b.WriteString("        private static readonly Regex Email = new Regex(@\"^[A-Za-z0-9._%+\\-]+@[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?)+$\", RegexOptions.Compiled);\n\n")
	b.WriteString(`        /// <summary>DetectKind reports the PIX key kind, or null when value is not a key.</summary>
        public static string? DetectKind(string value)
        {
            var v = value.Trim();
            if (Evp.IsMatch(v))
            {
                return "evp";
            }

            if (v.Contains('@'))
            {
                return Email.IsMatch(v) ? "email" : null;
            }

            if (v.StartsWith("+", StringComparison.Ordinal))
            {
                return PhoneRe.IsMatch(v) ? "phone" : null;
            }

            var digits = Mod11.OnlyDigits(v).Length;
            if (digits == 11 && Cpf.Validate(v))
            {
                return "cpf";
            }

            if (digits == 14 && Cnpj.Validate(v))
            {
                return "cnpj";
            }

            return null;
        }

        /// <summary>Validate reports whether value is a well-formed PIX key of any kind.</summary>
        public static bool Validate(string value)
        {
            return DetectKind(value) != null;
        }

        /// <summary>Format returns the trimmed key verbatim, or throws when invalid.</summary>
        public static string Format(string value)
        {
            var v = value.Trim();
            if (DetectKind(v) == null)
            {
                ` + csFormatThrow("ErrInvalidLength") + `
            }

            return v;
        }
`)

	b.WriteString("    }\n}\n")

	return b.String()
}

// renderCEP emits the table-lookup CEP class: prefix-range validation, mask
// format, and UF origin from the embedded Data.CepRanges table.
func (e csharpEmitter) renderCEP(_ KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Cep", "CEP validation, formatting, and UF origin.")

	b.WriteString(`        /// <summary>RangeFor returns the UF whose prefix range contains prefix, or null.</summary>
        private static string? RangeFor(int prefix)
        {
            foreach (var r in Data.CepRanges)
            {
                if (prefix >= r.From && prefix <= r.To)
                {
                    return r.Uf;
                }
            }

            return null;
        }

        /// <summary>Validate reports whether value is a CEP whose prefix maps to a UF.</summary>
        public static bool Validate(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 8)
            {
                return false;
            }

            var prefix = int.Parse(d.Substring(0, 3), System.Globalization.CultureInfo.InvariantCulture);
            return RangeFor(prefix) != null;
        }

        /// <summary>Format masks a CEP as #####-###, or throws on bad length.</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 8)
            {
                ` + csFormatThrow("ErrInvalidLength") + `
            }

            return $"{d.Substring(0, 5)}-{d.Substring(5, 3)}";
        }

        /// <summary>Origin returns the UF whose prefix range contains value, or throws.</summary>
        public static string Origin(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 8)
            {
                ` + csFormatThrow("ErrInvalidLength") + `
            }

            var uf = RangeFor(int.Parse(d.Substring(0, 3), System.Globalization.CultureInfo.InvariantCulture));
            if (uf == null)
            {
                ` + csFormatThrow("ErrInvalidFormat") + `
            }

            return uf;
        }
`)

	csClassClose(&b)

	return b.String()
}

// renderPhone emits the table-lookup phone class: optional +55/0055 prefix,
// DDD->UF validation, mobile/landline mask, and DDD origin.
func (e csharpEmitter) renderPhone(_ KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Phone", "Phone validation, formatting, and UF origin (DDD lookup).")

	b.WriteString(`        /// <summary>NationalNumber strips a +55/0055 country prefix, returning the rest, or null.</summary>
        private static string? NationalNumber(string d)
        {
            if (d.StartsWith("0055", StringComparison.Ordinal))
            {
                d = d.Substring(4);
            }
            else if (d.StartsWith("55", StringComparison.Ordinal) && d.Length > 11)
            {
                d = d.Substring(2);
            }

            if (d.Length == 0)
            {
                return null;
            }

            return d;
        }

        /// <summary>Validate reports whether value is a valid phone whose DDD maps to a UF.</summary>
        public static bool Validate(string value)
        {
            var n = NationalNumber(Mod11.OnlyDigits(value));
            if (n == null)
            {
                return false;
            }

            if (n.Length != 10 && n.Length != 11)
            {
                return false;
            }

            var ddd = n.Substring(0, 2);
            if (!Data.DddToUf.ContainsKey(ddd))
            {
                return false;
            }

            if (n.Length == 11 && n[2] != '9')
            {
                return false;
            }

            return true;
        }

        /// <summary>Format masks as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or throws.</summary>
        public static string Format(string value)
        {
            var n = NationalNumber(Mod11.OnlyDigits(value));
            if (n == null || (n.Length != 10 && n.Length != 11))
            {
                ` + csFormatThrow("ErrInvalidLength") + `
            }

            var ddd = n.Substring(0, 2);
            if (!Data.DddToUf.ContainsKey(ddd))
            {
                ` + csFormatThrow("ErrInvalidFormat") + `
            }

            var sub = n.Substring(2);
            if (sub.Length == 9)
            {
                return $"({ddd}) {sub.Substring(0, 5)}-{sub.Substring(5, 4)}";
            }

            return $"({ddd}) {sub.Substring(0, 4)}-{sub.Substring(4, 4)}";
        }

        /// <summary>Origin returns the UF for the phone's DDD, or throws.</summary>
        public static string Origin(string value)
        {
            var n = NationalNumber(Mod11.OnlyDigits(value));
            if (n == null || (n.Length != 10 && n.Length != 11))
            {
                ` + csFormatThrow("ErrInvalidLength") + `
            }

            var ddd = n.Substring(0, 2);
            if (!Data.DddToUf.TryGetValue(ddd, out var uf))
            {
                ` + csFormatThrow("ErrInvalidFormat") + `
            }

            return uf;
        }
`)

	csClassClose(&b)

	return b.String()
}

// renderVoterID emits the dual-DV voter class (bespoke per the Note): DV1 over
// the 8 sequence digits; DV2 over [ufDigit0, ufDigit1, dv1]; UF code 01..28.
func (e csharpEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "VoterId", "Voter ID (Título Eleitoral) validation, formatting, and origin.")

	dv1 := csCheckDigitLiteral(plan.Checks[0])
	dv2 := csCheckDigitLiteral(plan.Checks[1])
	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv1 = %s;
        private static readonly CheckDigit Dv2 = %s;

        /// <summary>ComputeDv1 computes the first check digit over the 8 sequence digits.</summary>
        private static int ComputeDv1(string d)
        {
            var seq = Mod11.DigitsOf(d.Substring(0, 8));
            return Mod11.ComputeDigit(Mod11.WeightedSum(seq, Dv1.Weights), Dv1);
        }

        /// <summary>ComputeDv2 computes the second check digit over [uf0, uf1, dv1].</summary>
        private static int ComputeDv2(string d, int dv1)
        {
            var vals = new[] { d[8] - '0', d[9] - '0', dv1 };
            return Mod11.ComputeDigit(Mod11.WeightedSum(vals, Dv2.Weights), Dv2);
        }

        /// <summary>Validate reports whether value is a well-formed Título Eleitoral.</summary>
        public static bool Validate(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 12)
            {
                return false;
            }

            if (Mod11.AllEqual(d))
            {
                return false;
            }

            var ufCode = ((d[8] - '0') * 10) + (d[9] - '0');
            if (ufCode < 1 || ufCode > 28)
            {
                return false;
            }

            var dv1 = ComputeDv1(d);
            var dv2 = ComputeDv2(d, dv1);
            return dv1 == d[10] - '0' && dv2 == d[11] - '0';
        }

        /// <summary>Format groups the voter ID as "SSSS SSSS UUDD", or throws.</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 12)
            {
                %s
            }

            return %s;
        }

        /// <summary>Origin returns the region encoded in the UF code, or throws.</summary>
        public static string Origin(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 12)
            {
                %s
            }

            var ufCode = ((d[8] - '0') * 10) + (d[9] - '0');
            if (!Data.VoterUfNames.TryGetValue(ufCode, out var name))
            {
                %s
            }

            return name;
        }
`, dv1, dv2,
		csFormatThrow("ErrInvalidLength"), csMaskExpr(plan.Mask, "d"),
		csFormatThrow("ErrInvalidLength"), csFormatThrow("ErrInvalidFormat"))

	csClassClose(&b)

	return b.String()
}
