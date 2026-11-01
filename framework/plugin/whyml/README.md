# Go → WhyML prototype

This backend translates a restricted, type-checked Go file into executable
WhyML. It is available as a reusable Go package, the `go2whyml` CLI, and the
registered GPMA `whyml` plugin. Insertion sort is proved to terminate, produce
an ordered slice, and preserve the complete multiset of its input values.

## Quick start

Requirements: Go 1.25.2+, Java for the framework's ANTLR generation, Why3 and Z3.
The recorded runs use Why3 **1.8.0** and Z3 **4.13.3**. Install Why3/Z3 using
your package manager, then run `why3 config detect` and check
`why3 config list-provers`.

From the repository root:

```sh
cd framework
go generate ./...
go build -o build/go2whyml ./cmd/go2whyml
./build/go2whyml -prove \
  -o ../whyml-demo/generated/insertion_sort.mlw \
  ../whyml-demo/insertion_sort.go
```

The expected result is **48 goals Valid**. The generated file is checked
without manual edits. `-prove` returns a nonzero exit code for unknown goals,
timeouts, invalid goals, missing provers, or an empty proof report.

Translation and proof can also be separate:

```sh
./build/go2whyml -o sort.mlw ../whyml-demo/insertion_sort.go
why3 prove sort.mlw -a split_vc -P z3 -t 10
why3 ide sort.mlw
```

`-int-bits 32` selects 32-bit Go `int`; the default is explicitly 64-bit,
independent of the machine running the translator. Other CLI flags are
`-why3`, `-why3-config`, `-prover`, and `-time-limit` (seconds per obligation).
`-o -` writes WhyML to stdout and cannot be combined with `-prove`.

Run the complete demonstration, including negative tests, from the repository
root with `./whyml-demo/verify.sh`. `WHY3` and `WHY3_CONFIG` can select a custom
installation. Generated files and proof logs go into `whyml-demo/generated/`.

## Existing GPMA CLI

```sh
cd framework
make build
./build/framework file --plugin whyml \
  --plugin-config 'whyml:output:sort.mlw;prove:true;int_bits:64' \
  ../whyml-demo/insertion_sort.go
```

The plugin writes a `.mlw` sidecar, leaving the Go AST intact. Its `output`
option is distinct from the framework's positional Go output argument. With
no `output` option, the sidecar replaces the source filename's `.go` extension.
Additional configuration keys are `why3`, `why3_config`, `prover`, and
`time_limit`. Unknown options are errors. This is a **single-file** backend;
cross-file calls and whole-package verification are outside the prototype.

## Reusable API

```go
code, err := whyml.Translate("sort.go", source, whyml.Options{IntBits: 64})
// Handle err before writing or proving anything.
err = whyml.WriteFile("sort.mlw", code)
result, err := whyml.Prove(ctx, "sort.mlw", whyml.ProofOptions{Prover: "z3"})
// err == nil and result.Success mean every reported obligation was Valid.
```

Import `gpma/framework/plugin/whyml` from the framework module. `Generate`
also accepts `*data.ASTData`. The backend reads annotations from the AST's
comments with its own strict GPMA-subset parser. It deliberately does not
consume the general frontend's permissive fallback expressions: unsupported
specifications must not silently disappear from a verification task.

## Supported language and annotations

Programs support `int`, `bool`, and one `[]int` parameter per function; scalar
locals; zero initialization; lexical shadowing; direct acyclic calls within
the same file; scalar or void returns; `if`/`else`; nested condition-only,
three-clause, and unconditional `for` loops; unlabeled `break`/`continue`;
parallel assignment; `++`, `--`, `+=`, `-=`, `*=`; array access; `len`; and
integer arithmetic/comparisons and short-circuit Boolean expressions.
Scalar parameters are read-only in this prototype. Copy one into a local
when mutation is needed. There are no closures or addresses of loop variables,
so Go's per-iteration loop-variable identity is unobservable in this subset.

Annotations are normal Go comments:

```go
/*@
requires 0 <= n && n <= 100;
ensures \result == n;
*/
func Count(n int) int {
    i := 0
    /*@
    loop invariant 0 <= i && i <= n;
    loop variant n - i;
    */
    for i < n { i++ }
    return i
}
```

Supported clauses: `requires`, `ensures`, `assert`, `loop invariant`, and
`loop variant`. Put contracts immediately before functions, loop annotations
before loops, and assertions before statements. Every loop needs one integer
variant, so successful verification includes termination. Clauses may have
labels (`ensures ordered: sorted(a);`). Both `/*@ … */` and `//@ …` work,
including the block-comment formatting produced by `gofmt`.

Specification expressions support decimal mathematical integers, Boolean
connectives, `==>`, `<==>`, comparisons, `+`, `-`, `*`, indexing, `\result`,
and `\old` on parameter expressions. Quantifiers use GPMA's binder order:
`\forall k, l integer; predicate` (also `\exists`). Parenthesize quantified
expressions when combining them with other predicates. Chained comparisons
are rejected; write `0 <= i && i < n`. `min_int` and `max_int` denote the
selected machine bounds. Logical arithmetic is mathematical, as in ACSL.

