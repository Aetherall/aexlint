package model

import (
	"maps"
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// Left is data a decision's branches leave to the code after it: the writes its branches make to one root, a variable
// or parameter of the function, or this.
type Left struct {
	Value  Value
	Writes []Write
}

// LeftValues lists the data a decision leaves: an if or switch statement, or a conditional expression run as a
// statement. Its branches leave the data they rebind on ways that reach its end, when the statements after it read
// that data before assigning it again, and the data they rebind on ways that break out of a statement around it, when
// the statements after that one read it. Writes into one root are one entry, valued at that root unless they all write
// the same data. Writes in callbacks are not read. Nothing is left to statements that start with a return: that return
// builds the one value the branches decided.
func (m *Analyzer) LeftValues(n *ast.Node) []Left {
	function := enclosingFunction(n)
	if function == nil {
		return nil
	}
	var e ending
	switch {
	case n.Kind == ast.KindIfStatement:
		s := n.AsIfStatement()
		e = m.writeControl().of(s.ThenStatement).merge(m.writeControl().of(s.ElseStatement))
	case n.Kind == ast.KindSwitchStatement:
		e = m.writeControl().of(n)
	case n.Kind == ast.KindExpressionStatement && Unwrap(n.Expression()).Kind == ast.KindConditionalExpression:
		conditional := Unwrap(n.Expression()).AsConditionalExpression()
		e = reached(writesIn([]*ast.Node{conditional.WhenTrue, conditional.WhenFalse}, false))
	default:
		return nil
	}
	left := m.leftAfter(n, n, function, e.end, nil)
	targets := slices.SortedFunc(maps.Keys(e.breaks), func(a, b *ast.Node) int { return a.Pos() - b.Pos() })
	for _, target := range targets {
		left = m.leftAfter(n, target, function, e.breaks[target], left)
	}
	slices.SortStableFunc(left, func(a, b Left) int { return first(a.Writes) - first(b.Writes) })
	return left
}

func first(writes []Write) int {
	position := writes[0].Target.Pos()
	for _, w := range writes {
		position = min(position, w.Target.Pos())
	}
	return position
}

// DecisionCondition returns what a decision LeftValues reads branches on: the condition of an if statement or
// conditional expression statement, or the expression a switch tests.
func DecisionCondition(n *ast.Node) *ast.Node {
	if n.Kind == ast.KindExpressionStatement {
		return Unwrap(n.Expression()).AsConditionalExpression().Condition
	}
	return n.Expression()
}

// leftAfter adds to left the data written on way that the statements after statement read.
func (m *Analyzer) leftAfter(decision, statement, function *ast.Node, way *exit, left []Left) []Left {
	for statement.Parent != nil && statement.Parent.Kind == ast.KindLabeledStatement {
		statement = statement.Parent
	}
	if way == nil || statement.Parent == nil {
		return left
	}
	list := statementList(statement.Parent)
	i := slices.Index(list, statement)
	if i < 0 || i == len(list)-1 || list[i+1].Kind == ast.KindReturnStatement {
		return left
	}
	for _, w := range way.writes {
		v, ok := m.state(w, decision, function)
		if !ok || !m.live(list[i+1:], v) {
			continue
		}
		at := slices.IndexFunc(left, func(l Left) bool { return l.Value.Root == v.Root && l.Value.This == v.This })
		switch {
		case at < 0:
			left = append(left, Left{Value: v, Writes: []Write{w}})
		case !slices.ContainsFunc(left[at].Writes, func(o Write) bool { return o.Node == w.Node && o.Target == w.Target }):
			if !left[at].Value.Same(v) {
				left[at].Value = Value{Root: v.Root, This: v.This, Path: []string{}}
			}
			left[at].Writes = append(left[at].Writes, w)
		}
	}
	return left
}

// state returns the data a write in decision rebinds, when it is state of function that outlives decision: a variable
// or parameter of function declared outside decision, or this in it, read again by name. A method called on data is a
// command, not a result its branch decides, and a write repeated by a loop inside decision accumulates rather than
// decides.
func (m *Analyzer) state(w Write, decision, function *ast.Node) (Value, bool) {
	v, ok := m.WriteValue(w)
	if !ok || !w.Rebinds() || repeated(w.Node, decision) {
		return Value{}, false
	}
	if v.This != nil {
		return v, v.This == ast.GetThisContainer(decision, false, false)
	}
	declaration := v.Root.ValueDeclaration
	if declaration == nil || encloses(decision, declaration) || enclosingFunction(declaration) != function {
		return Value{}, false
	}
	switch declaration.Kind {
	case ast.KindVariableDeclaration, ast.KindBindingElement, ast.KindParameter:
		return v, true
	}
	return Value{}, false
}

// repeated reports whether a loop inside decision holds n.
func repeated(n, decision *ast.Node) bool {
	for p := n.Parent; p != nil && p != decision; p = p.Parent {
		if ast.IsIterationStatement(p, false) {
			return true
		}
	}
	return false
}

// live reports whether statements, run in order, read v on some way before every way assigns it again.
func (m *Analyzer) live(statements []*ast.Node, v Value) bool {
	reads, _ := m.liveIn(statements, v)
	return reads
}

// liveIn reports whether statements read v before assigning it again, and whether, without reading it, every way
// through them assigns it again or leaves the function.
func (m *Analyzer) liveIn(statements []*ast.Node, v Value) (bool, bool) {
	for _, s := range statements {
		reads, kills := m.liveAt(s, v)
		if reads {
			return true, false
		}
		if kills {
			return false, true
		}
	}
	return false, false
}

func (m *Analyzer) liveAt(n *ast.Node, v Value) (bool, bool) {
	switch n.Kind {
	case ast.KindBlock:
		return m.liveIn(n.Statements(), v)
	case ast.KindIfStatement:
		s := n.AsIfStatement()
		if m.readsValue(s.Expression, v) {
			return true, false
		}
		thenReads, thenKills := m.liveAt(s.ThenStatement, v)
		elseReads, elseKills := false, false
		if s.ElseStatement != nil {
			elseReads, elseKills = m.liveAt(s.ElseStatement, v)
		}
		return thenReads || elseReads, thenKills && elseKills
	case ast.KindReturnStatement, ast.KindThrowStatement:
		return m.readsValue(n, v), true
	case ast.KindExpressionStatement:
		for _, w := range WritesAt(Unwrap(n.Expression())) {
			if w.Kind == Assignment && w.Source != nil && v.Within(m.ValueOf(w.Target)) {
				return m.readsValue(w.Source, v), true
			}
		}
	}
	return m.readsValue(n, v), false
}

// readsValue reports whether n reads v, data within it, or data it is within, other than as the target an assignment
// replaces. Reads in callbacks count.
func (m *Analyzer) readsValue(n *ast.Node, v Value) bool {
	targets := map[*ast.Node]bool{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		for _, w := range WritesAt(c) {
			if w.Kind == Assignment {
				targets[Unwrap(w.Target)] = true
			}
		}
		if targets[c] {
			return false
		}
		if c.Kind == ast.KindIdentifier || c.Kind == ast.KindThisKeyword {
			read := c
			for p := read.Parent; p != nil && (p.Kind == ast.KindPropertyAccessExpression || p.Kind == ast.KindElementAccessExpression && ast.IsStringOrNumericLiteralLike(Unwrap(p.AsElementAccessExpression().ArgumentExpression))) && p.Expression() == read; p = read.Parent {
				read = p
			}
			if targets[read] {
				return false
			}
			value := m.ValueOf(read)
			return value.Named() && (value.Within(v) || v.Within(value))
		}
		return c.ForEachChild(walk)
	}
	return walk(n)
}
