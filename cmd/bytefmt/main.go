// Command bytefmt is the CLI front-end for the Go port of bytes.js.
//
// Usage:
//
//	bytefmt 1024                  → 1KB        (format)
//	bytefmt -p "1.5 TB"           → 1649267441664 (parse)
//	bytefmt -p -1                 → -1
//	bytefmt --unit-separator ' ' 1024   → 1 KB
//	bytefmt --decimal-places 0 1536      → 2KB
//	bytefmt --thousands-separator ',' 1000000 → 1,000,000B
//	bytefmt --bench N             → format+parse N values, timing report
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"portmortem/bytes-go"
)

func main() {
	parseMode := flag.Bool("p", false, "parse mode: interpret the argument as a byte string")
	unitSeparator := flag.String("unit-separator", "", "separator between number and unit")
	thousandsSeparator := flag.String("thousands-separator", "", "thousands separator")
	decimalPlaces := flag.Int("decimal-places", -1, "decimal places (-1 = default 2)")
	fixedDecimals := flag.Bool("fixed", false, "keep trailing zeros")
	bench := flag.Int("bench", 0, "run benchmark over N random values (format+parse each)")
	flag.Parse()

	if *bench > 0 {
		runBench(*bench)
		return
	}

	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: bytefmt [-p] [--unit-separator S] [--thousands-separator S] [--decimal-places N] [--fixed] VALUE")
		os.Exit(2)
	}
	arg := args[0]

	if *parseMode {
		v, ok := bytefmt.Parse(arg)
		if !ok {
			fmt.Println("null")
			os.Exit(1)
		}
		fmt.Println(strconv.FormatFloat(v, 'f', -1, 64))
		return
	}

	v, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		fmt.Println("null")
		os.Exit(1)
	}
	var opts bytefmt.Options
	opts.UnitSeparator = *unitSeparator
	opts.ThousandsSeparator = *thousandsSeparator
	opts.FixedDecimals = *fixedDecimals
	if *decimalPlaces >= 0 {
		dp := *decimalPlaces
		opts.DecimalPlaces = &dp
	}
	out, ok := bytefmt.Format(v, &opts)
	if !ok {
		fmt.Println("null")
		os.Exit(1)
	}
	fmt.Println(out)
}

func runBench(n int) {
	rng := rand.New(rand.NewSource(42))
	inputs := make([]float64, n)
	for i := range inputs {
		inputs[i] = rng.Float64() * 1e15
	}

	start := time.Now()
	for _, v := range inputs {
		bytefmt.Format(v, nil)
	}
	formatElapsed := time.Since(start)

	start = time.Now()
	for i := 0; i < n; i++ {
		s := fmt.Sprintf("%.3f", math.Mod(inputs[i], 1e6))
		bytefmt.Parse(s)
	}
	parseElapsed := time.Since(start)

	fmt.Printf("bench: %d values\n", n)
	fmt.Printf("format: %v (%.1f ns/op)\n", formatElapsed, float64(formatElapsed.Nanoseconds())/float64(n))
	fmt.Printf("parse:  %v (%.1f ns/op)\n", parseElapsed, float64(parseElapsed.Nanoseconds())/float64(n))
	fmt.Printf("total:  %v\n", formatElapsed+parseElapsed)
	fmt.Printf("version: go port of bytes.js v3.1.2 (track F)\n")
	fmt.Printf("unsafe blocks: 0\n")
	_ = strings.Builder{}
}
