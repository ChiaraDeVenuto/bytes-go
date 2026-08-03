// Benchmark runner for the Go port - same deterministic inputs as bench-node.js.
// Usage: go run ./bench [-n N]
package main

import (
	"flag"
	"fmt"
	"time"

	"portmortem/bytes-go"
)

// deterministic LCG, same seed as bench-node.js (42)
var seed uint32 = 42

func rnd() float64 {
	seed = seed*1664525 + 1013904223
	return float64(seed) / 4294967296
}

func main() {
	n := flag.Int("n", 200000, "number of values per op")
	flag.Parse()

	// format
	seed = 42
	t0 := time.Now()
	for i := 0; i < *n; i++ {
		bytefmt.Format(rnd()*1e15, nil)
	}
	tf := time.Since(t0)

	// parse
	seed = 42
	t0 = time.Now()
	for i := 0; i < *n; i++ {
		bytefmt.Parse(fmt.Sprintf("%.2f%s", rnd()*1e6, []string{"B", "KB", "MB", "GB", "TB", "PB"}[int(rnd()*6)%6]))
	}
	tp := time.Since(t0)

	fmt.Printf("bench: %d values\n", *n)
	fmt.Printf("format: %.3fms (%.1f ns/op)\n", float64(tf.Microseconds())/1000, float64(tf.Nanoseconds())/float64(*n))
	fmt.Printf("parse:  %.3fms (%.1f ns/op)\n", float64(tp.Microseconds())/1000, float64(tp.Nanoseconds())/float64(*n))
	fmt.Printf("total:  %.3fms\n", float64((tf+tp).Microseconds())/1000)
}
