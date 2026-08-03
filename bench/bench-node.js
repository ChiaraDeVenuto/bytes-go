// Benchmark runner for the ORIGINAL bytes.js (Node) — port-mortem bench.
// Usage: node bench/bench-node.js [N]
'use strict';
const bytes = require('../../bytes-original/index.js');
const N = Number(process.argv[2] || 200000);

// deterministic LCG, same seed as the Go bench (42)
let seed = 42;
function rnd() {
  seed = (seed * 1664525 + 1013904223) >>> 0;
  return seed / 4294967296;
}

function main() {
  const t0 = process.hrtime.bigint();
  for (let i = 0; i < N; i++) {
    bytes.format(rnd() * 1e15);
  }
  const t1 = process.hrtime.bigint();

  for (let i = 0; i < N; i++) {
    bytes.parse((rnd() * 1e6).toFixed(3));
  }
  const t2 = process.hrtime.bigint();

  const fmtNs = Number(t1 - t0);
  const parseNs = Number(t2 - t1);
  console.log(`bench: ${N} values`);
  console.log(`format: ${(fmtNs / 1e6).toFixed(2)} ms (${(fmtNs / N).toFixed(1)} ns/op)`);
  console.log(`parse:  ${(parseNs / 1e6).toFixed(2)} ms (${(parseNs / N).toFixed(1)} ns/op)`);
  console.log(`total:  ${((fmtNs + parseNs) / 1e6).toFixed(2)} ms`);
  console.log(`rss_mb: ${(process.memoryUsage().rss / 1048576).toFixed(1)}`);
  console.log(`version: bytes.js v3.1.2 (Node ${process.version})`);
}

main();