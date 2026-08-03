# I Ported bytes.js to Go in 48 Hours. Here's What 68 Million Fuzz Checks Taught Me.

*Port Mortem 2026, Track F — Go port of bytes.js v3.1.2*

---

`bytes.js` is one of the most-downloaded npm packages. ~100M weekly downloads, zero dependencies, ~170 lines of code. It does one thing: convert between numbers and human-readable byte strings (`"1.5KB"` ↔ `1536`).

For Port Mortem 2026, I ported it to Go. Not a rewrite. Not a "similar library." A **behavior-identical** port where the original mocha test suite passes 30/30 against the Go binary, and 68 million differential fuzz checks show zero divergences.

Here's what I learned.

## The easy part that wasn't

The core logic is simple: format divides by 1024 and picks a unit, parse matches a regex and multiplies back. In Go, this is maybe 300 lines.

The hard part is the edge cases that JavaScript handles silently.

### The `toFixed` trap

`bytes.js` calls `val.toFixed(decimalPlaces)` for decimal formatting. The naive Go equivalent is `math.Floor(val*10^n + 0.5)`. It looks correct. It isn't.

Three distinct divergences, all discovered by fuzzing:

**Double rounding.** When `val*10^n` lands within one ulp of a half, the float64 multiplication rounds once, then `+0.5` rounds again. Values like `2.6586555755536324e12` produce `633` instead of `632`.

**Ties and negatives.** V8 rounds `(-2.5).toFixed(0)` to `"-3"` (half-away-from-zero on the absolute value). `math.Floor(-2.5 + 0.5)` gives `-2`.

**Sign of zero.** `(-3.7e-12).toFixed(2)` returns `"-0.00"` in V8. The sign of the input is preserved even when the result is zero. Go's `strconv.FormatFloat` drops it.

The fix: for values below 2^52, use FMA (fused multiply-add) with exact-residual correction on the absolute value, then reapply the sign. Beyond 2^52, where float64 halfway values aren't representable at all, use `math/big.Rat` to compute the exact rational rounding.

This took longer than everything else combined.

### The `parseInt` rabbit hole

When the regex doesn't match, `bytes.js` falls back to `parseInt(val, 10)`. But JavaScript's `parseInt` has its own quirks: it skips JS whitespace (not just ASCII), forces base 10 (so `'0x11'` returns `NaN`, not `17`), and accepts `'1.5kb'` as `1` (stops at the first non-digit).

Go's `strconv.ParseInt` doesn't share any of these semantics. So there's a small `jsParseInt` function that replicates the ECMAScript algorithm directly.

## Differential fuzzing: let the oracle talk

The real insight of this project is the proof strategy. Instead of writing new tests that encode my *understanding* of the original, I generate test vectors *from* the original and replay them against the port.

Here's how it works:

1. **Corpus generation** (`fuzz/gen/gen-vectors.js`): run the *original* bytes.js with 10,000+ inputs (valid, invalid, edge cases, all `Options` combinations) and record every input-output pair.
2. **Fuzz harness** (`fuzz/harness.go`): replay the frozen corpus against the Go port. Any divergence = bug.
3. **Continuous run**: let the harness run for 60+ seconds with randomized inputs to catch what the frozen corpus misses.

The result: **68,175,702 vectors checked, zero divergences.** The corpus is frozen and committed — anyone can rerun the exact same evidence.

This approach has a property I like: **the AI cannot bias the oracle.** The test vectors are generated from the original Node module. The Go code either produces the same output or it doesn't. No opinions, no "close enough," just bit-for-bit equivalence.

## What the benchmarks actually show

| Metric | Node.js | Go port | Delta |
|--------|---------|---------|-------|
| `format` (200k ops) | 749.5 ns/op | **583.6 ns/op** | -22% |
| `parse` (200k ops) | **355.5 ns/op** | 701.7 ns/op | +97% |
| Cold start | 0.07 s | **~0.00 s** | >30x |
| RSS | 44.4 MB | **2.9 MB** | 15x |

The `parse` regression is honest. V8's native regexp is faster than anything I can write in Go for this specific pattern. The manual matcher (702 ns/op) is 1.8x faster than Go's `regexp` package (1260 ns/op), but still can't match V8.

I could have hidden this. The judges care more about behavioral equivalence than microbenchmarks. But hiding it would be worse than losing the benchmark: the whole point of the project is that the port *actually behaves like the original*, including its performance characteristics.

## Zero unsafe, zero dependencies

The entire port uses only the Go standard library. No `unsafe`, no `cgo`, no third-party packages. `go.mod` requires nothing. The static binary is ~7 MB.

This wasn't a constraint of the competition — it was a design choice. If I'm claiming behavioral equivalence, I want the smallest possible surface area for bugs. Every dependency is a potential source of divergent behavior.

## What I'd do differently

**Start with the corpus, not the code.** I wrote the port first and generated the corpus second. If I'd generated the corpus first, I would have found the `toFixed` and `parseInt` issues immediately instead of discovering them through hours of fuzzing.

**The `math/big.Rat` path should be the default.** It's slower but correct. I spent hours tuning the FMA path to handle edge cases that `big.Rat` handles trivially. For a library where correctness is the primary goal, the slower-but-correct path should be the baseline.

**Budget time for the boring stuff.** The actual port took maybe 12 hours. The fuzz corpus generation, the benchmark harness, the Dockerfile, the documentation — that took 30+ hours. The "boring" infrastructure is what makes the proof credible.

## The numbers

- **48 hours** total (including sleep, eating, and questioning my life choices)
- **~300 lines** of Go (the actual port)
- **184 lines** of decisions documented
- **10,017** frozen fuzz vectors
- **68,175,702** differential checks, zero divergences
- **30/30** original mocha tests passing
- **0** unsafe blocks, **0** dependencies

## Try it

```sh
git clone https://github.com/ChiaraDeVenuto/bytes-go
cd bytes-go
go test ./...          # 30/30
go run ./fuzz          # replay corpus
go run ./fuzz -duration 60s  # full fuzz run
```

Or use the CLI:

```sh
go build -o bytefmt ./cmd/bytefmt
./bytefmt 1048576                # → "1MB"
./bytefmt -p "1.5GB"             # → 1610612736
```

---

*Built during Port Mortem 2026 (Track F). AI-assisted code generation, human-verified correctness. The proof is in the fuzz log.*
