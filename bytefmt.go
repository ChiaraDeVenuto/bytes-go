// Package bytefmt is a Go port of visionmedia/bytes.js v3.1.2 (MIT).
//
// Port Mortem 2026 — Code Resurrection Hackathon, Track F (JavaScript → Go).
// Behavioral parity target: the original test suite, translated 1:1.
package bytefmt

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"math/big"
)

// Options mirrors the JavaScript options object of bytes.format().
//
//   - DecimalPlaces: nil means "not provided" (JS: options.decimalPlaces === undefined → default 2).
//   - FixedDecimals:  JS: Boolean(options.fixedDecimals).
//   - ThousandsSeparator / UnitSeparator: JS: options.x || ” (empty string for nil).
//   - Unit: JS: options.unit || ”.
type Options struct {
	DecimalPlaces      *int
	FixedDecimals      bool
	ThousandsSeparator string
	UnitSeparator      string
	Unit               string
}

var unitMap = map[string]float64{
	"b":  1,
	"kb": 1 << 10,
	"mb": 1 << 20,
	"gb": 1 << 30,
	"tb": math.Pow(1024, 4),
	"pb": math.Pow(1024, 5),
}

// parseRegExp mirrors the original:
//
//	/^((-|\+)?(\d+(?:\.\d+)?)) *(kb|mb|gb|tb|pb)$/i
//
// It is kept for reference and classification; parseString performs the
// same matching without regexp overhead (regexp matching is ~3.5x slower
// than the V8 regex engine, so the hot path avoids it — see DECISIONS.md).
var parseRegExp = regexp.MustCompile(`(?i)^((-|\+)?(\d+(?:\.\d+)?)) *(kb|mb|gb|tb|pb)$`)

// Format converts a byte count into a human-readable string.
// It returns (_, false) where the original would return null.
func Format(value float64, opts *Options) (string, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", false
	}

	var o Options
	if opts != nil {
		o = *opts
	}

	mag := math.Abs(value)
	decimalPlaces := 2
	if o.DecimalPlaces != nil {
		decimalPlaces = *o.DecimalPlaces
	}

	unit := o.Unit
	unitKey := strings.ToLower(unit)
	if unitKey == "" || unitMap[unitKey] == 0 {
		switch {
		case mag >= unitMap["pb"]:
			unit = "PB"
		case mag >= unitMap["tb"]:
			unit = "TB"
		case mag >= unitMap["gb"]:
			unit = "GB"
		case mag >= unitMap["mb"]:
			unit = "MB"
		case mag >= unitMap["kb"]:
			unit = "KB"
		default:
			unit = "B"
		}
		unitKey = strings.ToLower(unit)
	}

	val := value / unitMap[unitKey]
	str := toFixed(val, decimalPlaces)

	if !o.FixedDecimals {
		str = trimDecimals(str)
	}

	if o.ThousandsSeparator != "" {
		str = addThousands(str, o.ThousandsSeparator)
	}

	return str + o.UnitSeparator + unit, true
}

// Parse converts a string (or raw number) into a byte count.
// It returns (_, false) where the original would return null.
func Parse(input any) (float64, bool) {
	switch v := input.(type) {
	case float64:
		if !math.IsNaN(v) {
			return v, true
		}
		return 0, false
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		floatValue, unit, matched := parseString(v)
		if !matched {
			// JS: parseInt(val, 10) — leading integer part only, base 10.
			floatValue = jsParseInt(v)
			unit = "b"
		}
		if math.IsNaN(floatValue) {
			return 0, false
		}
		return math.Floor(unitMap[unit] * floatValue), true
	default:
		return 0, false
	}
}

// Bytes is the dispatcher equivalent of the exported bytes() function:
// strings are parsed, numbers are formatted, anything else is null.
func Bytes(input any, opts *Options) (any, bool) {
	switch v := input.(type) {
	case string:
		return Parse(v)
	case float64, int, int64, uint64:
		return Format(toFloat64(input), opts)
	default:
		return nil, false
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	}
	return math.NaN()
}

// toFixed replicates V8's Number.prototype.toFixed rounding:
// round-half-away-from-zero on the absolute value, then re-apply the
// sign of the input (e.g. (-2.5).toFixed(0) === "-3", (0.5).toFixed(0)
// === "1"). Note: JS toFixed preserves a negative sign even when the
// rounded result is zero (e.g. (-3.7e-12).toFixed(2) === "-0.00").
// Fast path (|scaled| < 2^52) uses an exact round-half-up via FMA
// (single-rounding, see roundHalfUp); larger values use exact rational
// arithmetic because float64 halfway values are not representable there.
func toFixed(val float64, places int) string {
	if places < 0 {
		places = 0
	}
	scale := math.Pow10(places)
	av := math.Abs(val)
	negative := math.Signbit(val) // JS preserves sign of the input

	if av*scale < 1<<52 {
		r := roundHalfUp(av, scale)
		n := int64(r)
		if places == 0 {
			if n == 0 && negative {
				return "-0"
			}
			if negative {
				return "-" + strconv.FormatInt(n, 10)
			}
			return strconv.FormatInt(n, 10)
		}

		digits := strconv.FormatInt(n, 10)
		for len(digits) <= places {
			digits = "0" + digits
		}
		intPart := digits[:len(digits)-places]
		fracPart := digits[len(digits)-places:]
		if negative {
			return "-" + intPart + "." + fracPart
		}
		return intPart + "." + fracPart
	}

	return toFixedBig(val, places)
}

