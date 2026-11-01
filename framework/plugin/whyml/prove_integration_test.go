//go:build integration

package whyml

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Deliberately fails, rather than skips, if Why3 or its prover is unavailable.
func verifySource(t *testing.T, source []byte, bits int, want bool) {
	t.Helper()
	code, err := Translate("test.go", source, Options{IntBits: bits})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.mlw")
	if err = WriteFile(path, code); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	limit := 10
	if !want {
		limit = 1
	}
	r, err := Prove(ctx, path, ProofOptions{Why3: os.Getenv("WHY3"), Config: os.Getenv("WHY3_CONFIG"), TimeLimit: limit})
	if want && (err != nil || !r.Success) {
		t.Fatalf("proof failed: %v\n%s\n%s", err, r.Output, code)
	}
	if !want && (err == nil || r.Success) {
		t.Fatalf("incorrect program was verified: %+v", r)
	}
	if !want && !strings.Contains(r.Output, "Prover result is:") {
		t.Fatalf("negative test failed before reaching the prover: %v\n%s", err, r.Output)
	}
	t.Logf("%d Valid goals; expected success = %v", r.Valid, want)
}
func TestWhy3Examples(t *testing.T) {
	for _, file := range []string{"insertion_sort.go", "semantics.go"} {
		for _, bits := range []int{32, 64} {
			t.Run(file+"/"+strconv.Itoa(bits), func(t *testing.T) {
				src, err := os.ReadFile(filepath.Join("../../..", "whyml-demo", file))
				if err != nil {
					t.Fatal(err)
				}
				verifySource(t, src, bits, true)
			})
		}
	}
}
func TestWhy3RejectsIncorrectPrograms(t *testing.T) {
	cases := map[string]string{
		"wrong postcondition": "/*@ ensures \\result == x+2; */ func f(x int) int {return x}",
		"integer overflow":    "/*@ requires x == max_int; ensures \\result > x; */func f(x int) int {return x+1}",
		"out of bounds":       "func f(a []int) int {return a[0]}",
		"false invariant":     "func f(n int){ /*@ loop invariant i == 0; loop variant n-i; */ for i:=0;i<n;i++{} }",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) { verifySource(t, []byte("package p\n"+body), 64, false) })
	}
}
func TestWhy3RejectsBrokenSort(t *testing.T) {
	src, err := os.ReadFile("../../../whyml-demo/insertion_sort.go")
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(src), "a[i], a[j] = a[j], a[i]", "a[i], a[j] = a[j], a[j]", 1)
	verifySource(t, []byte(broken), 64, false)
}
