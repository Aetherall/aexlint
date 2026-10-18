package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

func (m *Analyzer) ObjectResult(n *ast.Node) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil {
		return false
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	if t.Flags()&checker.TypeFlagsObject != 0 {
		return !m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(t)
	}
	if t.Flags()&checker.TypeFlagsUnion != 0 {
		for _, member := range t.Types() {
			if member.Flags()&checker.TypeFlagsObject != 0 && !m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(member) {
				return true
			}
		}
	}
	return false
}

// ConstructionFields returns the fields of object other than tag that the object and array literals in root read.
func (m *Analyzer) ConstructionFields(root *ast.Node, object Value, tag string) []string {
	fields := []string{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindObjectLiteralExpression || n.Kind == ast.KindArrayLiteralExpression {
			for _, field := range m.ReadsOf(n, object, tag).Fields {
				if !slices.Contains(fields, field) {
					fields = append(fields, field)
				}
			}
			return false
		}
		n.ForEachChild(walk)
		return false
	}
	walk(root)
	return fields
}

// constructs reports whether n builds an object from fields of object other than tag.
func (m *Analyzer) constructs(n *ast.Node, object Value, tag string) bool {
	return object.root() && m.ObjectResult(n) && len(m.ConstructionFields(n, object, tag)) > 0
}

// Built returns the objects an arm of d builds from fields of d's receiver other than the one d tests: its value, and
// what it returns, assigns or declares.
func (m *Analyzer) Built(d *Decision, b Branch) []*ast.Node {
	return m.Results(b.Body, d.Object, d.Tag)
}

func (m *Analyzer) Results(nodes []*ast.Node, object Value, tag string) []*ast.Node {
	results := []*ast.Node{}
	add := func(expression *ast.Node) {
		if expression != nil && m.constructs(expression, object, tag) {
			results = append(results, expression)
		}
	}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindReturnStatement {
			add(n.Expression())
			return false
		}
		if ast.IsAssignmentExpression(n, true) {
			add(n.AsBinaryExpression().Right)
			return false
		}
		if n.Kind == ast.KindVariableDeclaration {
			add(n.Initializer())
			return false
		}
		n.ForEachChild(walk)
		return false
	}
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if ast.IsExpression(n) {
			add(n)
		} else {
			walk(n)
		}
	}
	return results
}

// Setting is whether a returned object literal sets an optional property of its function's result.
type Setting int

const (
	Unset Setting = iota
	Set
	// MaybeSet is a value that may be undefined, or a spread that may carry the property.
	MaybeSet
)

// OptionalSettings returns the optional properties of a function's result, marked optional or typed with undefined, and, for each object literal the function
// returns, whether it sets each one. The result must be one project object type, not a union or intersection, and every
// returned expression must be an object literal, a conditional of them, or null or undefined, with at least two literals.
func (m *Analyzer) OptionalSettings(fn *ast.Node) ([]*ast.Symbol, [][]Setting, bool) {
	t := m.Result(fn)
	if fn.Body() == nil || t == nil || t.Flags()&checker.TypeFlagsObject == 0 || !m.Owned(t) || m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(t) {
		return nil, nil, false
	}
	optional := []*ast.Symbol{}
	for _, property := range m.ctx.TypeChecker.GetPropertiesOfType(t) {
		if property.Flags&ast.SymbolFlagsMethod == 0 && (property.Flags&ast.SymbolFlagsOptional != 0 || m.holds(m.ctx.TypeChecker.GetTypeOfSymbol(property))&AbsentUndefined != 0) {
			optional = append(optional, property)
		}
	}
	literals, ok := m.returnedLiterals(fn)
	if len(optional) < 2 || !ok || len(literals) < 2 {
		return nil, nil, false
	}
	settings := [][]Setting{}
	for _, literal := range literals {
		settings = append(settings, m.sets(literal, optional))
	}
	return optional, settings, true
}

// returnedLiterals returns the object literals a function returns, or false when it returns anything else but null or
// undefined.
func (m *Analyzer) returnedLiterals(fn *ast.Node) ([]*ast.Node, bool) {
	literals, ok := []*ast.Node{}, true
	var add func(*ast.Node)
	add = func(n *ast.Node) {
		switch n = Unwrap(n); {
		case n.Kind == ast.KindObjectLiteralExpression:
			literals = append(literals, n)
		case n.Kind == ast.KindConditionalExpression:
			add(n.AsConditionalExpression().WhenTrue)
			add(n.AsConditionalExpression().WhenFalse)
		case m.nothing(n):
		default:
			ok = false
		}
	}
	if body := fn.Body(); body.Kind != ast.KindBlock {
		add(body)
		return literals, ok
	}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindReturnStatement && n.Expression() != nil {
			add(n.Expression())
		}
		n.ForEachChild(walk)
		return false
	}
	fn.Body().ForEachChild(walk)
	return literals, ok
}

// sets reads which of the optional properties an object literal sets. A later member overrides an earlier one.
func (m *Analyzer) sets(literal *ast.Node, optional []*ast.Symbol) []Setting {
	found := make([]Setting, len(optional))
	for _, member := range literal.AsObjectLiteralExpression().Properties.Nodes {
		for i, property := range optional {
			switch {
			case member.Kind == ast.KindSpreadAssignment:
				spread := m.ctx.TypeChecker.GetTypeAtLocation(member.Expression())
				if spread == nil || spread.Flags()&(checker.TypeFlagsAny|checker.TypeFlagsUnknown) != 0 || m.ctx.TypeChecker.GetPropertyOfType(m.ctx.TypeChecker.GetNonNullableType(spread), property.Name) != nil {
					found[i] = MaybeSet
				}
			case member.Name() != nil && propertyKey(member.Name()) == property.Name:
				found[i] = m.settingOf(member)
			}
		}
	}
	return found
}

func (m *Analyzer) settingOf(member *ast.Node) Setting {
	var value *ast.Node
	switch member.Kind {
	case ast.KindPropertyAssignment:
		value = member.Initializer()
	case ast.KindShorthandPropertyAssignment:
		value = member.Name()
	default:
		return Set
	}
	t := m.ctx.TypeChecker.GetTypeAtLocation(value)
	if t == nil {
		return MaybeSet
	}
	if t.Flags()&(checker.TypeFlagsUndefined|checker.TypeFlagsVoid) != 0 {
		return Unset
	}
	if m.holds(t)&AbsentUndefined != 0 {
		return MaybeSet
	}
	return Set
}

// nothing reports whether n is typed null, undefined or void.
func (m *Analyzer) nothing(n *ast.Node) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	return t != nil && t.Flags()&(checker.TypeFlagsNull|checker.TypeFlagsUndefined|checker.TypeFlagsVoid) != 0
}
