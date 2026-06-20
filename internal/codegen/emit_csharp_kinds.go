package codegen

import (
	"fmt"
	"strings"
)

// emit_csharp_kinds.go holds the per-kind C# class renderers for groups A/B/C
// (CPF, PIS, RENAVAM, CNH, CNS, CNPJ). Each renders a deterministic class from
// the declarative KindPlan. The check-digit kinds reuse the shared Mod11.cs
// reducer; the irregular kinds (CNH coupled DVs, CNS sum-zero, CNPJ char-map)
// carry bespoke fragments, exactly as the spec Notes flag and the TS reference
// (emit_ts_kinds.go) does.
//
// Each class also carries a static Generate() that mirrors its TypeScript
// counterpart's generate<Kind>() (same output shape), using a shared
// System.Random source (the C# analogue of TS Math.random()).

// csRngField is the per-class random source used by Generate(), the C# analogue
// of the TS modules' Math.random(). Declared once per class that generates.
const csRngField = "        private static readonly Random Rng = new Random();\n\n"

// csMaskExpr converts a '#'/'X'-placeholder mask (e.g. "###.#####.##-#") into a
// C# interpolated-string expression slicing the cleaned digit variable v, e.g.
// $"{v.Substring(0, 3)}.{v.Substring(3, 5)}.{v.Substring(8, 2)}-{v.Substring(10, 1)}".
// Mirrors emit_ts_kinds2.go tsMaskExpr (slice end -> Substring length).
func csMaskExpr(mask, v string) string {
	var b strings.Builder
	b.WriteString("$\"")

	pos := 0

	i := 0
	for i < len(mask) {
		c := mask[i]
		if c == '#' || c == 'X' {
			start := pos

			for i < len(mask) && (mask[i] == '#' || mask[i] == 'X') {
				i++
				pos++
			}

			fmt.Fprintf(&b, "{%s.Substring(%d, %d)}", v, start, pos-start)

			continue
		}
		// literal separator
		b.WriteByte(c)

		i++
	}

	b.WriteString("\"")

	return b.String()
}

// csClassOpen writes the generated banner, usings, namespace, and class header.
func csClassOpen(b *strings.Builder, className, summary string) {
	b.WriteString(csHeaderComment())
	b.WriteString("\n")
	b.WriteString("using System;\n\n")
	b.WriteString("namespace Inovacc.Selo\n{\n")
	fmt.Fprintf(b, "    /// <summary>%s</summary>\n", summary)
	fmt.Fprintf(b, "    public static class %s\n    {\n", className)
}

// csClassClose closes the class and namespace braces.
func csClassClose(b *strings.Builder) {
	b.WriteString("    }\n}\n")
}

// renderCPF emits the CPF class: two coupled-by-input mod-11 DVs, all-equal
// rejection, mask format, and ninth-digit origin.
func (e csharpEmitter) renderCPF(plan KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Cpf", "CPF validation, formatting, and origin (issuing region).")

	dv1 := csCheckDigitLiteral(plan.Checks[0])
	dv2 := csCheckDigitLiteral(plan.Checks[1])

	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv1 = %s;
        private static readonly CheckDigit Dv2 = %s;

`+csRngField+`        /// <summary>Validate reports whether value is a valid CPF (formatted or not).</summary>
        public static bool Validate(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 11)
            {
                return false;
            }

            if (Mod11.AllEqual(d))
            {
                return false;
            }

            var digits = Mod11.DigitsOf(d);
            var dv1 = Mod11.ComputeDigit(Mod11.WeightedSum(Slice(digits, 0, 9), Dv1.Weights), Dv1);
            var dv2 = Mod11.ComputeDigit(Mod11.WeightedSum(Slice(digits, 0, 10), Dv2.Weights), Dv2);
            return dv1 == digits[9] && dv2 == digits[10];
        }

        /// <summary>Format renders value as XXX.XXX.XXX-XX, or throws on bad length.</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 11)
            {
                %s
            }

            return %s;
        }

        /// <summary>Origin returns the issuing region from the 9th digit, or throws.</summary>
        public static string Origin(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length < 9)
            {
                %s
            }

            if (!Data.CpfRegions.TryGetValue(d[8] - '0', out var region))
            {
                %s
            }

            return region;
        }

        /// <summary>Generate returns a random valid CPF in formatted form (XXX.XXX.XXX-XX).</summary>
        public static string Generate()
        {
            var d = new int[11];
            for (var i = 0; i < 9; i++)
            {
                d[i] = Rng.Next(10);
            }

            d[9] = Mod11.ComputeDigit(Mod11.WeightedSum(Slice(d, 0, 9), Dv1.Weights), Dv1);
            d[10] = Mod11.ComputeDigit(Mod11.WeightedSum(Slice(d, 0, 10), Dv2.Weights), Dv2);
            return Format(string.Concat(Array.ConvertAll(d, x => x.ToString(System.Globalization.CultureInfo.InvariantCulture))));
        }

        private static int[] Slice(int[] xs, int from, int to)
        {
            var n = to - from;
            var outv = new int[n];
            Array.Copy(xs, from, outv, 0, n);
            return outv;
        }
