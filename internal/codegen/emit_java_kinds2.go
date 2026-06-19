package codegen

import (
	"fmt"
	"strings"
)

// emit_java_kinds2.go holds the remaining per-kind Java class renderers: the
// UF-scoped RG/IE, the regex plate/pix, and the table-lookup cep/phone/voter
// kinds. Each faithfully translates the M2 TypeScript reference
// (emit_ts_kinds2.go) into Java.

// renderRG emits the UF-scoped RG class: 8 base digits + 1 check char
// (10->'X', 11->'0'); SP and RJ share the algorithm.
func (e javaEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "RG", "RG validates and formats a Brazilian RG (SP/RJ shared algorithm).")

	dv := checkDigitNew(plan.Checks[0])
	ufs := javaStringArray([]string{"SP", "RJ"})
	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV = %s;

    /** RG_UFS lists the implemented federative units (shared SP/RJ algorithm). */
    public static final String[] RG_UFS = %s;

    /** RGParsed holds the 8 base digits plus the parsed check value. */
    private record RGParsed(int[] base, int check) {
    }

    /** rgParse strips formatting and returns the 8 base digits + check value. */
    private static RGParsed rgParse(String value) {
        StringBuilder cleaned = new StringBuilder();
        for (int i = 0; i < value.length(); i++) {
            char ch = value.charAt(i);
            if ((ch >= '0' && ch <= '9') || ch == 'X' || ch == 'x') {
                cleaned.append(ch);
            }
        }
        if (cleaned.length() != 9) {
            return null;
        }
        char last = cleaned.charAt(8);
        int check;
        if (last == 'X' || last == 'x') {
            check = 10;
        } else if (last == '0') {
            check = 11;
        } else if (last >= '1' && last <= '9') {
            check = last - '0';
        } else {
            return null;
        }
        int[] base = new int[8];
        for (int i = 0; i < 8; i++) {
            char c = cleaned.charAt(i);
            if (c < '0' || c > '9') {
                return null;
            }
            base[i] = c - '0';
        }
        return new RGParsed(base, check);
    }

    /** validateForUF validates value as an RG for the given UF (SP/RJ only). */
    public static boolean validateForUF(String value, String uf) {
        boolean known = false;
        for (String u : RG_UFS) {
            if (u.equals(uf)) {
                known = true;
                break;
            }
        }
        if (!known) {
            return false;
        }
        RGParsed p = rgParse(value);
        if (p == null) {
            return false;
        }
        return Mod11.computeDigit(Mod11.weightedSum(p.base(), DV.weights), DV) == p.check();
    }

    /** validate validates value under any implemented UF (first match wins). */
    public static boolean validate(String value) {
        for (String uf : RG_UFS) {
            if (validateForUF(value, uf)) {
                return true;
            }
        }
        return false;
    }

    /** format renders an RG as XX.XXX.XXX-C (check char normalized). */
    public static String format(String value) {
        RGParsed p = rgParse(value);
        if (p == null) {
            %s
        }
        String checkChar = Mod11.encodeDigit(p.check(), DV);
        StringBuilder d = new StringBuilder();
        for (int v : p.base()) {
            d.append((char) ('0' + v));
        }
        String s = d.toString();
        return s.substring(0, 2) + "." + s.substring(2, 5) + "." + s.substring(5, 8) + "-" + checkChar;
    }
}
`, dv, ufs, javaThrow("ErrInvalidFormat"))

	return b.String()
}

// renderIE emits the UF-scoped IE class (SP only): two rightmost-digit DVs at
// non-adjacent positions 9 and 12.
func (e javaEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "IE", "IE validates and formats a Brazilian Inscricao Estadual (SP only).")

	dv1 := checkDigitNew(plan.Checks[0])
	dv2 := checkDigitNew(plan.Checks[1])
	ufs := javaStringArray([]string{"SP"})
	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV1 = %s;
    private static final Mod11.CheckDigit DV2 = %s;

    /** IE_UFS lists the implemented federative units (SP only). */
    public static final String[] IE_UFS = %s;

    /** ieSPValidate validates a 12-digit Sao Paulo IE. */
    private static boolean ieSPValidate(String d) {
        if (d.length() != 12) {
            return false;
        }
        int[] digits = Mod11.digits(d);
        if (Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(digits, 0, 8), DV1.weights), DV1) != digits[8]) {
            return false;
        }
        return Mod11.computeDigit(Mod11.weightedSum(Mod11.slice(digits, 0, 11), DV2.weights), DV2) == digits[11];
    }

    /** validateForUF validates value as an IE for the given UF (SP only). */
    public static boolean validateForUF(String value, String uf) {
        if (!uf.equals("SP")) {
            return false;
        }
        String d = Mod11.onlyDigits(value);
        if (d.length() != 12) {
            return false;
        }
        return ieSPValidate(d);
    }

    /** validate validates value under any implemented UF (first match wins). */
    public static boolean validate(String value) {
        for (String uf : IE_UFS) {
            if (validateForUF(value, uf)) {
                return true;
            }
        }
        return false;
    }

    /** format renders SP IE as AAA.AAA.AAA.AAA, or throws when invalid. */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() == 12 && ieSPValidate(d)) {
            return d.substring(0, 3) + "." + d.substring(3, 6) + "." + d.substring(6, 9) + "." + d.substring(9, 12);
        }
        %s
    }
}
`, dv1, dv2, ufs, javaThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPlate emits the regex-only plate class (national + Mercosul).
func (e javaEmitter) renderPlate(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(javaHeaderComment())
	b.WriteString("package com.inovacc.selo;\n\n")
	b.WriteString("import java.util.regex.Pattern;\n\n")
	b.WriteString("/** Plate validates and formats a Brazilian vehicle plate (national + Mercosul). */\n")
	b.WriteString("public final class Plate {\n")
	b.WriteString("    private Plate() {\n")
	b.WriteString("    }\n\n")
	fmt.Fprintf(&b, `    private static final Pattern NATIONAL = Pattern.compile("^[A-Z]{3}-?[0-9]{4}$");
    private static final Pattern MERCOSUL = Pattern.compile("^[A-Z]{3}[0-9][A-Z][0-9]{2}$");

    /** validate reports whether value is a national or Mercosul plate. */
    public static boolean validate(String value) {
        String v = value.trim().toUpperCase();
        return NATIONAL.matcher(v).matches() || MERCOSUL.matcher(v).matches();
    }

    /** format canonicalizes the plate (national gains a dash), or throws. */
    public static String format(String value) {
        String v = value.trim().toUpperCase();
        if (MERCOSUL.matcher(v).matches()) {
            return v;
        }
        if (NATIONAL.matcher(v).matches()) {
            String s = v.replace("-", "");
            return s.substring(0, 3) + "-" + s.substring(3, 7);
        }
        %s
    }
}
`, javaThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPIX emits the composite PIX class: dispatch EVP -> email -> phone ->
// CPF -> CNPJ, reusing the CPF/CNPJ validators.
func (e javaEmitter) renderPIX(_ KindPlan) string {
	var b strings.Builder
	b.WriteString(javaHeaderComment())
	b.WriteString("package com.inovacc.selo;\n\n")
	b.WriteString("import java.util.regex.Pattern;\n\n")
	b.WriteString("/** PIX validates and formats a Brazilian PIX key (composite key). */\n")
	b.WriteString("public final class PIX {\n")
	b.WriteString("    private PIX() {\n")
	b.WriteString("    }\n\n")
	fmt.Fprintf(&b, `    private static final Pattern EVP =
            Pattern.compile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$");
    private static final Pattern PHONE = Pattern.compile("^\\+55\\d{10,11}$");
    private static final Pattern EMAIL =
            Pattern.compile("^[A-Za-z0-9._%%+\\-]+@[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?(?:\\.[A-Za-z0-9](?:[A-Za-z0-9\\-]*[A-Za-z0-9])?)+$");

    /** detectKind reports the PIX key kind, or null when value is not a key. */
    public static String detectKind(String value) {
        String v = value.trim();
        if (EVP.matcher(v).matches()) {
            return "evp";
        }
        if (v.contains("@")) {
            return EMAIL.matcher(v).matches() ? "email" : null;
        }
        if (v.startsWith("+")) {
            return PHONE.matcher(v).matches() ? "phone" : null;
        }
        int digits = Mod11.onlyDigits(v).length();
        if (digits == 11 && CPF.validate(v)) {
            return "cpf";
        }
        if (digits == 14 && CNPJ.validate(v)) {
            return "cnpj";
        }
        return null;
    }

    /** validate reports whether value is a well-formed PIX key of any kind. */
    public static boolean validate(String value) {
        return detectKind(value) != null;
    }

    /** format returns the trimmed key verbatim, or throws when invalid. */
    public static String format(String value) {
        String v = value.trim();
        if (detectKind(v) == null) {
            %s
        }
        return v;
    }
}
`, javaThrow("ErrInvalidLength"))

	return b.String()
}

// renderCEP emits the table-lookup CEP class: prefix-range validation, mask
// format, and UF origin from the embedded CEP_RANGES table.
func (e javaEmitter) renderCEP(_ KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "CEP", "CEP validates, formats, and resolves the UF of a Brazilian CEP.")

	fmt.Fprintf(&b, `    /** cepRangeFor returns the UF whose prefix range contains prefix, or null. */
    private static String cepRangeFor(int prefix) {
        for (Data.UFRange r : Data.CEP_RANGES) {
            if (prefix >= r.from() && prefix <= r.to()) {
                return r.uf();
            }
        }
        return null;
    }

    /** validate reports whether value is a CEP whose prefix maps to a UF. */
    public static boolean validate(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 8) {
            return false;
        }
        int prefix = Integer.parseInt(d.substring(0, 3));
        return cepRangeFor(prefix) != null;
    }

    /** format masks a CEP as #####-###, or throws on bad length. */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 8) {
            %s
        }
        return d.substring(0, 5) + "-" + d.substring(5, 8);
    }

    /** origin returns the UF whose prefix range contains value, or throws. */
    public static String origin(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 8) {
            %s
        }
        String uf = cepRangeFor(Integer.parseInt(d.substring(0, 3)));
        if (uf == null) {
            %s
        }
        return uf;
    }
}
`, javaThrow("ErrInvalidLength"), javaThrow("ErrInvalidLength"), javaThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPhone emits the table-lookup phone class: optional +55/0055 prefix,
// DDD->UF validation, mobile/landline mask, and DDD origin.
func (e javaEmitter) renderPhone(_ KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "Phone", "Phone validates, formats, and resolves the UF of a Brazilian phone number.")

	fmt.Fprintf(&b, `    /** nationalNumber strips a +55/0055 country prefix, returning the rest. */
    private static String nationalNumber(String d) {
        if (d.startsWith("0055")) {
            d = d.substring(4);
        } else if (d.startsWith("55") && d.length() > 11) {
            d = d.substring(2);
        }
        if (d.isEmpty()) {
            return null;
        }
        return d;
    }

    /** validate reports whether value is a valid phone whose DDD maps to a UF. */
    public static boolean validate(String value) {
        String n = nationalNumber(Mod11.onlyDigits(value));
        if (n == null) {
            return false;
        }
        if (n.length() != 10 && n.length() != 11) {
            return false;
        }
        String ddd = n.substring(0, 2);
        if (!Data.DDD_TO_UF.containsKey(ddd)) {
            return false;
        }
        if (n.length() == 11 && n.charAt(2) != '9') {
            return false;
        }
        return true;
    }

    /** format masks as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or throws. */
    public static String format(String value) {
        String n = nationalNumber(Mod11.onlyDigits(value));
        if (n == null || (n.length() != 10 && n.length() != 11)) {
            %s
        }
        String ddd = n.substring(0, 2);
        if (!Data.DDD_TO_UF.containsKey(ddd)) {
            %s
        }
        String sub = n.substring(2);
        if (sub.length() == 9) {
            return "(" + ddd + ") " + sub.substring(0, 5) + "-" + sub.substring(5, 9);
        }
        return "(" + ddd + ") " + sub.substring(0, 4) + "-" + sub.substring(4, 8);
    }

    /** origin returns the UF for the phone's DDD, or throws. */
    public static String origin(String value) {
        String n = nationalNumber(Mod11.onlyDigits(value));
        if (n == null || (n.length() != 10 && n.length() != 11)) {
            %s
        }
        String ddd = n.substring(0, 2);
        String uf = Data.DDD_TO_UF.get(ddd);
        if (uf == null) {
            %s
        }
        return uf;
    }
}
`, javaThrow("ErrInvalidLength"), javaThrow("ErrInvalidFormat"),
		javaThrow("ErrInvalidLength"), javaThrow("ErrInvalidFormat"))

	return b.String()
}

// renderVoterID emits the dual-DV voter class (bespoke per the Note): DV1 over
// the 8 sequence digits; DV2 over [ufDigit0, ufDigit1, dv1]; UF code 01..28.
func (e javaEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	javaClassOpen(&b, "VoterId", "VoterId validates, formats, and resolves the region of a Titulo Eleitoral.")

	dv1 := checkDigitNew(plan.Checks[0])
	dv2 := checkDigitNew(plan.Checks[1])
	fmt.Fprintf(&b, `    private static final Mod11.CheckDigit DV1 = %s;
    private static final Mod11.CheckDigit DV2 = %s;

    /** voterDV1 computes the first check digit over the 8 sequence digits. */
    private static int voterDV1(String d) {
        int[] seq = Mod11.digits(d.substring(0, 8));
        return Mod11.computeDigit(Mod11.weightedSum(seq, DV1.weights), DV1);
    }

    /** voterDV2 computes the second check digit over [uf0, uf1, dv1]. */
    private static int voterDV2(String d, int dv1) {
        int[] vals = new int[]{d.charAt(8) - '0', d.charAt(9) - '0', dv1};
        return Mod11.computeDigit(Mod11.weightedSum(vals, DV2.weights), DV2);
    }

    /** validate reports whether value is a well-formed Titulo Eleitoral. */
    public static boolean validate(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 12) {
            return false;
        }
        if (Mod11.allEqual(d)) {
            return false;
        }
        int ufCode = (d.charAt(8) - '0') * 10 + (d.charAt(9) - '0');
        if (ufCode < 1 || ufCode > 28) {
            return false;
        }
        int dv1 = voterDV1(d);
        int dv2 = voterDV2(d, dv1);
        return dv1 == (d.charAt(10) - '0') && dv2 == (d.charAt(11) - '0');
    }

    /** format groups the voter ID as "SSSS SSSS UUDD", or throws. */
    public static String format(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 12) {
            %s
        }
        return d.substring(0, 4) + " " + d.substring(4, 8) + " " + d.substring(8, 12);
    }

    /** origin returns the region encoded in the UF code, or throws. */
    public static String origin(String value) {
        String d = Mod11.onlyDigits(value);
        if (d.length() != 12) {
            %s
        }
        int ufCode = (d.charAt(8) - '0') * 10 + (d.charAt(9) - '0');
        String name = Data.VOTER_UF_NAMES.get(ufCode);
        if (name == null) {
            %s
        }
        return name;
    }
}
`, dv1, dv2, javaThrow("ErrInvalidLength"),
		javaThrow("ErrInvalidLength"), javaThrow("ErrInvalidFormat"))

	return b.String()
}