// roundHalfUp computes floor(av*scale + 0.5) with a single rounding
// (fused multiply-add), which matches ECMAScript's "closest n, ties pick
// the larger" exactly for av*scale < 2^52. The result is an integer.
func roundHalfUp(av, scale float64) float64 {
	r := math.FMA(av, scale, 0.5)
	if r == math.Trunc(r) {
		// r is an integer; recover the exact residual: true = r + e.
		if e := math.FMA(av, scale, 0.5-r); e < 0 {
			return r - 1
		}
	}
	return math.Floor(r)
}

// toFixedBig rounds the exact rational value of the float64 according to
// the ECMAScript spec (n/10^f closest to x, ties toward +Infinity),
// then formats it with exactly `places` decimals. Only reached for
// |val*10^places| >= 2^52 where float64 halfway arithmetic is lossy.
func toFixedBig(val float64, places int) string {
	negative := math.Signbit(val)
	av := math.Abs(val)

	rat := new(big.Rat).SetFloat64(av)
	scaleRat := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil))
	rat.Mul(rat, scaleRat)
	// n = floor(rat + 1/2) — rat >= 0, so truncation is floor.
	rat.Add(rat, big.NewRat(1, 2))
	q := new(big.Int).Quo(rat.Num(), rat.Denom())

	qs := q.String()
	sign := ""
	if negative {
		sign = "-"
	}
	if places == 0 {
		return sign + qs
	}
	for len(qs) <= places {
		qs = "0" + qs
	}
	intPart := qs[:len(qs)-places]
	fracPart := qs[len(qs)-places:]
	return sign + intPart + "." + fracPart
}

// trimDecimals replicates the original formatDecimalsRegExp
// /(?:\.0*|(\.[^0]+)0+)$/ replacement:
//   - a decimal part made of zeros only (".00", ".000") vanishes entirely;
//   - trailing zeros are dropped only when the decimal part matches
//     [^0]+0*$ — a run of non-zero digits followed by zeros — so
//     "1.50" -> "1.5" but "6.4010" and "1.050" are left untouched.
func trimDecimals(s string) string {
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return s
	}
	frac := s[i+1:]
	// (?:\.0*)$ — decimals are all zeros (or empty)
	if strings.Trim(frac, "0") == "" {
		return s[:i]
	}
	// (\.[^0]+)0+$ — trailing zeros are removable only when no zero
	// appears between the point and the final run of zeros.
	j := len(s)
	for j > i+1 && s[j-1] == '0' {
		j--
	}
	for k := i + 1; k < j; k++ {
		if s[k] == '0' {
			return s
		}
	}
	return s[:j]
}

// addThousands replicates the original formatThousandsRegExp
// /\B(?=(\d{3})+(?!\d))/g applied to the integer part only.
func addThousands(s, sep string) string {
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]

	neg := false
	if strings.HasPrefix(intPart, "-") {
		neg = true
		intPart = intPart[1:]
	}

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(c)
	}
	if len(parts) == 2 {
		b.WriteByte('.')
		b.WriteString(parts[1])
	}
	return b.String()
}

// parseString manually replicates the original parse regex
// /^((-|\+)?(\d+(?:\.\d+)?)) *(kb|mb|gb|tb|pb)$/i without regexp overhead.
// Returns the numeric part, the unit (lowercased) and whether it matched.
func parseString(s string) (float64, string, bool) {
	i := 0
	n := len(s)

	// sign: (-|\+)?
	sign := 1.0
	if i < n && (s[i] == '-' || s[i] == '+') {
		if s[i] == '-' {
			sign = -1
		}
		i++
	}

	// \d+
	digitsStart := i
	for i < n && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == digitsStart {
		return 0, "", false
	}
	numberEnd := i

	// (?:\.\d+)?
	if i < n && s[i] == '.' {
		i++
		decStart := i
		for i < n && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == decStart {
			return 0, "", false
		}
		numberEnd = i
	}

	//  * (single space character, zero or more — matching the original regex)
	spaceEnd := i
	for i < n && s[i] == ' ' {
		i++
	}
	_ = spaceEnd

	// (kb|mb|gb|tb|pb) case-insensitive
	var unit string
	switch {
	case i+2 <= n && (s[i] == 'k' || s[i] == 'K') && (s[i+1] == 'b' || s[i+1] == 'B'):
		unit = "kb"
	case i+2 <= n && (s[i] == 'm' || s[i] == 'M') && (s[i+1] == 'b' || s[i+1] == 'B'):
		unit = "mb"
	case i+2 <= n && (s[i] == 'g' || s[i] == 'G') && (s[i+1] == 'b' || s[i+1] == 'B'):
		unit = "gb"
	case i+2 <= n && (s[i] == 't' || s[i] == 'T') && (s[i+1] == 'b' || s[i+1] == 'B'):
		unit = "tb"
	case i+2 <= n && (s[i] == 'p' || s[i] == 'P') && (s[i+1] == 'b' || s[i+1] == 'B'):
		unit = "pb"
	default:
		return 0, "", false
	}
	i += 2

	// $ anchor
	if i != n {
		return 0, "", false
	}

	numStr := s[digitsStart:numberEnd]
	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return math.NaN(), unit, true
	}
	return sign * f, unit, true
}

// jsParseInt replicates parseInt(val, 10): skip leading whitespace,
// optional sign, then consume the longest leading run of decimal digits.
// Returns NaN when nothing parseable is found.
func jsParseInt(s string) float64 {
	s = strings.TrimLeft(s, " \t\n\r\v\f")
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == start {
		return math.NaN()
	}
	n, err := strconv.ParseInt(s[:i], 10, 64)
	if err != nil {
		return math.NaN()
	}
	return float64(n)
}
