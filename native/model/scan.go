package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// Scan is a loop emitting values computed from state it carries from one pass to the next. Carried are the variables
// of its function, declared before the loop, that the loop rebinds outside its own iteration control (a for
// statement's initializer and incrementor, or the variable a for-of or for-in loop assigns), and reads other than where
// a statement writes them. Emits are the effects in the loop appending or storing into data declared outside it a value,
// or under a key, that reads a variable the loop rebinds, directly or through a constant of the pass initialized from
// one. Each emitted value depends on what earlier passes did, as the
// values a generator yields from its own state do. Other is the work the loop does besides the scan: its effects on
// other data declared outside it, and the names of the variables it rebinds but only reads where a statement writes
// them, which build up a separate result. Computed are the carried variables the emitted values or keys read, and
// Counters the carried variables that only count passes (see countsPasses).
type Scan struct {
	Loop     *ast.Node
	Carried  []*ast.Symbol
	Computed []*ast.Symbol
	Counters []*ast.Symbol
	Emits    []Effect
	Other    []*ast.Node
}

// ScansIn lists the loops of a function, outside the functions it declares, that are scans.
func (m *Analyzer) ScansIn(fn *ast.Node) []Scan {
	found := []Scan{}
	if fn.Body() == nil {
		return found
	}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if ast.IsIterationStatement(n, false) {
			if s, ok := m.scan(fn, n); ok {
				found = append(found, s)
			}
		}
		n.ForEachChild(walk)
		return false
	}
	fn.Body().ForEachChild(walk)
	return found
}

func (m *Analyzer) scan(fn, loop *ast.Node) (Scan, bool) {
	carried := m.carriedBy(fn, loop)
	if len(carried) == 0 {
		return Scan{}, false
	}
	derived := m.derivedIn(loop, carried)
	outside := func(v Value) bool {
		return v.This != nil || v.Root != nil && v.Root.ValueDeclaration != nil && !encloses(loop, v.Root.ValueDeclaration)
	}
	s := Scan{Loop: loop, Carried: carried}
	others := []Effect{}
	for _, e := range m.EffectsIn([]*ast.Node{loop}) {
		if !outside(e.Receiver) || slices.ContainsFunc(carried, func(c *ast.Symbol) bool { return e.Receiver.Root == c }) {
			continue
		}
		if (e.Kind == Store || e.Kind == Append) && (m.reads(e.Stored, derived) || m.reads(e.Key, derived)) {
			s.Emits = append(s.Emits, e)
		} else {
			others = append(others, e)
		}
	}
	for _, e := range others {
		if !slices.ContainsFunc(s.Emits, func(emit Effect) bool { return emit.Receiver.Same(e.Receiver) }) {
			s.Other = append(s.Other, e.Node)
		}
	}
	s.Carried = slices.DeleteFunc(slices.Clone(carried), func(symbol *ast.Symbol) bool {
		if m.onlyUpdated(symbol, loop) {
			s.Other = append(s.Other, symbol.ValueDeclaration.Name())
			return true
		}
		return false
	})
	read := []*ast.Symbol{}
	for _, e := range s.Emits {
		for _, n := range []*ast.Node{e.Stored, e.Key} {
			if n != nil {
				read = append(read, m.symbolsRead(n, derived)...)
			}
		}
	}
	for _, symbol := range s.Carried {
		if slices.Contains(read, symbol) {
			s.Computed = append(s.Computed, symbol)
		}
		if m.countsPasses(symbol, loop) {
			s.Counters = append(s.Counters, symbol)
		}
	}
	return s, len(s.Emits) > 0
}

// countsPasses reports whether symbol only counts loop's passes: loop writes it once, adding one (`++`, `+= 1`), in a
// statement of its body itself, where the write runs whenever the statement does, and no `continue` of loop comes
// before that statement.
func (m *Analyzer) countsPasses(symbol *ast.Symbol, loop *ast.Node) bool {
	writes := []Write{}
	for _, w := range writesIn([]*ast.Node{loop}, false) {
		if v, ok := m.WriteValue(w); ok && v.Root == symbol && len(v.Path) == 0 {
			writes = append(writes, w)
		}
	}
	if len(writes) != 1 || !addsOne(writes[0]) || conditional(writes[0].Node) {
		return false
	}
	statement := writes[0].Node
	for !ast.IsStatement(statement) {
		statement = statement.Parent
	}
	body := loop.Statement()
	if statement != body && (body.Kind != ast.KindBlock || !slices.Contains(body.Statements(), statement)) {
		return false
	}
	skipped := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindContinueStatement && n.Pos() < statement.Pos() && jumpTarget(n) == loop {
			skipped = true
		}
		n.ForEachChild(walk)
		return skipped
	}
	walk(body)
	return !skipped
}

