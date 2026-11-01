package whyml

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ProofOptions struct {
	Why3, Config, Prover string
	TimeLimit            int
}
type ProofResult struct {
	Valid   int    `json:"valid"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
}

// Prove checks every generated goal. Unknown, timeout, invalid, absent goals,
// missing provers and subprocess failures are all failures, never success.
func Prove(ctx context.Context, filename string, options ProofOptions) (ProofResult, error) {
	if options.Why3 == "" {
		options.Why3 = "why3"
	}
	if options.Prover == "" {
		options.Prover = "z3"
	}
	if options.TimeLimit == 0 {
		options.TimeLimit = 10
	}
	if options.TimeLimit < 0 {
		return ProofResult{}, fmt.Errorf("proof time limit must be positive")
	}
	args := []string{}
	if options.Config != "" {
		args = append(args, "-C", options.Config)
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return ProofResult{}, err
	}
	args = append(args, "prove", abs, "-a", "split_vc", "-P", options.Prover, "-t", strconv.Itoa(options.TimeLimit))
	out, err := exec.CommandContext(ctx, options.Why3, args...).CombinedOutput()
	r := ProofResult{Output: string(out)}
	results := regexp.MustCompile(`(?m)^Prover result is: ([^\r\n]+)`).FindAllStringSubmatch(r.Output, -1)
	allValid := true
	for _, m := range results {
		if strings.HasPrefix(m[1], "Valid (") {
			r.Valid++
		} else {
			allValid = false
		}
	}
	r.Success = err == nil && allValid && r.Valid > 0
	if !r.Success {
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		if err != nil {
			return r, fmt.Errorf("Why3 did not prove all goals: %w", err)
		}
		return r, fmt.Errorf("Why3 did not report a nonempty set of entirely Valid goals")
	}
	return r, nil
}

// WriteFile atomically replaces a generated .mlw file after successful translation.
func WriteFile(filename, code string) error {
	if filepath.Ext(filename) != ".mlw" {
		return fmt.Errorf("output must have a .mlw extension")
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(filename), ".go2whyml-*.mlw")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.WriteString(code); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Chmod(f.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(f.Name(), filename)
}
