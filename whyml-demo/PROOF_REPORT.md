# Recorded verification — 2026-11-01

The prototype was run in the attached Linux/amd64 workspace with Go 1.25.2,
Why3 1.8.0, and Z3 4.13.3. These are actual solver runs, not expected output.
The final verification script disabled Go's test cache.

## Formal results

| Translated Go source | Go `int` width | Valid obligations | Result |
| --- | ---: | ---: | --- |
| `insertion_sort.go` | 32 | 48 / 48 | Pass |
| `insertion_sort.go` | 64 | 48 / 48 | Pass |
| `semantics.go` | 32 | 80 / 80 | Pass |
| `semantics.go` | 64 | 80 / 80 | Pass |

All **256 obligations** in these four configurations were Valid. This includes
the swap implementation, both sorting postconditions, loop invariant
initialization/preservation, termination variants, array bounds, and machine
integer range checks. No generated proof file was manually edited, and no
custom axioms or assumptions were added.

The sorting postconditions are:

```text
ensures { sorted a }
ensures { permut_all (old a) a }
```

These use Why3's standard definitions of pairwise ordering and full multiset
permutation. Both the helper and caller must pass: proving only the caller
would leave the helper contract unverified.

The 64-bit generated programs and complete per-goal solver output are in:

- `generated/insertion_sort.mlw` and `generated/insertion_sort.proof.log`
- `generated/semantics.mlw` and `generated/semantics.proof.log`
- `generated/integration.log` records both widths and the negative tests.

## Additional validation

- The real Go insertion sort agreed with `sort.Ints` on all 3,280 arrays of
  lengths 0–7 over `{-1, 0, 1}`, plus 1,000 deterministic random arrays.
- Edge tests covered nil/empty slices, minimum/maximum machine integers,
  duplicate values, and preservation of backing-array elements outside the
  sorted subslice.
- Executable Go and translated WhyML semantic examples covered calls with
  shared mutable state, short-circuiting, parallel assignment, `continue`
  executing the post statement, `break`, early returns, and lexical shadowing.
- Negative integration tests required proof failure for a wrong postcondition,
  overflow, out-of-bounds access, a false loop invariant, and a corrupted swap.
  A non-Valid/unknown/timeout result is a failure to certify, not a proof of
  incorrectness; the runner rejects all such outcomes.
- Targeted unit tests, `go vet`, and builds of both CLIs passed.
- The actual GPMA `file --plugin whyml` path produced and proved the sort:
  `Why3: 48 goals Valid`.
- The standalone CLI returned failure for an unproved postcondition. The GPMA
  CLI returned failure for an unsupported fixed-array parameter and reported
  a misplaced contract without panicking.

The complete existing repository test suite was not run; validation targeted
the new backend and its CLI/framework integration.

## Reproduce

With Go, Java, Why3 and Z3 installed and `why3 config detect` completed:

```sh
./whyml-demo/verify.sh
```

For a custom Why3 installation, set `WHY3` to its executable and optionally
`WHY3_CONFIG` to its configuration file. The included logs were created with
`why3 prove … -a split_vc -P z3 -t 10`.

SHA-256 of the verified sources and generated 64-bit programs:

```text
4d2ae71338ccbff6f7801e642517c3f2c587ad4949d69cc3321a3d049353f9f2  insertion_sort.go
ca11d4571a6abdb0bf18b11d922d7f4c0bfc16248fbd245508c366f4b8f38953  generated/insertion_sort.mlw
94773cd4bcc6f252d7c8bb09dadb13ac3ed2818b24ec40da7d9421f87b1dcead  semantics.go
dae21f6a3573d1e09c5a8811773092b0ca6c55d8a831c10ec14d82567161528a  generated/semantics.mlw
```

The result concerns the generated WhyML and the documented Go subset. The
translation implementation is not itself formally verified. In particular,
overflowing executions and unsupported slice-aliasing cases are outside its
scope; see `framework/plugin/whyml/README.md` for the semantic boundary.