// addsOne reports whether w increments its target by one: `++`, or `+= 1`.
func addsOne(w Write) bool {
	switch w.Node.Kind {
	case ast.KindPrefixUnaryExpression:
		return w.Node.AsPrefixUnaryExpression().Operator == ast.KindPlusPlusToken
	case ast.KindPostfixUnaryExpression:
		return w.Node.AsPostfixUnaryExpression().Operator == ast.KindPlusPlusToken
	case ast.KindBinaryExpression:
		source := Unwrap(w.Source)
		return w.Node.AsBinaryExpression().OperatorToken.Kind == ast.KindPlusEqualsToken && source.Kind == ast.KindNumericLiteral && source.Text() == "1"
	}
	return false
}

// onlyUpdated reports whether loop reads symbol only where a statement writes it: its value only builds up for code
// after the loop, as a separate result rather than state the loop goes by. An update whose value is used reads it.
func (m *Analyzer) onlyUpdated(symbol *ast.Symbol, loop *ast.Node) bool {
	targets := []*ast.Node{}
	for _, w := range writesIn([]*ast.Node{loop}, false) {
		if p := w.Node.Parent; p != nil && (p.Kind == ast.KindExpressionStatement || p.Kind == ast.KindForStatement) {
			targets = append(targets, Unwrap(w.Target))
		}
	}
	only := true
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if c.Kind == ast.KindIdentifier && c.Parent != nil && !slices.Contains(targets, c) && m.referenced(c) == symbol {
			only = false
		}
		c.ForEachChild(walk)
		return !only
	}
	loop.ForEachChild(walk)
	return only
}

// carriedBy returns the variables of fn declared before loop that loop rebinds outside its iteration control.
func (m *Analyzer) carriedBy(fn, loop *ast.Node) []*ast.Symbol {
	control := []*ast.Node{}
	switch loop.Kind {
	case ast.KindForStatement:
		statement := loop.AsForStatement()
		control = append(control, statement.Initializer, statement.Incrementor)
	case ast.KindForInStatement, ast.KindForOfStatement:
		control = append(control, loop.AsForInOrOfStatement().Initializer)
	}
	controlled := func(n *ast.Node) bool {
		return slices.ContainsFunc(control, func(c *ast.Node) bool { return c != nil && encloses(c, n) })
	}
	carried := []*ast.Symbol{}
	for _, w := range writesIn([]*ast.Node{loop}, false) {
		if !w.Rebinds() || w.Node == loop || controlled(w.Node) {
			continue
		}
		v, ok := m.WriteValue(w)
		if !ok || v.Root == nil || len(v.Path) > 0 {
			continue
		}
		d := v.Root.ValueDeclaration
		if d == nil || d.Kind != ast.KindVariableDeclaration || enclosingFunction(d) != fn || encloses(loop, d) {
			continue
		}
		if !slices.Contains(carried, v.Root) {
			carried = append(carried, v.Root)
		}
	}
	return carried
}

// derivedIn maps the carried variables, and the constants declared in loop whose initializer reads one of them,
// directly or through another such constant, to the carried variables they read.
func (m *Analyzer) derivedIn(loop *ast.Node, carried []*ast.Symbol) map[*ast.Symbol][]*ast.Symbol {
	derived := map[*ast.Symbol][]*ast.Symbol{}
	for _, c := range carried {
		derived[c] = []*ast.Symbol{c}
	}
	for grew := true; grew; {
		grew = false
		var walk func(*ast.Node) bool
		walk = func(n *ast.Node) bool {
			if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
				return false
			}
			if n.Kind == ast.KindVariableDeclaration && ast.IsVarConst(n) && n.Initializer() != nil && n.Name().Kind == ast.KindIdentifier {
				if symbol := n.Symbol(); symbol != nil && derived[symbol] == nil {
					if reads := m.symbolsRead(n.Initializer(), derived); len(reads) > 0 {
						derived[symbol] = reads
						grew = true
					}
				}
			}
			n.ForEachChild(walk)
			return false
		}
		loop.ForEachChild(walk)
	}
	return derived
}

// reads reports whether n, when there is one, reads a carried variable through the symbols derived maps.
func (m *Analyzer) reads(n *ast.Node, derived map[*ast.Symbol][]*ast.Symbol) bool {
	return n != nil && len(m.symbolsRead(n, derived)) > 0
}

// symbolsRead returns the carried variables n reads through the symbols derived maps, outside the functions it
// declares.
func (m *Analyzer) symbolsRead(n *ast.Node, derived map[*ast.Symbol][]*ast.Symbol) []*ast.Symbol {
	found := []*ast.Symbol{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindIdentifier && c.Parent != nil {
			if symbol := m.referenced(c); symbol != nil {
				for _, carried := range derived[symbol] {
					if !slices.Contains(found, carried) {
						found = append(found, carried)
					}
				}
			}
		}
		c.ForEachChild(walk)
		return false
	}
	walk(n)
	return found
}
