package codegen

import (
	"fmt"
	"strings"
)

// emit_php_kinds2.go holds the remaining per-kind PHP class renderers (RG/IE
// UF-scoped, plate/pix regex, cep/phone table lookup, voter dual-DV). Every
// algorithm is translated verbatim from the Python reference.

// renderRG emits the UF-scoped RG class: 8 base digits + 1 check char
// (10->'X', 11->'0'); SP and RJ share the algorithm.
func (e phpEmitter) renderRG(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Rg\n{\n")

	dv := phpCheckDigitLiteral(plan.Checks[0])
	ufs := phpStringList([]string{"SP", "RJ"})
	fmt.Fprintf(&b, `    private const DV = %s;

    /** @var array<int, string> */
    public const UFS = %s;

    /**
     * Strip formatting and return [baseDigits, check] or null.
     *
     * @return array{0: array<int, int>, 1: int}|null
     */
    private static function parse(string $value): ?array
    {
        $cleaned = '';
        $len = strlen($value);
        for ($i = 0; $i < $len; $i++) {
            $ch = $value[$i];
            if (($ch >= '0' && $ch <= '9') || $ch === 'X' || $ch === 'x') {
                $cleaned .= $ch;
            }
        }
        if (strlen($cleaned) !== 9) {
            return null;
        }
        $last = $cleaned[8];
        if ($last === 'X' || $last === 'x') {
            $check = 10;
        } elseif ($last === '0') {
            $check = 11;
        } elseif ($last >= '1' && $last <= '9') {
            $check = (int) $last;
        } else {
            return null;
        }
        $base = [];
        for ($i = 0; $i < 8; $i++) {
            $c = $cleaned[$i];
            if ($c < '0' || $c > '9') {
                return null;
            }
            $base[] = (int) $c;
        }
        return [$base, $check];
    }

    /** Validate value as an RG for the given UF (SP/RJ only). */
    public static function validateRgForUf(string $value, string $uf): bool
    {
        if (!in_array($uf, self::UFS, true)) {
            return false;
        }
        $p = self::parse($value);
        if ($p === null) {
            return false;
        }
        return Mod11::computeDigit(Mod11::weightedSum($p[0], self::DV['weights']), self::DV) === $p[1];
    }

    /** Validate value under any implemented UF (first match wins). */
    public static function validateRg(string $value): bool
    {
        foreach (self::UFS as $uf) {
            if (self::validateRgForUf($value, $uf)) {
                return true;
            }
        }
        return false;
    }

    /** Render an RG as XX.XXX.XXX-C (check char normalized). */
    public static function formatRg(string $value): string
    {
        $p = self::parse($value);
        if ($p === null) {
            %s
        }
        $checkChar = Mod11::encodeDigit($p[1], self::DV);
        $d = implode('', array_map('strval', $p[0]));
        return substr($d, 0, 2) . '.' . substr($d, 2, 3) . '.' . substr($d, 5, 3) . '-' . $checkChar;
    }

    /** Return a valid SP-style RG in masked form (XX.XXX.XXX-C). */
    public static function generateRg(): string
    {
        $base = [];
        for ($i = 0; $i < 8; $i++) {
            $base[] = random_int(0, 9);
        }
        $dv = Mod11::computeDigit(Mod11::weightedSum($base, self::DV['weights']), self::DV);
        $checkChar = Mod11::encodeDigit($dv, self::DV);
        $d = implode('', array_map('strval', $base));
        return substr($d, 0, 2) . '.' . substr($d, 2, 3) . '.' . substr($d, 5, 3) . '-' . $checkChar;
    }
}
`, dv, ufs, phpThrow("ErrInvalidFormat"))

	return b.String()
}

// renderIE emits the UF-scoped IE class (SP only): two rightmost-digit DVs at
// non-adjacent positions 9 and 12.
func (e phpEmitter) renderIE(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Ie\n{\n")

	dv1 := phpCheckDigitLiteral(plan.Checks[0])
	dv2 := phpCheckDigitLiteral(plan.Checks[1])
	ufs := phpStringList([]string{"SP"})
	fmt.Fprintf(&b, `    private const DV1 = %s;
    private const DV2 = %s;

    /** @var array<int, string> */
    public const UFS = %s;

    /** @var array<int, int> */
    private const W1 = [1, 3, 4, 5, 6, 7, 8, 10];

    /** @var array<int, int> */
    private const W2 = [3, 2, 10, 9, 8, 7, 6, 5, 4, 3, 2];

    /** Validate a 12-digit São Paulo IE. */
    private static function spValidate(string $d): bool
    {
        if (strlen($d) !== 12) {
            return false;
        }
        $digits = [];
        for ($i = 0; $i < 12; $i++) {
            $digits[] = (int) $d[$i];
        }
        if (Mod11::computeDigit(Mod11::weightedSum(array_slice($digits, 0, 8), self::DV1['weights']), self::DV1) !== $digits[8]) {
            return false;
        }
        return Mod11::computeDigit(Mod11::weightedSum(array_slice($digits, 0, 11), self::DV2['weights']), self::DV2) === $digits[11];
    }

    /** Validate value as an IE for the given UF (SP only). */
    public static function validateIeForUf(string $value, string $uf): bool
    {
        if ($uf !== 'SP') {
            return false;
        }
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 12) {
            return false;
        }
        return self::spValidate($d);
    }

    /** Validate value under any implemented UF (first match wins). */
    public static function validateIe(string $value): bool
    {
        foreach (self::UFS as $uf) {
            if (self::validateIeForUf($value, $uf)) {
                return true;
            }
        }
        return false;
    }

    /** Render SP IE as AAA.AAA.AAA.AAA, or throw when invalid. */
    public static function formatIe(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) === 12 && self::spValidate($d)) {
            return substr($d, 0, 3) . '.' . substr($d, 3, 3) . '.' . substr($d, 6, 3) . '.' . substr($d, 9, 3);
        }
        %s
    }

    /**
     * @param array<int, int> $digits
     * @param array<int, int> $weights
     */
    private static function rightmostDv(array $digits, array $weights): int
    {
        $total = 0;
        $n = count($weights);
        for ($i = 0; $i < $n; $i++) {
            $total += $digits[$i] * $weights[$i];
        }
        return ($total %% 11) %% 10;
    }

    /** Return a valid SP IE in masked form (AAA.AAA.AAA.AAA). */
    public static function generateIe(): string
    {
        $d = array_fill(0, 12, 0);
        for ($i = 0; $i < 8; $i++) {
            $d[$i] = random_int(0, 9);
        }
        $d[8] = self::rightmostDv(array_slice($d, 0, 8), self::W1);
        $d[9] = random_int(0, 9);
        $d[10] = random_int(0, 9);
        $d[11] = self::rightmostDv(array_slice($d, 0, 11), self::W2);
        $s = implode('', array_map('strval', $d));
        return substr($s, 0, 3) . '.' . substr($s, 3, 3) . '.' . substr($s, 6, 3) . '.' . substr($s, 9, 3);
    }
}
`, dv1, dv2, ufs, phpThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPlate emits the regex-only plate class (national + Mercosul).
func (e phpEmitter) renderPlate() string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Plate\n{\n")

	fmt.Fprintf(&b, `    private const NATIONAL = '/^[A-Z]{3}-?[0-9]{4}$/';
    private const MERCOSUL = '/^[A-Z]{3}[0-9][A-Z][0-9]{2}$/';

    private const LETTERS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';

    /** Report whether value is a national or Mercosul plate. */
    public static function validatePlate(string $value): bool
    {
        $v = strtoupper(trim($value));
        return preg_match(self::NATIONAL, $v) === 1 || preg_match(self::MERCOSUL, $v) === 1;
    }

    /** Canonicalize the plate (national gains a dash), or throw. */
    public static function formatPlate(string $value): string
    {
        $v = strtoupper(trim($value));
        if (preg_match(self::MERCOSUL, $v) === 1) {
            return $v;
        }
        if (preg_match(self::NATIONAL, $v) === 1) {
            $s = str_replace('-', '', $v);
            return substr($s, 0, 3) . '-' . substr($s, 3, 4);
        }
        %s
    }

    /** Return a random valid plate (national or Mercosul). */
    public static function generatePlate(): string
    {
        $rl = static fn (): string => self::LETTERS[random_int(0, 25)];
        $rd = static fn (): string => (string) random_int(0, 9);
        $letters = $rl() . $rl() . $rl();
        if (random_int(0, 1) === 0) {
            return $letters . '-' . $rd() . $rd() . $rd() . $rd();
        }
        return $letters . $rd() . $rl() . $rd() . $rd();
    }
}
`, phpThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPIX emits the composite PIX class: dispatch EVP -> email -> phone ->
// CPF -> CNPJ, reusing the Cpf/Cnpj validators.
func (e phpEmitter) renderPIX() string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Pix\n{\n")

	fmt.Fprintf(&b, `    private const EVP = '/^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$/';
    private const PHONE = '/^\+55\d{10,11}$/';
    private const EMAIL = '/^[A-Za-z0-9._%%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)+$/';

    /** Report the PIX key kind, or null when value is not a key. */
    public static function detectPixKind(string $value): ?string
    {
        $v = trim($value);
        if (preg_match(self::EVP, $v) === 1) {
            return 'evp';
        }
        if (strpos($v, '@') !== false) {
            return preg_match(self::EMAIL, $v) === 1 ? 'email' : null;
        }
        if (strlen($v) > 0 && $v[0] === '+') {
            return preg_match(self::PHONE, $v) === 1 ? 'phone' : null;
        }
        $digits = strlen(Mod11::onlyDigits($v));
        if ($digits === 11 && Cpf::validateCpf($v)) {
            return 'cpf';
        }
        if ($digits === 14 && Cnpj::validateCnpj($v)) {
            return 'cnpj';
        }
        return null;
    }

    /** Report whether value is a well-formed PIX key of any kind. */
    public static function validatePix(string $value): bool
    {
        return self::detectPixKind($value) !== null;
    }

    /** Return the trimmed key verbatim, or throw when invalid. */
    public static function formatPix(string $value): string
    {
        $v = trim($value);
        if (self::detectPixKind($v) === null) {
            %s
        }
        return $v;
    }

    /** Return a random, valid EVP (UUIDv4) PIX key. */
    public static function generatePix(): string
    {
        $bytes = random_bytes(16);
        $bytes[6] = chr((ord($bytes[6]) & 0x0f) | 0x40);
        $bytes[8] = chr((ord($bytes[8]) & 0x3f) | 0x80);
        $hex = bin2hex($bytes);
        return substr($hex, 0, 8) . '-' . substr($hex, 8, 4) . '-' . substr($hex, 12, 4) . '-' . substr($hex, 16, 4) . '-' . substr($hex, 20, 12);
    }
}
`, phpThrow("ErrInvalidLength"))

	return b.String()
}

// renderCEP emits the table-lookup CEP class: prefix-range validation, mask
// format, and UF origin from the embedded Data::CEP_RANGES table.
func (e phpEmitter) renderCEP() string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Cep\n{\n")

	fmt.Fprintf(&b, `    /** Return the UF whose prefix range contains prefix, or null. */
    private static function rangeFor(int $prefix): ?string
    {
        foreach (Data::CEP_RANGES as $r) {
            if ($r['from'] <= $prefix && $prefix <= $r['to']) {
                return $r['uf'];
            }
        }
        return null;
    }

    /** Report whether value is a CEP whose prefix maps to a UF. */
    public static function validateCep(string $value): bool
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 8) {
            return false;
        }
        $prefix = (int) substr($d, 0, 3);
        return self::rangeFor($prefix) !== null;
    }

    /** Mask a CEP as #####-###, or throw on bad length. */
    public static function formatCep(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 8) {
            %s
        }
        return substr($d, 0, 5) . '-' . substr($d, 5, 3);
    }

    /** Return the UF whose prefix range contains value, or throw. */
    public static function originCep(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 8) {
            %s
        }
        $uf = self::rangeFor((int) substr($d, 0, 3));
        if ($uf === null) {
            %s
        }
        return $uf;
    }

    /** Return a random, valid 8-digit CEP (unformatted). */
    public static function generateCep(): string
    {
        $r = Data::CEP_RANGES[random_int(0, count(Data::CEP_RANGES) - 1)];
        $prefix = $r['from'] + random_int(0, $r['to'] - $r['from']);
        $suffix = random_int(0, 99999);
        return sprintf('%%03d%%05d', $prefix, $suffix);
    }
}
`, phpThrow("ErrInvalidLength"), phpThrow("ErrInvalidLength"), phpThrow("ErrInvalidFormat"))

	return b.String()
}

// renderPhone emits the table-lookup phone class: optional +55/0055 prefix,
// DDD->UF validation, mobile/landline mask, and DDD origin.
func (e phpEmitter) renderPhone() string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class Phone\n{\n")

	fmt.Fprintf(&b, `    /** Strip a +55/0055 country prefix, returning the rest or null. */
    private static function nationalNumber(string $d): ?string
    {
        if (str_starts_with($d, '0055')) {
            $d = substr($d, 4);
        } elseif (str_starts_with($d, '55') && strlen($d) > 11) {
            $d = substr($d, 2);
        }
        if ($d === '') {
            return null;
        }
        return $d;
    }

    /** Report whether value is a valid phone whose DDD maps to a UF. */
    public static function validatePhone(string $value): bool
    {
        $n = self::nationalNumber(Mod11::onlyDigits($value));
        if ($n === null) {
            return false;
        }
        if (strlen($n) !== 10 && strlen($n) !== 11) {
            return false;
        }
        $ddd = substr($n, 0, 2);
        if (!isset(Data::DDD_TO_UF[$ddd])) {
            return false;
        }
        if (strlen($n) === 11 && $n[2] !== '9') {
            return false;
        }
        return true;
    }

    /** Mask as (DD) NNNNN-NNNN or (DD) NNNN-NNNN, or throw. */
    public static function formatPhone(string $value): string
    {
        $n = self::nationalNumber(Mod11::onlyDigits($value));
        if ($n === null || (strlen($n) !== 10 && strlen($n) !== 11)) {
            %s
        }
        $ddd = substr($n, 0, 2);
        if (!isset(Data::DDD_TO_UF[$ddd])) {
            %s
        }
        $sub = substr($n, 2);
        if (strlen($sub) === 9) {
            return '(' . $ddd . ') ' . substr($sub, 0, 5) . '-' . substr($sub, 5, 4);
        }
        return '(' . $ddd . ') ' . substr($sub, 0, 4) . '-' . substr($sub, 4, 4);
    }

    /** Return the UF for the phone's DDD, or throw. */
    public static function originPhone(string $value): string
    {
        $n = self::nationalNumber(Mod11::onlyDigits($value));
        if ($n === null || (strlen($n) !== 10 && strlen($n) !== 11)) {
            %s
        }
        $ddd = substr($n, 0, 2);
        $uf = Data::DDD_TO_UF[$ddd] ?? null;
        if ($uf === null) {
            %s
        }
        return $uf;
    }

    /** Return a random valid Brazilian phone (unformatted national digits). */
    public static function generatePhone(): string
    {
        $ddds = array_keys(Data::DDD_TO_UF);
        $ddd = $ddds[random_int(0, count($ddds) - 1)];
        if (random_int(0, 1) === 0) {
            $sub = '9';
            for ($i = 0; $i < 8; $i++) {
                $sub .= (string) random_int(0, 9);
            }
            return $ddd . $sub;
        }
        $sub = (string) (2 + random_int(0, 3));
        for ($i = 0; $i < 7; $i++) {
            $sub .= (string) random_int(0, 9);
        }
        return $ddd . $sub;
    }
}
`, phpThrow("ErrInvalidLength"), phpThrow("ErrInvalidFormat"),
		phpThrow("ErrInvalidLength"), phpThrow("ErrInvalidFormat"))

	return b.String()
}

// renderVoterID emits the dual-DV voter class (bespoke per the spec Note): DV1
// over the 8 sequence digits; DV2 over [ufDigit0, ufDigit1, dv1]; UF code 01..28.
func (e phpEmitter) renderVoterID(plan KindPlan) string {
	var b strings.Builder
	b.WriteString(phpFileHeader())
	b.WriteString("namespace Selo;\n\n")
	b.WriteString("final class VoterId\n{\n")

	dv1 := phpCheckDigitLiteral(plan.Checks[0])
	dv2 := phpCheckDigitLiteral(plan.Checks[1])
	fmt.Fprintf(&b, `    private const DV1 = %s;
    private const DV2 = %s;

    /** Compute the first check digit over the 8 sequence digits. */
    private static function dv1(string $d): int
    {
        $seq = [];
        for ($i = 0; $i < 8; $i++) {
            $seq[] = (int) $d[$i];
        }
        return Mod11::computeDigit(Mod11::weightedSum($seq, self::DV1['weights']), self::DV1);
    }

    /** Compute the second check digit over [uf0, uf1, dv1]. */
    private static function dv2(string $d, int $dv1): int
    {
        $vals = [(int) $d[8], (int) $d[9], $dv1];
        return Mod11::computeDigit(Mod11::weightedSum($vals, self::DV2['weights']), self::DV2);
    }

    /** Report whether value is a well-formed Título Eleitoral. */
    public static function validateVoterId(string $value): bool
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 12) {
            return false;
        }
        if (Mod11::allEqual($d)) {
            return false;
        }
        $ufCode = ((int) $d[8]) * 10 + ((int) $d[9]);
        if ($ufCode < 1 || $ufCode > 28) {
            return false;
        }
        $dv1 = self::dv1($d);
        $dv2 = self::dv2($d, $dv1);
        return $dv1 === (int) $d[10] && $dv2 === (int) $d[11];
    }

    /** Group the voter ID as "SSSS SSSS UUDD", or throw. */
    public static function formatVoterId(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 12) {
            %s
        }
        return substr($d, 0, 4) . ' ' . substr($d, 4, 4) . ' ' . substr($d, 8, 4);
    }

    /** Return the region encoded in the UF code, or throw. */
    public static function originVoterId(string $value): string
    {
        $d = Mod11::onlyDigits($value);
        if (strlen($d) !== 12) {
            %s
        }
        $ufCode = ((int) $d[8]) * 10 + ((int) $d[9]);
        $name = Data::VOTER_UF_NAMES[$ufCode] ?? null;
        if ($name === null) {
            %s
        }
        return $name;
    }

    /** Return a random, valid Título Eleitoral (12 digits, unformatted). */
    public static function generateVoterId(): string
    {
        while (true) {
            $d = array_fill(0, 12, 0);
            for ($i = 0; $i < 8; $i++) {
                $d[$i] = random_int(0, 9);
            }
            $uf = 1 + random_int(0, 27);
            $d[8] = intdiv($uf, 10);
            $d[9] = $uf %% 10;
            $s = implode('', array_map('strval', array_slice($d, 0, 10)));
            $dv1 = self::dv1($s);
            $d[10] = $dv1;
            $d[11] = self::dv2($s, $dv1);
            $out = implode('', array_map('strval', $d));
            if (!Mod11::allEqual($out)) {
                return $out;
            }
        }
    }
}
`, dv1, dv2, phpThrow("ErrInvalidLength"),
		phpThrow("ErrInvalidLength"), phpThrow("ErrInvalidFormat"))

	return b.String()
}
