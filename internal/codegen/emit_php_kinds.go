package codegen

import (
	"fmt"
	"strings"
)

// emit_php_kinds.go holds the per-kind PHP class renderers for the check-digit
// kinds (groups A, B, C). Each renders a deterministic class from the
// declarative KindPlan. The numeric kinds reuse the shared Mod11 reducer; the
// irregular kinds (CNH coupled DVs, CNS sum-zero, CNPJ char-map) carry bespoke
// fragments, exactly as the Python reference.

// renderCPF emits the CPF class: two input-coupled mod-11 DVs, all-equal
// rejection, mask format, and ninth-digit origin.
func (e phpEmitter) renderCPF(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Cpf\n{\n")

	dv1 := phpCheckDigitLiteral(plan.Checks[0])
	dv2 := phpCheckDigitLiteral(plan.Checks[1])

	fmt.Fprintf(&b, `    private const DV1 = %s;
    private const DV2 = %s;

    /** Report whether value is a valid CPF (formatted or not). */
    public static function validateCpf(string $value): bool
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 11) {
            return false;
        }
        if (Mod11::allEqual($d)) {
            return false;
        }
        $digits = self::toInts($d);
        $dv1 = Mod11::computeDigit(Mod11::weightedSum(array_slice($digits, 0, 9), self::DV1['weights']), self::DV1);
        $dv2 = Mod11::computeDigit(Mod11::weightedSum(array_slice($digits, 0, 10), self::DV2['weights']), self::DV2);
        return $dv1 === $digits[9] && $dv2 === $digits[10];
    }

    /** Render value as XXX.XXX.XXX-XX, or throw on bad length. */
    public static function formatCpf(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 11) {
            %s
        }
        return %s;
    }

    /** Return the issuing region from the 9th digit, or throw. */
    public static function originCpf(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) < 9) {
            %s
        }
        $region = Data::CPF_REGIONS[(int) $d[8]] ?? null;
        if ($region === null) {
            %s
        }
        return $region;
    }

    /** Return a random, valid CPF (unformatted, 11 digits). */
    public static function generateCpf(): string
    {
        while (true) {
            $number = [];
            for ($i = 0; $i < 9; $i++) {
                $number[] = random_int(0, 9);
            }
            $number[] = Mod11::computeDigit(Mod11::weightedSum($number, self::DV1['weights']), self::DV1);
            $number[] = Mod11::computeDigit(Mod11::weightedSum($number, self::DV2['weights']), self::DV2);
            $out = implode('', array_map('strval', $number));
            if (!Mod11::allEqual($out)) {
                return $out;
            }
        }
    }

    /**
     * @return array<int, int>
     */
    private static function toInts(string $d): array
    {
        $out = [];
        $len = strlen($d);
        for ($i = 0; $i < $len; $i++) {
            $out[] = (int) $d[$i];
        }
        return $out;
    }
}
`, dv1, dv2, phpThrow("ErrInvalidLength"), phpMaskExpr(plan.Mask, "$d"),
		phpThrow("ErrInvalidLength"), phpThrow("ErrInvalidLength"))

	return b.String()
}

// renderSimpleNumeric emits a single-DV numeric kind (PIS): mod-11 DV over the
// first length-1 digits, all-equal rejection, and a mask format.
func (e phpEmitter) renderSimpleNumeric(plan KindPlan, class string, length int) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	fmt.Fprintf(&b, "final class %s\n{\n", class)

	dv := phpCheckDigitLiteral(plan.Checks[0])
	base := length - 1
	mask := phpMaskExpr(plan.Mask, "$d")

	fmt.Fprintf(&b, `    private const DV = %[1]s;

    /** Report whether value is a valid %[2]s. */
    public static function validate%[2]s(string $value): bool
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== %[3]d) {
            return false;
        }
        if (Mod11::allEqual($d)) {
            return false;
        }
        $digits = [];
        for ($i = 0; $i < %[3]d; $i++) {
            $digits[] = (int) $d[$i];
        }
        $dv = Mod11::computeDigit(Mod11::weightedSum(array_slice($digits, 0, %[4]d), self::DV['weights']), self::DV);
        return $dv === $digits[%[4]d];
    }

    /** Render the canonical mask, or throw on bad length. */
    public static function format%[2]s(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== %[3]d) {
            %[5]s
        }
        return %[6]s;
    }

    /** Return a random, valid %[2]s (unformatted). */
    public static function generate%[2]s(): string
    {
        while (true) {
            $b = [];
            for ($i = 0; $i < %[4]d; $i++) {
                $b[] = random_int(0, 9);
            }
            $b[] = Mod11::computeDigit(Mod11::weightedSum($b, self::DV['weights']), self::DV);
            $out = implode('', array_map('strval', $b));
            if (!Mod11::allEqual($out)) {
                return $out;
            }
        }
    }
}
`, dv, class, length, base, phpThrow("ErrInvalidLength"), mask)

	return b.String()
}

