package whyml

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
)

// Go orders calls left-to-right but leaves the ordering of ordinary reads
// relative to those calls unspecified. Without a heap/effect analysis, reject
// any full expression/assignment mixing a call with a slice element access.
// Calls cannot modify scalar locals because pointers and globals are excluded.
func (g *generator) checkOrder(nodes ...ast.Node) {
	call, index := false, false
	for _, n := range nodes {
		if n == nil {
			continue
		}
		ast.Inspect(n, func(n ast.Node) bool {
			switch e := n.(type) {
			case *ast.IndexExpr:
				index = true
			case *ast.CallExpr:
				if id, ok := e.Fun.(*ast.Ident); ok {
					if _, ok := g.info.Uses[id].(*types.Func); ok {
						call = true
					}
				}
			}
			return true
		})
	}
	if call && index {
		g.fail(nodes[0], "slice access mixed with function calls has potentially unspecified Go evaluation order; split it into statements")
	}
}
func (g *generator) expr(e ast.Expr) string {
	if tv, ok := g.info.Types[e]; ok && tv.Value != nil {
		switch tv.Value.Kind() {
		case constant.Int:
			return "(" + tv.Value.ExactString() + ")"
		case constant.Bool:
			return tv.Value.ExactString()
		}
	}
	switch e := e.(type) {
	case *ast.ParenExpr:
		return g.expr(e.X)
	case *ast.Ident:
		o := g.info.Uses[e]
		if n, ok := g.names[o]; ok {
			return n
		}
		g.fail(e, "unsupported identifier %s", e.Name)
	case *ast.IndexExpr:
		a, i := g.fresh(), g.fresh()
		return fmt.Sprintf("(let %s = %s in let %s = %s in %s[%s])", a, g.expr(e.X), i, g.expr(e.Index), a, i)
	case *ast.UnaryExpr:
		x := g.fresh()
		v := g.expr(e.X)
		switch e.Op {
		case token.NOT:
			return fmt.Sprintf("(let %s = %s in not %s)", x, v, x)
		case token.ADD:
			return v
		case token.SUB:
			return fmt.Sprintf("(let %s = %s in %s)", x, v, g.checked("(- "+x+")"))
		}
		g.fail(e, "unsupported unary operator %s", e.Op)
	case *ast.BinaryExpr:
		l, r := g.fresh(), g.fresh()
		lv, rv := g.expr(e.X), g.expr(e.Y)
		if e.Op == token.LAND {
			return fmt.Sprintf("(let %s = %s in if %s then %s else false)", l, lv, l, rv)
		}
		if e.Op == token.LOR {
			return fmt.Sprintf("(let %s = %s in if %s then true else %s)", l, lv, l, rv)
		}
		op := e.Op.String()
		switch e.Op {
		case token.EQL:
			op = "="
		case token.NEQ:
			op = "<>"
		case token.ADD, token.SUB, token.MUL, token.LSS, token.LEQ, token.GTR, token.GEQ:
		default:
			g.fail(e, "unsupported binary operator %s (division, remainder, shifts and bitwise operations are outside the prototype)", e.Op)
			return "0"
		}
		v := "(" + l + " " + op + " " + r + ")"
		if e.Op == token.ADD || e.Op == token.SUB || e.Op == token.MUL {
			v = g.checked(v)
		}
		return fmt.Sprintf("(let %s = %s in let %s = %s in %s)", l, lv, r, rv, v)
	case *ast.CallExpr:
		id, ok := e.Fun.(*ast.Ident)
		if !ok || e.Ellipsis.IsValid() {
			g.fail(e, "only direct, non-variadic calls are supported")
			return "0"
		}
		obj := g.info.Uses[id]
		if b, ok := obj.(*types.Builtin); ok {
			if b.Name() == "len" && len(e.Args) == 1 && kind(g.info.TypeOf(e.Args[0])) == "array int" {
				return "(length " + g.expr(e.Args[0]) + ")"
			}
			g.fail(e, "only len([]int) is supported among Go builtins")
			return "0"
		}
		f, ok := obj.(*types.Func)
		if !ok || g.functions[f] == nil {
			g.fail(e, "only functions defined in the same file can be called")
			return "0"
		}
		var prefix, args []string
		for _, a := range e.Args {
			n := g.fresh()
			prefix = append(prefix, fmt.Sprintf("let %s = %s in", n, g.expr(a)))
			args = append(args, n)
		}
		if len(args) == 0 {
			args = append(args, "()")
		}
		return "(" + strings.Join(prefix, " ") + " " + g.names[f] + " " + strings.Join(args, " ") + ")"
	default:
		g.fail(e, "unsupported expression %T", e)
	}
	return "0"
}
func (g *generator) checked(value string) string {
	n := g.fresh()
	return fmt.Sprintf("(let %s = %s in assert { in_int %s }; %s)", n, value, n, n)
}
