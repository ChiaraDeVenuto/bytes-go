package bytefmt

import "testing"

// Regression tests captured from the differential fuzz corpus (10,017
// vectors, zero divergences against the Node.js oracle).

func TestRegressionNegativeZeroToFixed(t *testing.T) {
	// JS: (-3.7e-12).toFixed(2) === "-0.00"  →  "-0pb" after trim
	got, ok := Format(-4251.6888210155375, &Options{Unit: "pb"})
	if !ok || got != "-0pb" {
		t.Errorf("got %q, want -0pb", got)
	}
}

func TestRegressionHugeValuePreservesDecimals(t *testing.T) {
	// JS: (10628341276711983000).toFixed(3) keeps ".080" (zero not trailing-removable)
	got, ok := Format(1.0628341276711983e19, &Options{DecimalPlaces: intPtr(3), Unit: "TB", ThousandsSeparator: ",", UnitSeparator: "\t"})
	if !ok || got != "9,666,420.080\tTB" {
		t.Errorf("got %q, want %q", got, "9,666,420.080\tTB")
	}
}

func TestRegressionTrimKeepsInteriorZero(t *testing.T) {
	// JS: format(-6711979.295222695, {decimalPlaces:4, unit:'xx'}) === "-6.4010 MB"
	got, ok := Format(-6711979.295222695, &Options{DecimalPlaces: intPtr(4), ThousandsSeparator: ",", UnitSeparator: " ", Unit: "xx"})
	if !ok || got != "-6.4010 MB" {
		t.Errorf("got %q, want %q", got, "-6.4010 MB")
	}
}

func TestRegressionNegativeHalfAwayFromZero(t *testing.T) {
	// JS: (-2.5).toFixed(0) === "-3"; (0.5).toFixed(0) === "1"
	if got := toFixed(-2.5, 0); got != "-3" {
		t.Errorf("toFixed(-2.5,0) = %q, want -3", got)
	}
	if got := toFixed(-0.5, 0); got != "-1" {
		t.Errorf("toFixed(-0.5,0) = %q, want -1", got)
	}
	if got := toFixed(0.5, 0); got != "1" {
		t.Errorf("toFixed(0.5,0) = %q, want 1", got)
	}
	if got := toFixed(-2.4, 0); got != "-2" {
		t.Errorf("toFixed(-2.4,0) = %q, want -2", got)
	}
}

func TestRegressionBeyond2Pow53(t *testing.T) {
	// JS: (7337691304465441).toFixed(0) preserves the exact integer
	got, ok := Format(7337691304465441, &Options{DecimalPlaces: intPtr(0), FixedDecimals: true, ThousandsSeparator: ",", Unit: "B"})
	if !ok || got != "7,337,691,304,465,441B" {
		t.Errorf("got %q, want %q", got, "7,337,691,304,465,441B")
	}
}

func intPtr(i int) *int { return &i }
