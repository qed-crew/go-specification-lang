package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gpma/framework/plugin/whyml"
)

func run() error {
	output := flag.String("o", "", "output .mlw file (default: source basename.mlw; - for stdout)")
	bits := flag.Int("int-bits", 64, "target Go int width: 32 or 64")
	prove := flag.Bool("prove", false, "prove all generated obligations using Why3")
	why3 := flag.String("why3", "why3", "Why3 executable")
	config := flag.String("why3-config", "", "optional Why3 configuration file")
	prover := flag.String("prover", "z3", "Why3 prover selection")
	limit := flag.Int("time-limit", 10, "seconds per proof obligation")
	flag.Parse()
	if flag.NArg() != 1 {
		return fmt.Errorf("usage: go2whyml [flags] input.go")
	}
	if *prove && *output == "-" {
		return fmt.Errorf("-prove requires an output file")
	}
	source := flag.Arg(0)
	contents, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	code, err := whyml.Translate(source, contents, whyml.Options{IntBits: *bits})
	if err != nil {
		return err
	}
	if *output == "-" {
		fmt.Print(code)
		return nil
	}
	if *output == "" {
		*output = strings.TrimSuffix(source, filepath.Ext(source)) + ".mlw"
	}
	if err = whyml.WriteFile(*output, code); err != nil {
		return err
	}
	fmt.Printf("WhyML written to %s\n", *output)
	if *prove {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		r, err := whyml.Prove(ctx, *output, whyml.ProofOptions{Why3: *why3, Config: *config, Prover: *prover, TimeLimit: *limit})
		fmt.Print(r.Output)
		if err != nil {
			return err
		}
		fmt.Printf("Why3: %d goals Valid\n", r.Valid)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
