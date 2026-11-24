package dynamicchecks_test

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestDynamicChecksClauseAssertions(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		clause    string
	}{
		{name: "switch case", statement: "switch 1", clause: "case 1"},
		{name: "switch default", statement: "switch 1", clause: "default"},
		{name: "type switch case", statement: "switch any(1).(type)", clause: "case int"},
		{name: "select case", statement: "select", clause: "case make(chan int, 1) <- 1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// The expression switch case reproduces issue #75.
			source := fmt.Sprintf(`package main
func main() {
/*@ assert \true; */
_ = 0
%s {
%s:
/*@ assert \false; */
return
}
}
`, test.statement, test.clause)
			path := filepath.ToSlash(filepath.Join(t.TempDir(), "input.go"))
			if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := runDynamicChecksPlugin(path)
			if err != nil {
				t.Fatalf("instrument assertions: %v", err)
			}

			want := fmt.Sprintf(`package main

import checks "gpma/framework/plugin/dynamic_checks/check"

func main() {
	checks.CheckAt(true, %[1]q, 3, "assertion")
	_ = 0
	%[2]s {
	%[3]s:
		checks.CheckAt(false, %[1]q, 7, "assertion")
		return
	}
}
`, path, test.statement, test.clause)
			if got, want := formatGoWithoutPositions(t, got), formatGoWithoutPositions(t, want); got != want {
				t.Errorf("generated Go mismatch\n\n--- want ---\n%s\n--- got ---\n%s", want, got)
			}
		})
	}
}

func formatGoWithoutPositions(t *testing.T, source string) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		t.Fatalf("parse Go output: %v\n%s", err, source)
	}

	// Use an empty FileSet so source positions do not preserve blank lines.
	var formatted bytes.Buffer
	if err := format.Node(&formatted, token.NewFileSet(), file); err != nil {
		t.Fatalf("format Go output: %v", err)
	}
	return formatted.String()
}
