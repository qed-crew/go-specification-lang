package whyml

// A deliberately small, strict parser for the GPMA subset supported by this
// backend. The general frontend currently has permissive fallback expressions;
// a verifier must never silently lose part of a contract.
import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"
	"unicode"
)

type formula struct {
	op   string
	args []*formula
	vars []string
}
type clause struct {
	kind string
	expr *formula
}
type specParser struct {
	tokens []string
	at     int
}

func lexSpec(s string) ([]string, error) {
	var out []string
	for i := 0; i < len(s); {
		if s[i] == ' ' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t' {
			i++
			continue
		}
		start := i
		if unicode.IsLetter(rune(s[i])) || s[i] == '_' || s[i] == '\\' {
			i++
			for i < len(s) && (unicode.IsLetter(rune(s[i])) || unicode.IsDigit(rune(s[i])) || s[i] == '_') {
				i++
			}
		} else if s[i] >= '0' && s[i] <= '9' {
			i++
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
		} else {
			matched := false
			for _, op := range []string{"<==>", "==>", "<=", ">=", "==", "!=", "&&", "||"} {
				if strings.HasPrefix(s[i:], op) {
					i += len(op)
					matched = true
					break
				}
			}
			if !matched {
				if !strings.ContainsRune("()+-*/%!,;:[]<>", rune(s[i])) {
					return nil, fmt.Errorf("unsupported specification character %q", s[i])
				}
				i++
			}
		}
		out = append(out, s[start:i])
	}
	return append(out, "<eof>"), nil
}
func (p *specParser) peek() string { return p.tokens[p.at] }
func (p *specParser) pop() string {
	s := p.peek()
	if s != "<eof>" {
		p.at++
	}
	return s
}
func (p *specParser) want(s string) error {
	if p.pop() != s {
		return fmt.Errorf("expected %q near %q", s, p.tokens[p.at-1])
	}
	return nil
}

var precedence = map[string]int{"<==>": 1, "==>": 2, "||": 3, "&&": 4, "==": 5, "!=": 5, "<": 5, "<=": 5, ">": 5, ">=": 5, "+": 6, "-": 6, "*": 7, "/": 7, "%": 7}

func (p *specParser) expression(min int) (*formula, error) {
	t := p.pop()
	var left *formula
	switch t {
	case "(":
		v, e := p.expression(1)
		if e != nil {
			return nil, e
		}
		if e = p.want(")"); e != nil {
			return nil, e
		}
		left = v
	case "!", "-", "+":
		v, e := p.expression(8)
		if e != nil {
			return nil, e
		}
		left = &formula{op: "unary" + t, args: []*formula{v}}
	case `\forall`, `\exists`:
		var vars []string
		for {
			n := p.pop()
			if !validIdent(n) {
				return nil, fmt.Errorf("expected quantified variable, got %q", n)
			}
			vars = append(vars, n)
			if p.peek() != "," {
				break
			}
			p.pop()
		}
		if e := p.want("integer"); e != nil {
			return nil, e
		}
		if e := p.want(";"); e != nil {
			return nil, e
		}
		v, e := p.expression(1)
		if e != nil {
			return nil, e
		}
		left = &formula{op: t, vars: vars, args: []*formula{v}}
	default:
		if !validIdent(t) && t != `\old` && t != `\result` && t != `\true` && t != `\false` && (len(t) == 0 || t[0] < '0' || t[0] > '9') {
			return nil, fmt.Errorf("expected expression, got %q", t)
		}
		left = &formula{op: t}
	}
	for {
		if p.peek() == "(" {
			if len(left.args) > 0 {
				return nil, fmt.Errorf("unsupported call target")
			}
			p.pop()
			var args []*formula
			if p.peek() != ")" {
				for {
					a, e := p.expression(1)
					if e != nil {
						return nil, e
					}
					args = append(args, a)
					if p.peek() != "," {
						break
					}
					p.pop()
				}
			}
			if e := p.want(")"); e != nil {
				return nil, e
			}
			left = &formula{op: "call:" + left.op, args: args}
			continue
		}
		if p.peek() == "[" {
			p.pop()
			idx, e := p.expression(1)
			if e != nil {
				return nil, e
			}
			if e = p.want("]"); e != nil {
				return nil, e
			}
			left = &formula{op: "index", args: []*formula{left, idx}}
			continue
		}
		op := p.peek()
		prec := precedence[op]
		if prec < min || prec == 0 {
			break
		}
		p.pop()
		next := prec + 1
		if op == "==>" {
			next = prec
		}
		right, e := p.expression(next)
		if e != nil {
			return nil, e
		}
		if prec == 5 && precedence[left.op] == 5 {
			return nil, fmt.Errorf("chained comparisons are unsupported; use &&")
		}
		left = &formula{op: op, args: []*formula{left, right}}
	}
	return left, nil
}
func validIdent(s string) bool {
	if len(s) == 0 || !(s[0] == '_' || s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z') {
		return false
	}
	for _, c := range s {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
func parseSpec(s string) ([]clause, error) {
	tokens, e := lexSpec(s)
	if e != nil {
		return nil, e
	}
	p := &specParser{tokens: tokens}
	var out []clause
	for p.peek() != "<eof>" {
		kind := p.pop()
		if kind == "loop" {
			kind += " " + p.pop()
		}
		switch kind {
		case "requires", "ensures", "assert", "loop invariant", "loop variant":
		default:
			return nil, fmt.Errorf("unsupported annotation %q", kind)
		}
		if p.at+1 < len(tokens) && tokens[p.at+1] == ":" {
			p.pop()
			p.pop()
		} // optional GPMA label
		f, e := p.expression(1)
		if e != nil {
			return nil, e
		}
		if e = p.want(";"); e != nil {
			return nil, e
		}
		out = append(out, clause{kind, f})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty specification")
	}
	return out, nil
}

func readSpecs(file *ast.File, fs *token.FileSet) (map[ast.Node][]clause, error) {
	var nodes []ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncDecl, ast.Stmt:
			nodes = append(nodes, n)
		}
		return true
	})
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Pos() == nodes[j].Pos() {
			return nodes[i].End() > nodes[j].End()
		}
		return nodes[i].Pos() < nodes[j].Pos()
	})
	out := map[ast.Node][]clause{}
	for _, group := range file.Comments {
		for _, c := range group.List {
			s := c.Text
			if strings.HasPrefix(s, "/*") {
				s = strings.TrimSuffix(s[2:], "*/")
			} else {
				s = strings.TrimPrefix(s, "//")
			}
			s = strings.TrimSpace(s)
			if !strings.HasPrefix(s, "@") {
				continue
			}
			clauses, e := parseSpec(strings.TrimSpace(s[1:]))
			if e != nil {
				return nil, fmt.Errorf("%s: %w", fs.Position(c.Pos()), e)
			}
			var node ast.Node
			for _, n := range nodes {
				if n.Pos() >= c.End() {
					node = n
					break
				}
			}
			if node == nil {
				return nil, fmt.Errorf("%s: annotation has no following statement", fs.Position(c.Pos()))
			}
			for _, cl := range clauses {
				ok := false
				switch cl.kind {
				case "requires", "ensures":
					_, ok = node.(*ast.FuncDecl)
				case "loop invariant", "loop variant":
					_, ok = node.(*ast.ForStmt)
				case "assert":
					_, ok = node.(ast.Stmt)
				}
				if !ok {
					return nil, fmt.Errorf("%s: misplaced %s annotation", fs.Position(c.Pos()), cl.kind)
				}
			}
			out[node] = append(out[node], clauses...)
		}
	}
	return out, nil
}
