package whyml

import (
	"fmt"
	"strings"
)

func (g *generator) logic(f *formula, result, old bool, bound map[string]string) string {
	if f == nil {
		g.fail(g.fn, "missing specification expression")
		return "false"
	}
	arg := func(i int) string { return g.logic(f.args[i], result, old, bound) }
	if strings.HasPrefix(f.op, "call:") {
		n := strings.TrimPrefix(f.op, "call:")
		if n == `\old` {
			if len(f.args) != 1 {
				g.fail(g.fn, "\\old requires one argument")
				return "false"
			}
			return "(old " + g.logic(f.args[0], false, true, bound) + ")"
		}
		arity := map[string]int{"len": 1, "sorted": 1, "sorted_sub": 3, "permut_all": 2, "exchange": 4}
		if a, ok := arity[n]; !ok || a != len(f.args) {
			g.fail(g.fn, "unsupported specification call or arity: %s", n)
			return "false"
		}
		if n == "len" {
			n = "length"
		}
		args := []string{n}
		for i := range f.args {
			args = append(args, arg(i))
		}
		return "(" + strings.Join(args, " ") + ")"
	}
	switch f.op {
	case "index":
		return "(" + arg(0) + ")[" + arg(1) + "]"
	case `\forall`, `\exists`:
		inner := map[string]string{}
		for k, v := range bound {
			inner[k] = v
		}
		var names []string
		for _, v := range f.vars {
			if _, dup := inner[v]; dup {
				g.fail(g.fn, "duplicate or shadowed quantified variable %s", v)
			}
			n := g.fresh() + "_" + v
			inner[v] = n
			names = append(names, n)
		}
		return "(" + strings.TrimPrefix(f.op, `\`) + " " + strings.Join(names, " ") + ":int. " + g.logic(f.args[0], result, old, inner) + ")"
	case `\result`:
		if !result {
			g.fail(g.fn, "\\result is only supported in ensures")
		}
		return "result"
	case `\true`, "true":
		return "true"
	case `\false`, "false":
		return "false"
	case "unary!":
		return "(not " + arg(0) + ")"
	case "unary-":
		return "(- " + arg(0) + ")"
	case "unary+":
		return arg(0)
	case "/", "%":
		g.fail(g.fn, "division/remainder are unsupported in specifications and code")
		return "0"
	}
	if len(f.args) == 2 {
		op := map[string]string{"&&": "/\\", "||": "\\/", "==>": "->", "<==>": "<->", "==": "=", "!=": "<>"}[f.op]
		if op == "" {
			op = f.op
		}
		return "(" + arg(0) + " " + op + " " + arg(1) + ")"
	}
	if len(f.op) > 0 && f.op[0] >= '0' && f.op[0] <= '9' {
		return f.op
	}
	if n, ok := bound[f.op]; ok {
		return n
	}
	if f.op == "max_int" {
		return g.max
	}
	if f.op == "min_int" {
		return "(" + g.min + ")"
	}
	if b, ok := g.lookup(f.op); ok {
		if old && !b.param {
			g.fail(g.fn, "\\old may reference only parameters, not local %s", f.op)
		}
		return b.name
	}
	g.fail(g.fn, "unknown specification variable %s", fmt.Sprintf("%q", f.op))
	return "false"
}
