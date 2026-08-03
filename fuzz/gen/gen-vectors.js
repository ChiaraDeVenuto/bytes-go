// Vector generator — runs the ORIGINAL bytes.js through Node and emits
// expected outputs for a randomized corpus. The Go port is then checked
// against these vectors (differential fuzzing, frozen oracle).
'use strict';

const bytes = require('/home/chiara/progetti/port-mortem/bytes-original/index.js');
const fs = require('fs');

const units = ['', 'b', 'B', 'kb', 'KB', 'mb', 'MB', 'gb', 'GB', 'tb', 'TB', 'pb', 'PB', 'bb', 'xx'];
const decimals = [undefined, 0, 1, 2, 3, 4];
const seps = ['', '.', ',', ' ', '_'];
const unitSeps = ['', ' ', '\t'];

function rng(seed) {
  let s = seed >>> 0;
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0;
    return s / 4294967296;
  };
}

function main() {
  const rand = rng(1337);
  const vectors = [];

  // format corpus
  for (let i = 0; i < 4000; i++) {
    const mag = Math.pow(1024, Math.floor(rand() * 6));
    const neg = rand() < 0.25 ? -1 : 1;
    const v = neg * mag * (0.0001 + rand() * 9999.9);
    const opts = {
      decimalPlaces: decimals[Math.floor(rand() * decimals.length)],
      fixedDecimals: rand() < 0.2,
      thousandsSeparator: seps[Math.floor(rand() * seps.length)],
      unitSeparator: unitSeps[Math.floor(rand() * unitSeps.length)],
      unit: units[Math.floor(rand() * units.length)],
    };
    const out = bytes.format(v, opts);
    vectors.push({ kind: 'format', input: v, opts: jsonOpts(opts), expect: out === null ? null : out });
  }

  // parse corpus
  const prefixes = ['', '-', '+'];
  const values = ['0', '1', '0.5', '1.5', '10.5', '1024', '1.0001', '999', '0.1', '123456'];
  const units2 = ['', 'b', 'kb', 'KB', 'Kb', 'kB', 'mb', 'MB', 'gb', 'GB', 'tb', 'TB', 'pb', 'PB'];
  for (let i = 0; i < 4000; i++) {
    const s = prefixes[Math.floor(rand() * 3)] +
      values[Math.floor(rand() * values.length)] +
      (rand() < 0.3 ? ' ' : '') +
      units2[Math.floor(rand() * units2.length)];
    const out = bytes.parse(s);
    vectors.push({ kind: 'parse', input: s, expect: out === null ? null : out });
  }

  // raw numbers through parse (incl. invalid types)
  const raw = [0, -1, 1, 10.5, 1024, -1024, NaN, Infinity, true, false, null, undefined, {}, [], 'foobar', '0x11', ''];
  for (const v of raw) {
    const out = bytes.parse(v);
    vectors.push({ kind: 'parse', input: jsSafe(v), expect: out === null ? null : out });
  }

  // dispatch corpus (bytes(value, opts))
  for (let i = 0; i < 2000; i++) {
    const kind = Math.floor(rand() * 3);
    let input;
    if (kind === 0) {
      input = (rand() < 0.5 ? '-' : '') + (rand() * 1e12).toFixed(2);
    } else if (kind === 1) {
      input = rand() * 1e15;
      if (rand() < 0.1) input = NaN;
    } else {
      input = ['', 'x', '1kb', '1 TB', true, null, undefined, {}][Math.floor(rand() * 8)];
    }
    const opts = { thousandsSeparator: rand() < 0.5 ? ',' : '' };
    const out = bytes(input, opts);
    vectors.push({ kind: 'dispatch', input: jsSafe(input), opts: { thousandsSeparator: opts.thousandsSeparator }, expect: out === null ? null : out });
  }

  fs.writeFileSync('fuzz/vectors.json', JSON.stringify(vectors));
  console.log('wrote', vectors.length, 'vectors');
}

function jsonOpts(o) {
  const out = {};
  if (o.decimalPlaces !== undefined) out.decimalPlaces = o.decimalPlaces;
  if (o.fixedDecimals) out.fixedDecimals = true;
  if (o.thousandsSeparator !== '') out.thousandsSeparator = o.thousandsSeparator;
  if (o.unitSeparator !== '') out.unitSeparator = o.unitSeparator;
  if (o.unit !== '') out.unit = o.unit;
  return out;
}

function jsSafe(v) {
  if (typeof v === 'number') {
    if (Number.isNaN(v)) return { __nan: true };
    if (v === Infinity) return { __inf: 1 };
    if (v === -Infinity) return { __inf: -1 };
    return v;
  }
  return v;
}

main();
