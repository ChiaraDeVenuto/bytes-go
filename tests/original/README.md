# Original bytes.js v3.1.2 - unedited upstream tree (hashed at clone time)

- `index.js` - the original module, byte-identical to
  https://github.com/visionmedia/bytes.js@v3.1.2 (MIT).
- `bytes.js`, `byte-format.js`, `byte-parse.js` - the upstream mocha
  test suite shipping with the package, untouched.

```
SHA-256 (index.js) =
893fcbbbe962dc00e40dc2e4b20e76e92d874dd257345003c6575d940e91a37f
```

These files are never modified. The port lives at the repository root
(`bytefmt.go`); equivalence is verified by `bytefmt_test.go` (1:1 port
of the suite below) and the differential fuzz harness (`fuzz/`).