// renderRenavam emits RENAVAM: single (sum*10)%11 DV, all-equal rejection, and a
// left-pad-to-11 format (no separator mask).
func (e phpEmitter) renderRenavam(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Renavam\n{\n")

	dv := phpCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `    private const DV = %s;

    /** Report whether value is a valid 11-digit RENAVAM. */
    public static function validateRenavam(string $value): bool
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 11) {
            return false;
        }
        if (Mod11::allEqual($d)) {
            return false;
        }
        $digits = [];
        for ($i = 0; $i < 11; $i++) {
            $digits[] = (int) $d[$i];
        }
        $dv = Mod11::computeDigit(Mod11::weightedSum(array_slice($digits, 0, 10), self::DV['weights']), self::DV);
        return $dv === $digits[10];
    }

    /** Left-pad shorter inputs to 11 digits (no separator mask). */
    public static function formatRenavam(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) < 11) {
            $d = str_repeat('0', 11 - strlen($d)) . $d;
        }
        return $d;
    }

    /** Return a random, valid RENAVAM (unformatted, 11 digits). */
    public static function generateRenavam(): string
    {
        while (true) {
            $b = [];
            for ($i = 0; $i < 10; $i++) {
                $b[] = random_int(0, 9);
            }
            $b[] = Mod11::computeDigit(Mod11::weightedSum($b, self::DV['weights']), self::DV);
            $out = implode('', array_map('strval', $b));
            if (!Mod11::allEqual($out)) {
                return $out;
            }
        }
    }
}
`, dv)

	return b.String()
}

// renderCNH emits the coupled-DV CNH class (bespoke fragment per the spec Note):
// DV1 descending 9..1 (raw remainder >=10 -> DV1=0, carry offset 2); DV2
// ascending 1..9 with the offset subtracted before the mod-11 fold.
func (e phpEmitter) renderCNH() string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Cnh\n{\n")

	fmt.Fprintf(&b, `    /**
     * Compute both coupled CNH check digits over the 9-digit base.
     *
     * @return array{0: int, 1: int}
     */
    private static function checkDigits(string $base): array
    {
        $dsc = 0;
        $total = 0;
        for ($i = 0; $i < 9; $i++) {
            $total += ((int) $base[$i]) * (9 - $i);
        }
        $r = $total %% 11;
        if ($r >= 10) {
            $dv1 = 0;
            $dsc = 2;
        } else {
            $dv1 = $r;
        }
        $total = 0;
        for ($i = 0; $i < 9; $i++) {
            $total += ((int) $base[$i]) * (1 + $i);
        }
        $r = ($total %% 11) - $dsc;
        if ($r < 0) {
            $r += 11;
        }
        $dv2 = $r >= 10 ? 0 : $r;
        return [$dv1, $dv2];
    }

    /** Report whether value is a valid 11-digit CNH. */
    public static function validateCnh(string $value): bool
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 11) {
            return false;
        }
        if (Mod11::allEqual($d)) {
            return false;
        }
        [$dv1, $dv2] = self::checkDigits(substr($d, 0, 9));
        return $dv1 === (int) $d[9] && $dv2 === (int) $d[10];
    }

    /** Return the cleaned 11-digit CNH (no separator mask). */
    public static function formatCnh(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 11) {
            %s
        }
        return $d;
    }

    /** Return a random, valid 11-digit CNH (unformatted). */
    public static function generateCnh(): string
    {
        while (true) {
            $base = '';
            for ($i = 0; $i < 9; $i++) {
                $base .= (string) random_int(0, 9);
            }
            [$dv1, $dv2] = self::checkDigits($base);
            $out = $base . (string) $dv1 . (string) $dv2;
            if (!Mod11::allEqual($out)) {
                return $out;
            }
        }
    }
}
`, phpThrow("ErrInvalidLength"))

	return b.String()
}