`, dv1, dv2,
		csFormatThrow("ErrInvalidLength"), csMaskExpr(plan.Mask, "d"),
		csFormatThrow("ErrInvalidLength"), csFormatThrow("ErrInvalidLength"))

	csClassClose(&b)

	return b.String()
}

// renderSimpleNumeric emits a single-DV numeric kind (PIS): mod-11 DV over the
// first length-1 digits, all-equal rejection, and a mask format.
func (e csharpEmitter) renderSimpleNumeric(plan KindPlan, kind interface{ String() string }, length int) string {
	className := csNameForString(kind.String())

	var b strings.Builder

	csClassOpen(&b, className, className+" validation and formatting.")

	dv := csCheckDigitLiteral(plan.Checks[0])
	base := length - 1
	mask := csMaskExpr(plan.Mask, "d")

	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv = %s;

`+csRngField+`        /// <summary>Validate reports whether value is a valid %[2]s.</summary>
        public static bool Validate(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != %[3]d)
            {
                return false;
            }

            if (Mod11.AllEqual(d))
            {
                return false;
            }

            var digits = Mod11.DigitsOf(d);
            var dv = Mod11.ComputeDigit(Mod11.WeightedSum(Slice(digits, 0, %[4]d), Dv.Weights), Dv);
            return dv == digits[%[4]d];
        }

        /// <summary>Format renders the canonical mask, or throws on bad length.</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != %[3]d)
            {
                %[5]s
            }

            return %[6]s;
        }

        /// <summary>Generate returns a random valid %[2]s in formatted form.</summary>
        public static string Generate()
        {
            string outv;
            do
            {
                var d = new int[%[4]d];
                for (var i = 0; i < %[4]d; i++)
                {
                    d[i] = Rng.Next(10);
                }

                var dv = Mod11.ComputeDigit(Mod11.WeightedSum(d, Dv.Weights), Dv);
                outv = string.Concat(Array.ConvertAll(d, x => x.ToString(System.Globalization.CultureInfo.InvariantCulture))) + dv.ToString(System.Globalization.CultureInfo.InvariantCulture);
            }
            while (Mod11.AllEqual(outv));

            return Format(outv);
        }

        private static int[] Slice(int[] xs, int from, int to)
        {
            var n = to - from;
            var outv = new int[n];
            Array.Copy(xs, from, outv, 0, n);
            return outv;
        }
`, dv, className, length, base, csFormatThrow("ErrInvalidLength"), mask)

	csClassClose(&b)

	return b.String()
}

// renderRenavam emits RENAVAM: single (sum*10)%11 DV, all-equal rejection, and a
// left-pad-to-11 format (no separator mask).
func (e csharpEmitter) renderRenavam(plan KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Renavam", "RENAVAM validation and formatting.")

	dv := csCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv = %s;

`+csRngField+`        /// <summary>Validate reports whether value is a valid 11-digit RENAVAM.</summary>
        public static bool Validate(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 11)
            {
                return false;
            }

            if (Mod11.AllEqual(d))
            {
                return false;
            }

            var digits = Mod11.DigitsOf(d);
            var dv = Mod11.ComputeDigit(Mod11.WeightedSum(Slice(digits, 0, 10), Dv.Weights), Dv);
            return dv == digits[10];
        }

        /// <summary>Format left-pads shorter inputs to 11 digits (no separator mask).</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length < 11)
            {
                d = new string('0', 11 - d.Length) + d;
            }

            return d;
        }

        /// <summary>Generate returns a random valid 11-digit RENAVAM.</summary>
        public static string Generate()
        {
            string outv;
            do
            {
                var d = new int[10];
                for (var i = 0; i < 10; i++)
                {
                    d[i] = Rng.Next(10);
                }

                var dv = Mod11.ComputeDigit(Mod11.WeightedSum(d, Dv.Weights), Dv);
                outv = string.Concat(Array.ConvertAll(d, x => x.ToString(System.Globalization.CultureInfo.InvariantCulture))) + dv.ToString(System.Globalization.CultureInfo.InvariantCulture);
            }
            while (Mod11.AllEqual(outv));

            return outv;
        }

        private static int[] Slice(int[] xs, int from, int to)
        {
            var n = to - from;
            var outv = new int[n];
            Array.Copy(xs, from, outv, 0, n);
            return outv;
        }
