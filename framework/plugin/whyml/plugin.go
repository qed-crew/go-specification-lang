package whyml

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gpma/framework/data"
	"gpma/framework/plugin"
)

func init() { plugin.Register("whyml", func() plugin.Plugin { return &Plugin{} }) }

type Plugin struct {
	Options      Options
	ProofOptions ProofOptions
	Output       string
	Verify       bool
}

func (*Plugin) Name() string { return "whyml" }
func (p *Plugin) Configure(params map[string]interface{}) error {
	for k, v := range params {
		switch k {
		case "output", "why3", "why3_config", "prover":
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("%s must be a string", k)
			}
			switch k {
			case "output":
				p.Output = s
			case "why3":
				p.ProofOptions.Why3 = s
			case "why3_config":
				p.ProofOptions.Config = s
			case "prover":
				p.ProofOptions.Prover = s
			}
		case "prove":
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("prove must be a bool")
			}
			p.Verify = b
		case "int_bits", "time_limit":
			n, ok := v.(int)
			if !ok {
				return fmt.Errorf("%s must be an integer", k)
			}
			if k == "int_bits" {
				if n != 32 && n != 64 {
					return fmt.Errorf("int_bits must be 32 or 64")
				}
				p.Options.IntBits = n
			} else {
				if n <= 0 {
					return fmt.Errorf("time_limit must be positive")
				}
				p.ProofOptions.TimeLimit = n
			}
		default:
			return fmt.Errorf("unknown whyml option %q", k)
		}
	}
	return nil
}
func (p *Plugin) Process(d *data.ASTData) *plugin.PluginResult {
	r := plugin.NewPluginResult()
	code, err := Generate(d, p.Options)
	if err != nil {
		return r.WithFatalError(err)
	}
	output := p.Output
	if output == "" {
		source := d.FileSet.Position(d.File.Pos()).Filename
		if source == "" {
			return r.WithFatalError(fmt.Errorf("whyml: output is required for an unnamed source"))
		}
		output = strings.TrimSuffix(source, filepath.Ext(source)) + ".mlw"
	}
	if err = WriteFile(output, code); err != nil {
		return r.WithFatalError(err)
	}
	fmt.Printf("WhyML written to %s\n", output)
	if p.Verify {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		proof, err := Prove(ctx, output, p.ProofOptions)
		fmt.Print(proof.Output)
		if err != nil {
			return r.WithFatalError(err)
		}
		fmt.Printf("Why3: %d goals Valid\n", proof.Valid)
	}
	return r
}
