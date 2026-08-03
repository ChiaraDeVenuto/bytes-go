package bytefmt

import (
	"math"
	"testing"
)

func dp(v int) *int { return &v }

// ---------------------------------------------------------------------------
// Test byte format function — translated 1:1 from test/byte-format.js
// ---------------------------------------------------------------------------

func TestFormatInvalidInputs(t *testing.T) {
	// JS: undefined, null, true, false, NaN, Infinity, '', 'string', fn, {} → null
	if _, ok := Format(math.NaN(), nil); ok {
		t.Error("NaN should be invalid")
	}
	if _, ok := Format(math.Inf(1), nil); ok {
		t.Error("Inf should be invalid")
	}
	if _, ok := Format(math.Inf(-1), nil); ok {
		t.Error("-Inf should be invalid")
	}
	// '' / 'string' / fn / {} reach format() only via dispatcher (typeof checks);
	// Format is typed float64, so NaN/Inf cover the non-finite path.
}

func TestFormatBelow1024(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{0, "0b"}, {100, "100b"}, {-100, "-100b"},
	} {
		got, ok := Format(tc.in, nil)
		if !ok {
			t.Fatalf("Format(%v) returned invalid", tc.in)
		}
		if toLower(got) != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatKB(t *testing.T) {
	kb := float64(1 << 10)
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{kb, "1kb"}, {-kb, "-1kb"}, {2 * kb, "2kb"},
	} {
		got, _ := Format(tc.in, nil)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatMB(t *testing.T) {
	mb := float64(1 << 20)
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{mb, "1mb"}, {-mb, "-1mb"}, {2 * mb, "2mb"},
	} {
		got, _ := Format(tc.in, nil)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatGB(t *testing.T) {
	gb := float64(1 << 30)
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{gb, "1gb"}, {-gb, "-1gb"}, {2 * gb, "2gb"},
	} {
		got, _ := Format(tc.in, nil)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatTB(t *testing.T) {
	tb := float64(1<<30) * 1024
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{tb, "1tb"}, {-tb, "-1tb"}, {2 * tb, "2tb"},
	} {
		got, _ := Format(tc.in, nil)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatPB(t *testing.T) {
	pb := math.Pow(1024, 5)
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{pb, "1pb"}, {-pb, "-1pb"}, {2 * pb, "2pb"},
	} {
		got, _ := Format(tc.in, nil)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatStandardCase(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{10, "10B"},
		{1 << 10, "1KB"},
		{1 << 20, "1MB"},
		{1 << 30, "1GB"},
		{float64(1<<30) * 1024, "1TB"},
		{math.Pow(1024, 5), "1PB"},
	} {
		got, _ := Format(tc.in, nil)
		if got != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatThousandsSeparator(t *testing.T) {
	kb := float64(1 << 10)
	cases := []struct {
		in   float64
		opts *Options
		want string
	}{
		{1000, nil, "1000b"},
		{1000, &Options{ThousandsSeparator: ""}, "1000b"},
		{1000, &Options{ThousandsSeparator: "."}, "1.000b"},
		{1000, &Options{ThousandsSeparator: ","}, "1,000b"},
		{1000, &Options{ThousandsSeparator: " "}, "1 000b"},
		{1005.1005 * kb, &Options{DecimalPlaces: dp(4), ThousandsSeparator: "_"}, "1_005.1005kb"},
	}
	for _, tc := range cases {
		got, _ := Format(tc.in, tc.opts)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v, %+v) = %q, want %q", tc.in, tc.opts, got, tc.want)
		}
	}
}

func TestFormatUnitSeparator(t *testing.T) {
	cases := []struct {
		opts *Options
		want string
	}{
		{nil, "1KB"},
		{&Options{UnitSeparator: ""}, "1KB"},
		{&Options{UnitSeparator: " "}, "1 KB"},
		{&Options{UnitSeparator: "\t"}, "1\tKB"},
	}
	for _, tc := range cases {
		got, _ := Format(1024, tc.opts)
		if got != tc.want {
			t.Errorf("Format(1024, %+v) = %q, want %q", tc.opts, got, tc.want)
		}
	}
}

func TestFormatDecimalPlaces(t *testing.T) {
	kb := float64(1 << 10)
	cases := []struct {
		in   float64
		opts *Options
		want string
	}{
		{kb - 1, &Options{DecimalPlaces: dp(0)}, "1023b"},
		{kb, &Options{DecimalPlaces: dp(0)}, "1kb"},
		{1.4 * kb, &Options{DecimalPlaces: dp(0)}, "1kb"},
		{1.5 * kb, &Options{DecimalPlaces: dp(0)}, "2kb"},
		{kb - 1, &Options{DecimalPlaces: dp(1)}, "1023b"},
		{kb, &Options{DecimalPlaces: dp(1)}, "1kb"},
		{1.04 * kb, &Options{DecimalPlaces: dp(1)}, "1kb"},
		{1.05 * kb, &Options{DecimalPlaces: dp(1)}, "1.1kb"},
		{1.1005 * kb, &Options{DecimalPlaces: dp(4)}, "1.1005kb"},
	}
	for _, tc := range cases {
		got, _ := Format(tc.in, tc.opts)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v, %+v) = %q, want %q", tc.in, tc.opts, got, tc.want)
		}
	}
}

func TestFormatFixedDecimals(t *testing.T) {
	kb := float64(1 << 10)
	got, _ := Format(kb, &Options{DecimalPlaces: dp(3), FixedDecimals: true})
	if toLower(got) != "1.000kb" {
		t.Errorf("fixed decimals = %q, want %q", got, "1.000kb")
	}
}

func TestFormatFloats(t *testing.T) {
	mb := float64(1 << 20)
	kb := float64(1 << 10)
	cases := []struct {
		in   float64
		want string
	}{
		{1.2 * mb, "1.2mb"},
		{-1.2 * mb, "-1.2mb"},
		{1.2 * kb, "1.2kb"},
	}
	for _, tc := range cases {
		got, _ := Format(tc.in, nil)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatCustomUnit(t *testing.T) {
	mb := float64(1 << 20)
	gb := float64(1 << 30)
	tb := float64(1<<30) * 1024
	cases := []struct {
		in   float64
		opts *Options
		want string
	}{
		{12 * mb, &Options{Unit: "b"}, "12582912b"},
		{12 * mb, &Options{Unit: "kb"}, "12288kb"},
		{12 * gb, &Options{Unit: "mb"}, "12288mb"},
		{12 * tb, &Options{Unit: "gb"}, "12288gb"},
		{12 * mb, &Options{Unit: ""}, "12mb"},
		{12 * mb, &Options{Unit: "bb"}, "12mb"},
	}
	for _, tc := range cases {
		got, _ := Format(tc.in, tc.opts)
		if toLower(got) != tc.want {
			t.Errorf("Format(%v, %+v) = %q, want %q", tc.in, tc.opts, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Test byte parse function — translated 1:1 from test/byte-parse.js
// ---------------------------------------------------------------------------

func TestParseInvalidInputs(t *testing.T) {
	// JS: undefined, null, true, false, NaN, fn, {}, 'foobar' → null
	bad := []any{true, false, math.NaN(), func() {}, struct{}{}, "foobar"}
	for _, in := range bad {
		if _, ok := Parse(in); ok {
			t.Errorf("Parse(%v) should be invalid", in)
		}
	}
}

func TestParseRawNumber(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want float64
	}{
		{0, 0}, {-1, -1}, {1, 1}, {10.5, 10.5},
	} {
		got, ok := Parse(tc.in)
		if !ok || got != tc.want {
			t.Errorf("Parse(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseKB(t *testing.T) {
	for _, s := range []string{"1kb", "1KB", "1Kb", "1kB"} {
		if got, _ := Parse(s); got != math.Pow(1024, 1) {
			t.Errorf("Parse(%q) = %v, want %v", s, got, math.Pow(1024, 1))
		}
	}
	for _, s := range []string{"0.5kb", "0.5KB", "0.5Kb", "0.5kB"} {
		if got, _ := Parse(s); got != 0.5*math.Pow(1024, 1) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
	for _, s := range []string{"1.5kb", "1.5KB", "1.5Kb", "1.5kB"} {
		if got, _ := Parse(s); got != 1.5*math.Pow(1024, 1) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
}

func TestParseMB(t *testing.T) {
	for _, s := range []string{"1mb", "1MB", "1Mb", "1mB"} {
		if got, _ := Parse(s); got != math.Pow(1024, 2) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
}

func TestParseGB(t *testing.T) {
	for _, s := range []string{"1gb", "1GB", "1Gb", "1gB"} {
		if got, _ := Parse(s); got != math.Pow(1024, 3) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
}

func TestParseTB(t *testing.T) {
	for _, s := range []string{"1tb", "1TB", "1Tb", "1tB"} {
		if got, _ := Parse(s); got != math.Pow(1024, 4) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
	for _, s := range []string{"0.5tb", "0.5TB", "0.5Tb", "0.5tB"} {
		if got, _ := Parse(s); got != 0.5*math.Pow(1024, 4) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
	for _, s := range []string{"1.5tb", "1.5TB", "1.5Tb", "1.5tB"} {
		if got, _ := Parse(s); got != 1.5*math.Pow(1024, 4) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
}

func TestParsePB(t *testing.T) {
	for _, s := range []string{"1pb", "1PB", "1Pb", "1pB"} {
		if got, _ := Parse(s); got != math.Pow(1024, 5) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
	for _, s := range []string{"0.5pb", "0.5PB", "0.5Pb", "0.5pB"} {
		if got, _ := Parse(s); got != 0.5*math.Pow(1024, 5) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
	for _, s := range []string{"1.5pb", "1.5PB", "1.5Pb", "1.5pB"} {
		if got, _ := Parse(s); got != 1.5*math.Pow(1024, 5) {
			t.Errorf("Parse(%q) = %v", s, got)
		}
	}
}

func TestParseAssumeBytesWhenNoUnits(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{"0", 0}, {"-1", -1}, {"1024", 1024}, {"0x11", 0},
	} {
		if got, _ := Parse(tc.in); got != tc.want {
			t.Errorf("Parse(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseNegativeValues(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{"-1", -1},
		{"-1024", -1024},
		{"-1.5TB", -1.5 * math.Pow(1024, 4)},
	} {
		if got, _ := Parse(tc.in); got != tc.want {
			t.Errorf("Parse(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseDropPartialBytes(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want float64
	}{
		{"1.1b", 1},
		{"1.0001kb", 1024},
	} {
		if got, _ := Parse(tc.in); got != tc.want {
			t.Errorf("Parse(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseAllowWhitespace(t *testing.T) {
	if got, _ := Parse("1 TB"); got != math.Pow(1024, 4) {
		t.Errorf("Parse('1 TB') = %v", got)
	}
}

// ---------------------------------------------------------------------------
// Test constructor/dispatcher — translated 1:1 from test/bytes.js
// ---------------------------------------------------------------------------

func TestBytesFunctionExists(t *testing.T) {
	// JS: typeof bytes === 'function' — dispatcher exists and is callable.
	if _, ok := Bytes("1KB", nil); !ok {
		t.Error("Bytes dispatcher should be callable")
	}
}

func TestBytesInvalidInputs(t *testing.T) {
	// JS: undefined, null, true, false, NaN, fn, {}, 'foobar' → null
	bad := []any{true, false, math.NaN(), func() {}, struct{}{}, "foobar"}
	for _, in := range bad {
		if _, ok := Bytes(in, nil); ok {
			t.Errorf("Bytes(%v) should be invalid", in)
		}
	}
}

func TestBytesParseString(t *testing.T) {
	got, ok := Bytes("1KB", nil)
	if !ok {
		t.Fatal("Bytes('1KB') should be valid")
	}
	if got.(float64) != 1024 {
		t.Errorf("Bytes('1KB') = %v, want 1024", got)
	}
}

func TestBytesFormatNumber(t *testing.T) {
	got, ok := Bytes(float64(1024), nil)
	if !ok {
		t.Fatal("Bytes(1024) should be valid")
	}
	if got.(string) != "1KB" {
		t.Errorf("Bytes(1024) = %v, want 1KB", got)
	}
}

func TestBytesFormatNumberWithOptions(t *testing.T) {
	got, ok := Bytes(float64(1000), &Options{ThousandsSeparator: " "})
	if !ok {
		t.Fatal("Bytes(1000, opts) should be valid")
	}
	if got.(string) != "1 000B" {
		t.Errorf("Bytes(1000, opts) = %v, want '1 000B'", got)
	}
}

// helper: lowercase (mocha tests call .toLowerCase() on formatted strings)
func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
