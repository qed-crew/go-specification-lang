package whyml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectUnsupportedSemantics(t *testing.T) {
	cases := map[string]struct{ source, diagnostic string }{
		"aliasing":             {"func f(a,b []int) {}", "at most one"},
		"local alias":          {"func f(a []int) { b:=a; b[0]=1 }", "local slice"},
		"fixed array":          {"func f(a [2]int) {}", "unsupported type"},
		"globals":              {"var a int; func f() {}", "only function"},
		"goroutine":            {"func f() { go f() }", "recursive"},
		"recursion":            {"func f() { f() }", "recursive"},
		"indirect recursion":   {"func f() { g() }; func g() { f() }", "recursive"},
		"overflow constant":    {"func f() int { return 9223372036854775808 }", "Go type check"},
		"division":             {"func f(x,y int) int { return x/y }", "unsupported binary"},
		"pointer":              {"func f(x *int) {}", "unsupported type"},
		"rebind parameter":     {"func f(x int) { x++ }", "rebinding"},
		"append":               {"func f(a []int) { a=append(a,1) }", "rebinding"},
		"range":                {"func f(a []int) { for i:=range a {a[i]=0} }", "unsupported statement"},
		"missing variant":      {"func f() { for {} }", "loop variant"},
		"effectful read":       {"func bump(a []int) int { a[0]++;return a[0] };func f(a []int) int{return a[0]+bump(a)}", "unspecified Go evaluation order"},
		"effectful assignment": {"func bump(a []int) int{return a[0]};func f(a []int){a[0]=bump(a)}", "unspecified Go evaluation order"},
		"unknown annotation":   {"/*@ assumes x > 0; */\nfunc f(x int) {}", "unsupported annotation"},
		"assigns rejected":     {"/*@ assigns \\nothing; */\nfunc f() {}", "unsupported annotation"},
		"malformed contract":   {"/*@ ensures \\result == ; */\nfunc f()int{return 0}", "expected expression"},
		"unknown spec name":    {"/*@ ensures typo == 0; */\nfunc f() {}", "unknown specification variable"},
		"misplaced annotation": {"func f(){ /*@ ensures \\true; */ return }", "misplaced"},
		"chained comparison":   {"/*@ requires 0 <= x <= 5; */\nfunc f(x int) {}", "chained comparisons"},
		"old local":            {"func f(){ x:=0; /*@ assert \\old(x) == 0; */ x++ }", "only parameters"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Translate("bad.go", []byte("package p\n"+tc.source), Options{})
			if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
				t.Fatalf("expected %q, got %v", tc.diagnostic, err)
			}
		})
	}
}

func TestExamplesAndDeterminism(t *testing.T) {
	for _, file := range []string{"insertion_sort.go", "semantics.go"} {
		t.Run(file, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join("../../..", "whyml-demo", file))
			if err != nil {
				t.Fatal(err)
			}
			first, err := Translate(file, src, Options{})
			if err != nil {
				t.Fatal(err)
			}
			second, err := Translate(file, src, Options{})
			if err != nil || first != second {
				t.Fatal("translation must be deterministic", err)
			}
			if strings.Contains(first, "assume ") || strings.Contains(first, "axiom ") {
				t.Fatal("generated code must not assume its obligations")
			}
		})
	}
}

func TestIntWidthAndAtomicOutput(t *testing.T) {
	src := []byte("package p; func f() int { return 2147483648 }")
	if _, err := Translate("f.go", src, Options{IntBits: 32}); err == nil {
		t.Fatal("32-bit typecheck accepted out-of-range int")
	}
	if _, err := Translate("f.go", src, Options{IntBits: 64}); err != nil {
		t.Fatal(err)
	}
	if _, err := Translate("f.go", src, Options{IntBits: 16}); err == nil {
		t.Fatal("unsupported target width")
	}
	path := filepath.Join(t.TempDir(), "out.mlw")
	if err := WriteFile(path, "new"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "new" {
		t.Fatal(string(got))
	}
	if err := WriteFile(strings.TrimSuffix(path, ".mlw")+".go", "bad"); err == nil {
		t.Fatal("must not write generated WhyML over Go source")
	}
}

func TestSpecParserRejectsTruncation(t *testing.T) {
	for _, s := range []string{"", "requires ;", "ensures (x > 0;", "ensures x > 0", "ensures x > 0; garbage;", "ensures \\forall k integer; ;"} {
		if _, err := parseSpec(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}