// renderCNS emits the verify-only sum-zero class with prefix constraint.
func (e phpEmitter) renderCNS(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Cns\n{\n")

	dv := phpCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `    private const DV = %s;

    /** @var array<int, string> */
    private const PREFIXES = ['1', '2', '7', '8', '9'];

    /** Report whether value is a well-formed CNS (sum %% 11 == 0). */
    public static function validateCns(string $value): bool
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 15) {
            return false;
        }
        if (Mod11::allEqual($d)) {
            return false;
        }
        $lead = $d[0];
        if (!in_array($lead, self::PREFIXES, true)) {
            return false;
        }
        $digits = [];
        for ($i = 0; $i < 15; $i++) {
            $digits[] = (int) $d[$i];
        }
        return Mod11::computeDigit(Mod11::weightedSum($digits, self::DV['weights']), self::DV) === 0;
    }

    /** Return the cleaned 15-digit CNS (no separator mask). */
    public static function formatCns(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 15) {
            %s
        }
        return $d;
    }

    /** Return a random, valid CNS (15 digits, sum %% 11 == 0). */
    public static function generateCns(): string
    {
        while (true) {
            $d = [self::PREFIXES[random_int(0, count(self::PREFIXES) - 1)]];
            for ($i = 1; $i < 14; $i++) {
                $d[] = (string) random_int(0, 9);
            }
            $partial = 0;
            for ($i = 0; $i < 14; $i++) {
                $partial += ((int) $d[$i]) * (15 - $i);
            }
            $last = (11 - ($partial %% 11)) %% 11;
            if ($last === 10) {
                continue;
            }
            $d[] = (string) $last;
            $out = implode('', $d);
            if (!Mod11::allEqual($out)) {
                return $out;
            }
        }
    }
}
`, dv, phpThrow("ErrInvalidLength"))

	return b.String()
}

// renderCNPJ emits the alphanumeric CNPJ class (bespoke char-map + RL-cycling
// weights per the spec Note): two DVs, last two chars numeric, all-equal reject.
func (e phpEmitter) renderCNPJ(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Cnpj\n{\n")

	dv := phpCheckDigitLiteral(plan.Checks[0])
	fmt.Fprintf(&b, `    private const DV = %s;

    private const ALPHANUM = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ';

    /** Uppercase and keep only [0-9A-Z], capped at 14 chars. */
    private static function clean(string $value): string
    {
        $out = '';
        $len = strlen($value);
        for ($i = 0; $i < $len; $i++) {
            $up = strtoupper($value[$i]);
            if (($up >= '0' && $up <= '9') || ($up >= 'A' && $up <= 'Z')) {
                $out .= $up;
                if (strlen($out) === 14) {
                    break;
                }
            }
        }
        return $out;
    }

    /** Compute one check digit over the base string (RL-cycling weights). */
    private static function dv(string $base): int
    {
        $vals = [];
        $len = strlen($base);
        for ($i = 0; $i < $len; $i++) {
            $vals[] = Mod11::charValue($base[$i]);
        }
        return Mod11::computeDigit(Mod11::weightedSum($vals, self::DV['weights'], true), self::DV);
    }

    /** Report whether value is a valid alphanumeric CNPJ. */
    public static function validateCnpj(string $value): bool
    {
        $c = self::clean($value);
        if (strlen($c) !== 14) {
            return false;
        }
        if (Mod11::allEqual($c)) {
            return false;
        }
        if ($c[12] < '0' || $c[12] > '9') {
            return false;
        }
        if ($c[13] < '0' || $c[13] > '9') {
            return false;
        }
        $base = substr($c, 0, 12);
        $dv1 = self::dv($base);
        $dv2 = self::dv($base . (string) $dv1);
        return $dv1 === (int) $c[12] && $dv2 === (int) $c[13];
    }

    /** Render value as XX.XXX.XXX/XXXX-XX, or throw on bad length. */
    public static function formatCnpj(string $value): string
    {
        $c = self::clean($value);
        if (strlen($c) !== 14) {
            %s
        }
        return substr($c, 0, 2) . '.' . substr($c, 2, 3) . '.' . substr($c, 5, 3) . '/' . substr($c, 8, 4) . '-' . substr($c, 12, 2);
    }

    /** Return a random, valid alphanumeric CNPJ (14 chars, unformatted). */
    public static function generateCnpj(): string
    {
        $base = '';
        for ($i = 0; $i < 12; $i++) {
            $base .= self::ALPHANUM[random_int(0, strlen(self::ALPHANUM) - 1)];
        }
        $dv1 = self::dv($base);
        $dv2 = self::dv($base . (string) $dv1);
        return $base . (string) $dv1 . (string) $dv2;
    }
}
`, dv, phpThrow("ErrInvalidLength"))

	return b.String()
}
