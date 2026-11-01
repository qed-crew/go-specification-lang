package whyml

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

func (g *generator) statements(stmts []ast.Stmt) {
	for _, s := range stmts {
		if g.err != nil {
			return
		}
		g.statement(s)
	}
}
func (g *generator) block(b *ast.BlockStmt) {
	g.push()
	g.line("(")
	g.indent++
	g.statements(b.List)
	g.line("()")
	g.indent--
	g.line(")")
	g.pop()
}
func (g *generator) statement(s ast.Stmt) {
	p := g.d.FileSet.Position(s.Pos())
	g.line("(* %s:%d:%d *)", strings.ReplaceAll(p.Filename, "*)", "* )"), p.Line, p.Column)
	for _, c := range g.specs[s] {
		if c.kind == "assert" {
			g.line("assert { %s };", g.logic(c.expr, false, false, nil))
		}
	}
	switch s := s.(type) {
	case *ast.AssignStmt:
		g.assign(s)
	case *ast.IncDecStmt:
		g.checkOrder(s.X)
		lhs := g.location(s.X, false)
		op := "+"
		if s.Tok == token.DEC {
			op = "-"
		}
		g.line("%s;", lhs.write(g.checked("("+lhs.read+" "+op+" 1)")))
	case *ast.DeclStmt:
		d, ok := s.Decl.(*ast.GenDecl)
		if !ok || d.Tok != token.VAR {
			g.fail(s, "only var declarations are supported inside functions")
			return
		}
		for _, sp := range d.Specs {
			v := sp.(*ast.ValueSpec)
			var nodes []ast.Node
			for _, e := range v.Values {
				nodes = append(nodes, e)
			}
			g.checkOrder(nodes...)
			var vals []string
			for _, e := range v.Values {
				n := g.fresh()
				g.line("let %s = %s in", n, g.expr(e))
				vals = append(vals, n)
			}
			if len(vals) != 0 && len(vals) != len(v.Names) {
				g.fail(s, "multiple-return declarations are unsupported")
				return
			}
			for i, id := range v.Names {
				if id.Name == "_" {
					continue
				}
				b := g.declare(id, false)
				if b.kind == "array int" {
					g.fail(id, "local slice aliases/allocations are unsupported; use the slice parameter directly")
					return
				}
				value := "0"
				if b.kind == "bool" {
					value = "false"
				}
				if len(vals) > 0 {
					value = vals[i]
				}
				g.line("let ref %s = %s in", b.name, value)
			}
		}
	case *ast.ExprStmt:
		g.checkOrder(s.X)
		if _, ok := s.X.(*ast.CallExpr); !ok {
			g.fail(s, "only call expression statements are supported")
			return
		}
		g.line("let _ = %s in", g.expr(s.X))
	case *ast.ReturnStmt:
		if len(s.Results) == 0 {
			if g.returnKind != "unit" {
				g.fail(s, "bare returns are unsupported")
				return
			}
			g.line("raise %s;", g.returnException)
		} else if len(s.Results) == 1 {
			g.checkOrder(s.Results[0])
			g.line("raise (%s %s);", g.returnException, g.expr(s.Results[0]))
		} else {
			g.fail(s, "multiple return values are unsupported")
		}
	case *ast.IfStmt:
		g.push()
		g.line("(")
		g.indent++
		if s.Init != nil {
			g.statement(s.Init)
		}
		g.checkOrder(s.Cond)
		g.line("if %s then", g.expr(s.Cond))
		g.block(s.Body)
		if s.Else != nil {
			g.line("else (")
			g.indent++
			g.statement(s.Else)
			g.line("()")
			g.indent--
			g.line(")")
		} else {
			g.line("else ()")
		}
		g.indent--
		g.line(");")
		g.pop()
	case *ast.ForStmt:
		g.loop(s)
	case *ast.BranchStmt:
		if s.Label != nil || g.loopDepth == 0 {
			g.fail(s, "only unlabeled break/continue inside loops are supported")
			return
		}
		switch s.Tok {
		case token.BREAK:
			g.line("raise GoBreak;")
		case token.CONTINUE:
			g.line("raise GoContinue;")
		default:
			g.fail(s, "unsupported branch %s", s.Tok)
		}
	case *ast.BlockStmt:
		g.block(s)
		g.line(";")
	case *ast.EmptyStmt:
	default:
		g.fail(s, "unsupported statement %T", s)
	}
}

type location struct {
	name, index, read string
	discard           bool
}