`, dv)

	csClassClose(&b)

	return b.String()
}

// renderCNH emits the coupled-DV CNH class (bespoke fragment per the Note):
// DV1 descending 9..1 (raw remainder >=10 -> DV1=0, carry offset 2); DV2
// ascending 1..9 with the offset subtracted before the mod-11 fold.
func (e csharpEmitter) renderCNH(_ KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Cnh", "CNH validation and formatting (coupled check digits).")

	b.WriteString(csRngField)
	b.WriteString(`        /// <summary>CheckDigits computes both coupled CNH check digits over the 9-digit base.</summary>
        private static (int Dv1, int Dv2) CheckDigits(string baseDigits)
        {
            var dsc = 0;
            var sum = 0;
            for (var i = 0; i < 9; i++)
            {
                sum += (baseDigits[i] - '0') * (9 - i);
            }

            var r = sum % 11;
            int dv1;
            if (r >= 10)
            {
                dv1 = 0;
                dsc = 2;
            }
            else
            {
                dv1 = r;
            }

            sum = 0;
            for (var i = 0; i < 9; i++)
            {
                sum += (baseDigits[i] - '0') * (1 + i);
            }

            r = (sum % 11) - dsc;
            if (r < 0)
            {
                r += 11;
            }

            var dv2 = r >= 10 ? 0 : r;
            return (dv1, dv2);
        }

        /// <summary>Validate reports whether value is a valid 11-digit CNH.</summary>
        public static bool Validate(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 11)
            {
                return false;
            }

            if (Mod11.AllEqual(d))
            {
                return false;
            }

            var (dv1, dv2) = CheckDigits(d.Substring(0, 9));
            return dv1 == d[9] - '0' && dv2 == d[10] - '0';
        }

        /// <summary>Format returns the cleaned 11-digit CNH (no separator mask).</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 11)
            {
                ` + csFormatThrow("ErrInvalidLength") + `
            }

            return d;
        }

        /// <summary>Generate returns a random valid 11-digit CNH.</summary>
        public static string Generate()
        {
            string outv;
            do
            {
                var baseDigits = new char[9];
                for (var i = 0; i < 9; i++)
                {
                    baseDigits[i] = (char)('0' + Rng.Next(10));
                }

                var baseStr = new string(baseDigits);
                var (dv1, dv2) = CheckDigits(baseStr);
                outv = baseStr + dv1.ToString(System.Globalization.CultureInfo.InvariantCulture) + dv2.ToString(System.Globalization.CultureInfo.InvariantCulture);
            }
            while (Mod11.AllEqual(outv));

            return outv;
        }
`)

	csClassClose(&b)

	return b.String()
}

// renderCNS emits the verify-only sum-zero class with prefix constraint.
func (e csharpEmitter) renderCNS(plan KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Cns", "CNS validation and formatting (sum % 11 == 0).")

	dv := csCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv = %s;

`+csRngField+`        /// <summary>Validate reports whether value is a well-formed CNS (sum %% 11 == 0).</summary>
        public static bool Validate(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 15)
            {
                return false;
            }

            if (Mod11.AllEqual(d))
            {
                return false;
            }

            var lead = d[0];
            if (!(lead == '1' || lead == '2' || lead == '7' || lead == '8' || lead == '9'))
            {
                return false;
            }

            var digits = Mod11.DigitsOf(d);
            return Mod11.ComputeDigit(Mod11.WeightedSum(digits, Dv.Weights), Dv) == 0;
        }

        /// <summary>Format returns the cleaned 15-digit CNS (no separator mask).</summary>
        public static string Format(string value)
        {
            var d = Mod11.OnlyDigits(value);
            if (d.Length != 15)
            {
                %s
            }

            return d;
        }
`, dv, csFormatThrow("ErrInvalidLength"))

	b.WriteString(`        private static readonly string[] CnsPrefixes = { "1", "2", "7", "8", "9" };

        /// <summary>Generate returns a random valid 15-digit CNS.</summary>
        public static string Generate()
        {
            while (true)
            {
                var d = new int[15];
                d[0] = CnsPrefixes[Rng.Next(CnsPrefixes.Length)][0] - '0';
                for (var i = 1; i < 14; i++)
                {
                    d[i] = Rng.Next(10);
                }

                var partial = 0;
                for (var i = 0; i < 14; i++)
                {
                    partial += d[i] * (15 - i);
                }

                var last = (11 - (partial % 11)) % 11;
                if (last == 10)
                {
                    continue;
                }

                d[14] = last;
                var outv = string.Concat(Array.ConvertAll(d, x => x.ToString(System.Globalization.CultureInfo.InvariantCulture)));
                if (!Mod11.AllEqual(outv))
                {
                    return outv;
                }
            }
        }
`)

	csClassClose(&b)

	return b.String()
}

