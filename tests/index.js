'use strict';

// Thin Node adapter that re-points the ORIGINAL bytes.js mocha suite at the
// Go port. The suite in tests/original/ does `require('..')`, which resolves
// to this file. Every call is forwarded synchronously to the Go binary via
// `execFileSync`; the Go side does the actual work.
//
// The adapter touches NOTHING in tests/original/ — the hashed suite runs
// as-is. This is the "thin adapter" the Port Mortem rules describe for
// running the original test suite against the port's artifact.
//
// Usage:
//   BYTEFMT_BIN=./bin/bytefmt npx mocha tests/original/*.js

const { execFileSync } = require('child_process');

const BIN = process.env.BYTEFMT_BIN || './bin/bytefmt';

function run(args, input) {
  try {
    return execFileSync(BIN, args, { input, encoding: 'utf8', timeout: 10000 }).trim();
  } catch (e) {
    if (e.status === 1) return 'null'; // Go side: invalid input -> null
    throw e;
  }
}

function toArgs(opts) {
  const a = [];
  if (!opts) return a;
  if (opts.decimalPlaces !== undefined && opts.decimalPlaces !== null) {
    a.push('--decimal-places', String(opts.decimalPlaces));
  }
  if (opts.fixedDecimals) a.push('--fixed');
  if (opts.thousandsSeparator) a.push('--thousands-separator', opts.thousandsSeparator);
  if (opts.unitSeparator) a.push('--unit-separator', opts.unitSeparator);
  if (opts.unit) a.push('--unit', opts.unit);
  return a;
}

// bytes.parse(value)
function parse(value) {
  // Mirror the original: parse(number) returns the number itself.
  if (typeof value === 'number' && !isNaN(value)) return value;
  if (typeof value !== 'string') return null;
  const out = run(['-p', '--', value]);
  return out === 'null' ? null : Number(out);
}

// bytes.format(value, options)
function format(value, opts) {
  if (typeof value !== 'number' || !isFinite(value)) return null;
  const out = run(toArgs(opts).concat(['--', String(value)]));
  return out === 'null' ? null : out;
}

// bytes(value, options) — dispatcher
function bytes(value, opts) {
  if (typeof value === 'string') return parse(value);
  if (typeof value === 'number' && isFinite(value)) return format(value, opts);
  return null;
}

module.exports = bytes;
module.exports.parse = parse;
module.exports.format = format;
