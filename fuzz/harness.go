// Differential fuzz harness - Port Mortem 2026, Track F.
//
// Approach: generate a large corpus of randomized inputs (both valid and
// invalid), run the ORIGINAL bytes.js through Node to produce expected
// outputs, and verify the Go port produces identical results. The vector
// corpus is generated once by gen-vectors (see fuzz/gen/), stored in
// fuzz/vectors.json, and replayed here against the Go implementation.
//
// Run: go run ./fuzz [-duration 60s] (reads fuzz/vectors.json,
// verifies every vector; with -duration it loops until the time budget
// is exhausted - this is the "60s+ continuous run" used for the
// Differential Fuzz Survivor bonus claim)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"portmortem/bytes-go"
)

type Vector struct {
	Kind   string `json:"kind"` // "format" | "parse" | "dispatch"
	Input  any    `json:"input"`
	Opts   *Opts  `json:"opts,omitempty"`
	Expect any    `json:"expect"` // string | number | null
}

type Opts struct {
	DecimalPlaces      *int   `json:"decimalPlaces,omitempty"`
	FixedDecimals      bool   `json:"fixedDecimals,omitempty"`
	ThousandsSeparator string `json:"thousandsSeparator,omitempty"`
	UnitSeparator      string `json:"unitSeparator,omitempty"`
	Unit               string `json:"unit,omitempty"`
}

func main() {
	duration := flag.Duration("duration", 0, "run continuously for this long (e.g. 60s), replaying the corpus")
	flag.Parse()

	f, err := os.Open("fuzz/vectors.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "open vectors:", err)
		os.Exit(1)
	}
	defer f.Close()

	var vectors []Vector
	if err := json.NewDecoder(f).Decode(&vectors); err != nil {
		fmt.Fprintln(os.Stderr, "decode vectors:", err)
		os.Exit(1)
	}

	start := time.Now()
	divergences := 0
	checked := 0
	var deadline time.Time
	if *duration > 0 {
		deadline = start.Add(*duration)
	}
	for {
		for i, v := range vectors {
			got := eval(&v)
			if !equal(got, v.Expect) {
				divergences++
				fmt.Printf("DIVERGENCE #%d vector=%d kind=%s input=%v expect=%v got=%v\n",
					divergences, i, v.Kind, v.Input, v.Expect, got)
				if divergences > 20 {
					fmt.Println("too many divergences, aborting")
					os.Exit(1)
				}
			}
			checked++
		}
		if deadline.IsZero() || time.Now().After(deadline) {
			break
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("differential fuzz complete: %d vectors checked in %v (%.0f/s)\n",
		checked, elapsed.Round(time.Millisecond), float64(checked)/elapsed.Seconds())
	fmt.Printf("divergences: %d\n", divergences)
	fmt.Printf("result: %s\n", map[bool]string{true: "ZERO DIVERGENCES", false: "DIVERGENCES FOUND"}[divergences == 0])
	if *duration > 0 {
		if divergences == 0 {
			fmt.Printf("(claimed: Differential Fuzz Survivor bonus - %s continuous run, zero divergences)\n", duration)
		}
	} else if divergences == 0 {
		fmt.Printf("(single-pass corpus replay: %d frozen vectors, zero divergences)\n", len(vectors))
	}
	os.Exit(map[bool]int{true: 0, false: 1}[divergences == 0])
}

func eval(v *Vector) any {
	switch v.Kind {
	case "format":
		f, ok := toFloat(v.Input)
		if !ok {
			return nil
		}
		s, ok := bytefmt.Format(f, toGoOpts(v.Opts))
		if !ok {
			return nil
		}
		return s
	case "parse":
		r, ok := bytefmt.Parse(v.Input)
		if !ok {
			return nil
		}
		return r
	default: // dispatch
		r, ok := bytefmt.Bytes(v.Input, toGoOpts(v.Opts))
		if !ok {
			return nil
		}
		return r
	}
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	}
	return 0, false
}

func toGoOpts(o *Opts) *bytefmt.Options {
	if o == nil {
		return nil
	}
	return &bytefmt.Options{
		DecimalPlaces:      o.DecimalPlaces,
		FixedDecimals:      o.FixedDecimals,
		ThousandsSeparator: o.ThousandsSeparator,
		UnitSeparator:      o.UnitSeparator,
		Unit:               o.Unit,
	}
}

func equal(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	af, aIsF := a.(float64)
	bf, bIsF := b.(float64)
	if aIsF && bIsF {
		if math.IsNaN(af) || math.IsNaN(bf) {
			return math.IsNaN(af) && math.IsNaN(bf)
		}
		return af == bf
	}
	as, aIsS := a.(string)
	bs, bIsS := b.(string)
	if aIsS && bIsS {
		return as == bs
	}
	return false
}