// renderCNPJ emits the alphanumeric CNPJ class (bespoke char-map + RL-cycling
// weights per the Note): two DVs, last two chars numeric, all-equal rejection.
func (e csharpEmitter) renderCNPJ(plan KindPlan) string {
	var b strings.Builder
	csClassOpen(&b, "Cnpj", "CNPJ validation and formatting (alphanumeric, char-map).")

	dv := csCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `        private static readonly CheckDigit Dv = %s;

`+csRngField+`        /// <summary>Clean uppercases and keeps only [0-9A-Z], capped at 14 chars.</summary>
        private static string Clean(string value)
        {
            var sb = new System.Text.StringBuilder(14);
            foreach (var ch in value)
            {
                var up = char.ToUpperInvariant(ch);
                if ((up >= '0' && up <= '9') || (up >= 'A' && up <= 'Z'))
                {
                    sb.Append(up);
                    if (sb.Length == 14)
                    {
                        break;
                    }
                }
            }

            return sb.ToString();
        }

        /// <summary>ComputeDv computes one check digit over the base string (RL-cycling weights).</summary>
        private static int ComputeDv(string baseChars)
        {
            var vals = new int[baseChars.Length];
            for (var i = 0; i < baseChars.Length; i++)
            {
                vals[i] = Mod11.CharValue(baseChars[i]);
            }

            return Mod11.ComputeDigit(Mod11.WeightedSum(vals, Dv.Weights, true), Dv);
        }

        /// <summary>Validate reports whether value is a valid alphanumeric CNPJ.</summary>
        public static bool Validate(string value)
        {
            var c = Clean(value);
            if (c.Length != 14)
            {
                return false;
            }

            if (Mod11.AllEqual(c))
            {
                return false;
            }

            if (c[12] < '0' || c[12] > '9')
            {
                return false;
            }

            if (c[13] < '0' || c[13] > '9')
            {
                return false;
            }

            var baseChars = c.Substring(0, 12);
            var dv1 = ComputeDv(baseChars);
            var dv2 = ComputeDv(baseChars + dv1.ToString(System.Globalization.CultureInfo.InvariantCulture));
            return dv1 == c[12] - '0' && dv2 == c[13] - '0';
        }

        /// <summary>Format renders value as XX.XXX.XXX/XXXX-XX, or throws on bad length.</summary>
        public static string Format(string value)
        {
            var c = Clean(value);
            if (c.Length != 14)
            {
                %s
            }

            return %s;
        }
`, dv, csFormatThrow("ErrInvalidLength"), csMaskExpr(plan.Mask, "c"))

	b.WriteString(`        private const string Alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ";

        /// <summary>Generate returns a random valid alphanumeric CNPJ.</summary>
        public static string Generate()
        {
            var baseChars = new char[12];
            for (var i = 0; i < 12; i++)
            {
                baseChars[i] = Alphanum[Rng.Next(Alphanum.Length)];
            }

            var baseStr = new string(baseChars);
            var dv1 = ComputeDv(baseStr);
            var dv2 = ComputeDv(baseStr + dv1.ToString(System.Globalization.CultureInfo.InvariantCulture));
            return baseStr + dv1.ToString(System.Globalization.CultureInfo.InvariantCulture) + dv2.ToString(System.Globalization.CultureInfo.InvariantCulture);
        }
`)

	csClassClose(&b)

	return b.String()
}

// csNameForString maps a kind string id to a PascalCase class name without
// needing a selo.Kind (used by renderSimpleNumeric's generic kind param).
func csNameForString(id string) string {
	switch id {
	case "cpf":
		return "Cpf"
	case "cnpj":
		return "Cnpj"
	case "cnh":
		return "Cnh"
	case "pis":
		return "Pis"
	case "renavam":
		return "Renavam"
	case "voter_id":
		return "VoterId"
	case "cep":
		return "Cep"
	case "phone":
		return "Phone"
	case "plate":
		return "Plate"
	case "cns":
		return "Cns"
	case "rg":
		return "Rg"
	case "pix":
		return "Pix"
	case "ie":
		return "Ie"
	default:
		return strings.Title(id) //nolint:staticcheck // simple ASCII fallback
	}
}