Built-in specification calls map to Why3's standard library:

| GPMA | Meaning |
| --- | --- |
| `len(a)` | Slice length |
| `sorted(a)` | All valid indices `k < l` satisfy `a[k] <= a[l]` |
| `sorted_sub(a, lo, hi)` | Ordered half-open segment `[lo, hi)` |
| `permut_all(\old(a), a)` | Equal multisets, including multiplicities |
| `exchange(\old(a), a, i, j)` | Exchange the two elements and preserve all others |

Logic array reads are Why3 logical reads, not executable Go accesses. Guard
quantifiers with valid index ranges. There is no `assume`, user axiom, raw
WhyML injection, or admitted lemma facility.

## Semantic boundary

The proof applies to executions satisfying the contracts and generated
representation constraints. Unsupported syntax is rejected, not approximated.

| Difference | Treatment |
| --- | --- |
| Go orders calls left-to-right; WhyML applications evaluate arguments right-to-left | Every application and ordinary binary operation is lowered to explicit sequential `let` bindings. `CallOrder` proves `Pack(Bump(a), Bump(a)) == 12`. |
| Go leaves ordinary reads partly unordered relative to calls | An expression or assignment mixing a user call and slice indexing is conservatively rejected. Split it into statements. Calls cannot mutate caller scalar locals in this subset. |
| `&&` / `||` must short-circuit | Lowered to conditional expressions; the unchosen operand is never evaluated. |
| Go multiple assignment has two phases | Save all LHS indices and RHS values, then commit stores left-to-right. Both swaps and `i, a[i] = 1, 7` are tested. |
| Go `int` is finite and signed arithmetic may wrap; Why3 `int` is unbounded | Require bounded inputs, maintain bounded values, and generate overflow assertions for runtime `+`, `-`, `*`, negation, increments and compounds. Successful proofs cover **no-overflow executions**. Wrapping executions are not modelled. |
| Go integer division truncates toward zero and has a minimum-integer/-1 corner case | Runtime division and remainder are rejected. Go constant expressions are folded by `go/types`. Specification division/remainder are rejected too. |
| Go `[N]int` copies values; slices share backing storage | Fixed arrays are rejected. A single un-rebound slice parameter maps to one WhyML mutable array. Multiple slices, local slice aliases, slicing, allocation, `append`, `copy`, `cap`, and slice returns are rejected. |
| Nil versus empty slice | Both are represented by length zero; nil tests and operations observing this distinction are unsupported. |
| Go indexing can panic | Why3 array-access preconditions must prove every executed index valid. No `panic`, `recover`, or `defer` support. |
| Go `continue` executes the `for` post statement | A local exception exits the body; the post statement still runs. `break` bypasses the post; returns are caught at the function boundary. |
| Go locals have zero values and lexical scope | Emit explicit zero initialization and distinct names for each Go binding, using `go/types` object identity. |

Also rejected: package variables/imports, pointers, maps, structs, interfaces,
strings/floats, unsigned or explicitly sized integers, casts, variadic and
indirect calls, recursion, methods/generics, multi-value/named returns,
`range`, `switch`, labels/`goto`, concurrency, and unsupported annotation
clauses such as `assigns`, `assumes`, or custom predicate declarations.

References: [Go evaluation order](https://go.dev/ref/spec#Order_of_evaluation),
[assignments](https://go.dev/ref/spec#Assignments),
[integer overflow](https://go.dev/ref/spec#Integer_overflow), and
[WhyML syntax and semantics](https://www.why3.org/doc/syntaxref.html).

## Proof and trust

The outer insertion-sort invariant says the processed prefix is ordered and
the whole slice is a permutation of its entry state. The inner invariant says
every pair in the prefix is ordered except pairs whose right endpoint is the
element currently moving left. Each swap preserves the multiset; the inner
variant is `j`, and the outer variant is `len(a)-i`. On loop exit these give
the two requested postconditions.

The generated WhyML is proved with Why3/Z3, including the swap helper's body.
There are no hand-edited proof files or added axioms. The translator itself
is **not a verified compiler**: transfer of the WhyML result back to Go relies
on this documented translation boundary, the Go parser/type checker, this
translator, Why3 and its standard library, and the SMT solver. This prototype
is a working foundation for verification, not a claim to cover arbitrary Go.

## Tests

```sh
cd framework
go generate ./...
go test ./plugin/whyml ./cmd/go2whyml
go test -tags=integration -v ./plugin/whyml
cd ../whyml-demo
go test -v ./...
```

Integration tests require real Why3/Z3 and fail if unavailable. They prove
insertion sort and semantic examples at both integer widths, then require
proof failure for false postconditions/invariants, overflow, out-of-bounds
access, and a broken swap. Go tests compare the real algorithm against
`sort.Ints` on exhaustive and randomized inputs. See the demonstration's
`PROOF_REPORT.md` for the recorded results.
