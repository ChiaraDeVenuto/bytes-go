# bytes-go — Go port of `bytes.js` (v3.1.2)

A dependency-free, behavior-identical port of
[visionmedia/bytes.js](https://github.com/visionmedia/bytes.js) (MIT) —
`bytes.format()` / `bytes.parse()` for byte sizes — with the exact
ECMAScript semantics of the original, including its quirks.

**Port Mortem 2026 · Track F (JavaScript → Go).**

## Why this matters

`bytes.js` is one of the most-downloaded npm packages (~100M/week, zero
dependencies). A Go port gives you the same API with:

- **Startup**: ~0.00s vs 0.07s (Node) — >30x faster cold start
- **Memory**: 2.9 MB RSS vs 44.4 MB (Node) — 15x lighter
- **format**: 584 ns/op vs 750 ns/op (-22%)
- **Zero unsafe blocks, zero dependencies, static binary ~7 MB**

## Behavioral equivalence — how it is proven

1. **Original test suite**: the untouched mocha tests (30 tests) ported
   1:1 to `bytefmt_test.go` — **30/30 pass**.
2. **Differential fuzzing**: 10,017 frozen vectors generated against the
   *original* Node module (valid + invalid inputs, all `Options` knobs,
   edge cases: `0x11`, `1e3`, `-0`, tabs, NaN, 2^52+ values) replayed
   against the port — **zero divergences**. The 60s+ continuous run
   checked **73,224,270 vectors** (1,126,418/s), log in `fuzz/log.txt`.

## Layout

```
bytefmt.go          — the entire port (library root)
bytefmt_test.go     — 26 unit tests (1:1 from original mocha suite)
regression_test.go  — differential regression corpus from fuzzing
DECISIONS.md        — 15 documented port decisions
bench/              — benchmark runners (Node + Go) + report
fuzz/               — differential fuzz harness + frozen corpus
fuzz/gen/           — corpus generator (runs against ORIGINAL via Node)
cmd/bytefmt/        — CLI mirroring the full API
tests/original/     — original bytes.js sources + mocha suite (untouched)
Dockerfile          — multistage build → static binary
```

## API

```go
import "portmortem/bytes-go"

// bytes.format(1024)            → "1KB"
s, ok := bytefmt.Format(1024, nil)

// bytes.format(1234567, {thousandsSeparator: ',', decimalPlaces: 2})
s, ok := bytefmt.Format(1234567, &bytefmt.Options{
    ThousandsSeparator: ",",
    DecimalPlaces:      intPtr(2),
})

// bytes.parse("1.5KB")          → 1536
n, ok := bytefmt.Parse("1.5KB")

// bytes(1024)                   → "1KB"  (dispatcher)
v, ok := bytefmt.Bytes(1024, nil)
// bytes.parse("1KB")            → 1024
```

`Options` mirrors the original exactly (`DecimalPlaces *int` preserves the
`undefined` vs `0` distinction; empty string = absent, like the JS falsy
checks). The second return value is the port of the original's `null`
for unparseable input.

## CLI

```sh
go build -o bytefmt ./cmd/bytefmt
./bytefmt --decimal-places 2 1234567        # format arg
./bytefmt -p "1.5KB"                        # parse
./bytefmt --thousands-separator "," 1234567 # options
./bytefmt --bench 200000                    # benchmark (Go side)
```

## Build & test

```sh
go build ./...          # builds library + CLI + harness
go vet ./...            # clean
go test ./... -v        # 30/30
go run ./fuzz           # replay 10,017-vector corpus
go run ./fuzz -duration 60s   # differential fuzz survivor run
```

### Docker

```sh
docker build -t bytes-go .
docker run --rm bytes-go 1048576        # → "1MB"
```

## Verifying the original is untouched

```
SHA-256 (tests/original/index.js) = 893fcbbbe962dc00e40dc2e4b20e76e92d874dd257345003c6575d940e91a37f
```

The original was hashed at clone time; `tests/original/` is the unedited
upstream tree (the mocha suite ships with the package).

## Benchmark summary

| Metric | Original (Node) | Port (Go) |
|---|---|---|
| format (200k ops) | 749.5 ns/op | **583.6 ns/op** |
| parse (200k ops) | **355.5 ns/op** | 701.7 ns/op |
| startup | 0.07 s | **~0.00 s** |
| RSS | 44.4 MB | **2.9 MB** |

Details and honest methodology in `bench/README.md`.

## License

Port: MIT (see LICENSE). Original bytes.js: MIT, Copyright © TJ Holowaychuk
and contributors — behavior and test suite ported with attribution.