func (l location) write(value string) string {
	if l.discard {
		return "let _ = " + value + " in ()"
	}
	if l.index != "" {
		return l.name + "[" + l.index + "] <- " + value
	}
	return l.name + " <- " + value
}
func (g *generator) location(e ast.Expr, define bool) location {
	if p, ok := e.(*ast.ParenExpr); ok {
		return g.location(p.X, define)
	}
	switch e := e.(type) {
	case *ast.Ident:
		if e.Name == "_" {
			return location{discard: true}
		}
		if define && g.info.Defs[e] != nil {
			return location{}
		}
		obj := g.info.Uses[e]
		n := g.names[obj]
		if b, ok := g.lookup(e.Name); ok && (b.param || b.kind == "array int") {
			g.fail(e, "rebinding parameters and slices is unsupported")
		}
		return location{name: n, read: n}
	case *ast.IndexExpr:
		id, ok := e.X.(*ast.Ident)
		if !ok {
			g.fail(e, "only indexing the slice parameter is supported as an assignment target")
			return location{}
		}
		n := g.expr(id)
		idx := g.fresh()
		g.line("let %s = %s in", idx, g.expr(e.Index))
		return location{name: n, index: idx, read: n + "[" + idx + "]"}
	default:
		g.fail(e, "unsupported assignment target %T", e)
		return location{}
	}
}
func (g *generator) assign(s *ast.AssignStmt) {
	var nodes []ast.Node
	for _, e := range s.Lhs {
		nodes = append(nodes, e)
	}
	for _, e := range s.Rhs {
		nodes = append(nodes, e)
	}
	g.checkOrder(nodes...)
	if len(s.Lhs) != len(s.Rhs) {
		g.fail(s, "multiple-return assignments are unsupported")
		return
	}
	var targets []location
	for _, e := range s.Lhs {
		targets = append(targets, g.location(e, s.Tok == token.DEFINE))
	}
	if s.Tok != token.DEFINE && s.Tok != token.ASSIGN {
		if len(targets) != 1 {
			g.fail(s, "invalid compound assignment")
			return
		}
		op := map[token.Token]string{token.ADD_ASSIGN: "+", token.SUB_ASSIGN: "-", token.MUL_ASSIGN: "*"}[s.Tok]
		if op == "" {
			g.fail(s, "unsupported compound assignment %s", s.Tok)
			return
		}
		old := g.fresh()
		g.line("let %s = %s in", old, targets[0].read)
		rhs := g.fresh()
		g.line("let %s = %s in", rhs, g.expr(s.Rhs[0]))
		g.line("%s;", targets[0].write(g.checked("("+old+" "+op+" "+rhs+")")))
		return
	}
	// Phase one: freeze all LHS indices and RHS values. Phase two: stores in
	// source order. In particular i, a[i] = i+1, x uses the original index.
	var rhs []string
	for _, e := range s.Rhs {
		n := g.fresh()
		g.line("let %s = %s in", n, g.expr(e))
		rhs = append(rhs, n)
	}
	for i, e := range s.Lhs {
		if id, ok := e.(*ast.Ident); ok && id.Name != "_" && s.Tok == token.DEFINE && g.info.Defs[id] != nil {
			b := g.declare(id, false)
			if b.kind == "array int" {
				g.fail(id, "local slice aliases/allocations are unsupported")
				return
			}
			g.line("let ref %s = %s in", b.name, rhs[i])
		} else {
			g.line("%s;", targets[i].write(rhs[i]))
		}
	}
}
func (g *generator) loop(s *ast.ForStmt) {
	g.push()
	g.line("(")
	g.indent++
	if s.Init != nil {
		g.statement(s.Init)
	}
	variant := false
	g.line("try")
	g.indent++
	cond := "true"
	if s.Cond != nil {
		g.checkOrder(s.Cond)
		cond = g.expr(s.Cond)
	}
	g.line("while %s do", cond)
	g.indent++
	// Bounds are representation invariants, maintained by checked arithmetic
	// and typed stores. They are proved at loop entry and on every iteration.
	var repr []string
	for _, scope := range g.scopes {
		for _, b := range scope {
			if b.kind == "int" {
				repr = append(repr, "in_int "+b.name)
			}
			if b.kind == "array int" {
				repr = append(repr, "int_array "+b.name)
			}
		}
	}
	sort.Strings(repr)
	for _, p := range repr {
		g.line("invariant { %s }", p)
	}
	for _, c := range g.specs[s] {
		switch c.kind {
		case "loop invariant":
			g.line("invariant { %s }", g.logic(c.expr, false, false, nil))
		case "loop variant":
			if variant {
				g.fail(s, "only one loop variant is supported")
			}
			variant = true
			g.line("variant { %s }", g.logic(c.expr, false, false, nil))
		}
	}
	if !variant {
		g.fail(s, "each loop needs a 'loop variant' annotation to prove termination")
	}
	g.line("try")
	g.indent++
	g.loopDepth++
	g.block(s.Body)
	g.loopDepth--
	g.indent--
	g.line("with GoContinue -> () end;")
	if s.Post != nil {
		g.statement(s.Post)
	}
	g.line("()")
	g.indent--
	g.line("done")
	g.indent--
	g.line("with GoBreak -> () end")
	g.indent--
	g.line(");")
	g.pop()
}
