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
//	bytefmt --bridge              → JSON-lines RPC bridge (used by the
//	                                Node adapter to run the ORIGINAL mocha
//	                                suite against this binary)
package main

import (
	"bufio"
	"encoding/json"
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
	unit := flag.String("unit", "", "force unit (kb/mb/gb/tb/pb)")
	bench := flag.Int("bench", 0, "run benchmark over N random values (format+parse each)")
	bridge := flag.Bool("bridge", false, "JSON-lines RPC bridge for the Node test adapter")
	flag.Parse()

	if *bridge {
		runBridge()
		return
	}

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
	opts.Unit = *unit
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

// JSON-lines RPC bridge. One JSON object per line on stdin:
//
//	{"id":1,"op":"format","value":1024,"opts":{"decimalPlaces":2}}
//	{"id":2,"op":"parse","value":"1.5KB"}
//	{"id":3,"op":"bytes","value":1024}
//
// One JSON object per line on stdout:
//
//	{"id":1,"ok":true,"result":"1KB"}
//	{"id":2,"ok":true,"result":1536}
//	{"id":3,"ok":true,"result":"1KB"}
//	{"id":4,"ok":false,"result":null}
//
// Used by tests/port/adapter.js so the ORIGINAL mocha suite (which does
// require('..')) can be re-pointed at the Go binary via a thin adapter.
type bridgeRequest struct {
	ID    int            `json:"id"`
	Op    string         `json:"op"`
	Value any            `json:"value"`
	Opts  *bridgeOptions `json:"opts,omitempty"`
}

type bridgeOptions struct {
	DecimalPlaces      *int   `json:"decimalPlaces,omitempty"`
	FixedDecimals      bool   `json:"fixedDecimals,omitempty"`
	ThousandsSeparator string `json:"thousandsSeparator,omitempty"`
	UnitSeparator      string `json:"unitSeparator,omitempty"`
	Unit               string `json:"unit,omitempty"`
}

type bridgeResponse struct {
	ID     int  `json:"id"`
	OK     bool `json:"ok"`
	Result any  `json:"result"`
}

func runBridge() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	enc := json.NewEncoder(os.Stdout)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req bridgeRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = enc.Encode(bridgeResponse{ID: req.ID, OK: false, Result: nil})
			continue
		}
		var opts bytefmt.Options
		if req.Opts != nil {
			opts.DecimalPlaces = req.Opts.DecimalPlaces
			opts.FixedDecimals = req.Opts.FixedDecimals
			opts.ThousandsSeparator = req.Opts.ThousandsSeparator
			opts.UnitSeparator = req.Opts.UnitSeparator
			opts.Unit = req.Opts.Unit
		}
		res := bridgeResponse{ID: req.ID}
		switch req.Op {
		case "format":
			f, ok := asFloat(req.Value)
			if !ok {
				res.OK, res.Result = false, nil
				break
			}
			s, ok := bytefmt.Format(f, &opts)
			res.OK, res.Result = ok, s
		case "parse":
			// Mirror of the original: parse(number) returns the number itself.
			if f, ok := asFloat(req.Value); ok {
				res.OK, res.Result = true, f
				break
			}
			s, ok := req.Value.(string)
			if !ok {
				res.OK, res.Result = false, nil
				break
			}
			n, ok := bytefmt.Parse(s)
			res.OK, res.Result = ok, n
		case "bytes":
			v, ok := bytefmt.Bytes(req.Value, &opts)
			res.OK, res.Result = ok, v
		default:
			res.OK, res.Result = false, nil
		}
		_ = enc.Encode(res)
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	}
	return 0, false
}
