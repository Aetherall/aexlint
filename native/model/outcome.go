package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Source is the data an arm's result depends on, relative to the value its decision tests.
type Source string

const (
	// NoResult is an arm that produces no value: it leaves, or only has effects.
	NoResult Source = "none"
	// EmptyResult is nothing new: missing or empty values, a boolean, 0 or -1.
	EmptyResult Source = "empty"
	// ConstantResult depends on constants only.
	ConstantResult Source = "constant"
	// ValueResult depends on the tested value as a whole: it is the value, or the value is passed on.
	ValueResult Source = "value"
	// PartsResult depends on parts of the tested value other than the part the decision tests.
	PartsResult Source = "parts"
	// OtherResult depends on other data only.
	OtherResult Source = "other"
	// MixedResult depends on the tested value and on other data.
	MixedResult Source = "mixed"
)

// Outcome is what an arm produces when it runs: the data its result depends on, and whether it does more than produce
// it. Values identify a constant result: the literals it picks, or its form.
type Outcome struct {
	Source  Source
	Effects bool
	Values  []string
}

// Fallback reports whether the arm does no work: it produces nothing new and has no effects.
func (o Outcome) Fallback() bool {
	return !o.Effects && (o.Source == NoResult || o.Source == EmptyResult)
}

// Lookup reports whether the arm only picks a constant.
func (o Outcome) Lookup() bool { return !o.Effects && o.Source == ConstantResult }

// OutcomeOf reads what an arm produces. value is the value its decision tests, and tag the property of value it tests,
// or "" when it tests value itself.
func (m *Analyzer) OutcomeOf(body []*ast.Node, value Value, tag string) Outcome {
	if len(body) == 1 && body[0] != nil && ast.IsExpression(body[0]) {
		return m.result(body[0], value, tag, false)
	}
	statements := []*ast.Node{}
	for _, n := range body {
		switch {
		case n == nil:
		case n.Kind == ast.KindBlock:
			statements = append(statements, n.Statements()...)
		default:
			statements = append(statements, n)
		}
	}
	for len(statements) > 0 && statements[len(statements)-1].Kind == ast.KindBreakStatement {
		statements = statements[:len(statements)-1]
	}
	if len(statements) == 0 {
		return Outcome{Source: NoResult}
	}
	last := statements[len(statements)-1]
	effects := len(statements) > 1
	switch {
	case last.Kind == ast.KindReturnStatement && last.Expression() != nil:
		return m.result(last.Expression(), value, tag, effects)
	case !effects && (last.Kind == ast.KindReturnStatement || last.Kind == ast.KindThrowStatement || last.Kind == ast.KindContinueStatement):
		return Outcome{Source: NoResult}
	}
	return Outcome{Source: NoResult, Effects: true}
}

func (m *Analyzer) result(e *ast.Node, value Value, tag string, effects bool) Outcome {
	if empty(e) {
		return Outcome{Source: EmptyResult, Effects: effects}
	}
	if u := Unwrap(e); u.Kind == ast.KindCallExpression || u.Kind == ast.KindNewExpression || u.Kind == ast.KindAwaitExpression {
		if t := m.ctx.TypeChecker.GetTypeAtLocation(u); t != nil && t.Flags()&(checker.TypeFlagsVoid|checker.TypeFlagsUndefined) != 0 {
			return Outcome{Source: NoResult, Effects: true}
		}
	}
	reads := m.ReadsOf(e, value, tag)
	whole, parts, other := reads.Whole, len(reads.Fields) > 0, reads.Other
	switch {
	case (whole || parts) && other:
		return Outcome{Source: MixedResult, Effects: effects}
	case whole:
		return Outcome{Source: ValueResult, Effects: effects}
	case parts:
		return Outcome{Source: PartsResult, Effects: effects}
	case other:
		return Outcome{Source: OtherResult, Effects: effects}
	}
	values := []string{}
	if !literals(e, &values) {
		values = []string{m.Form(e, value)}
	}
	return Outcome{Source: ConstantResult, Effects: effects, Values: values}
}

// empty reports whether n is nothing new: a trivial value, or an object or array literal holding only missing or empty
// values.
func empty(n *ast.Node) bool {
	if trivialValue(n) {
		return true
	}
	var missing func(*ast.Node) bool
	missing = func(n *ast.Node) bool {
		switch n = Unwrap(n); n.Kind {
		case ast.KindNullKeyword, ast.KindVoidExpression:
			return true
		case ast.KindIdentifier:
			return n.Text() == "undefined"
		case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
			return n.Text() == ""
		case ast.KindArrayLiteralExpression:
			return !slices.ContainsFunc(n.Elements(), func(e *ast.Node) bool { return !missing(e) })
		case ast.KindObjectLiteralExpression:
			return !slices.ContainsFunc(n.Properties(), func(p *ast.Node) bool {
				return p.Kind != ast.KindPropertyAssignment || !missing(p.Initializer())
			})
		}
		return false
	}
	switch Unwrap(n).Kind {
	case ast.KindArrayLiteralExpression, ast.KindObjectLiteralExpression:
		return missing(n)
	}
	return false
}

func literals(n *ast.Node, out *[]string) bool {
	n = Unwrap(n)
	switch n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral:
		if !slices.Contains(*out, n.Text()) {
			*out = append(*out, n.Text())
		}
		return true
	case ast.KindConditionalExpression:
		c := n.AsConditionalExpression()
		return literals(c.WhenTrue, out) && literals(c.WhenFalse, out)
	}
	return false
}

// Reads is the data evaluating an expression reads, relative to a value: the value as a whole, the fields of it other
// than the tested one, by name, and other data.
type Reads struct {
	Whole  bool
	Fields []string
	Other  bool
}

// ReadsOf reports which data evaluating e reads relative to value, ignoring tag, the property of value the decision
// tests. Functions written in e do not run with it. Functions, classes, namespaces, enums, callable values, helper
// objects of functions and values declared only in declaration files are code, not data, and so is a function called by name or a method of this; constants, intrinsic values such as
// undefined, and bindings declared in e read nothing. A method called on value reads value as a whole, and one called
// on other data reads that data.
func (m *Analyzer) ReadsOf(e *ast.Node, value Value, tag string) Reads {
	tc := m.ctx.TypeChecker
	found := Reads{Fields: []string{}}
	inside := func(symbol *ast.Symbol) bool {
		return slices.ContainsFunc(symbol.Declarations, func(d *ast.Node) bool {
			return ast.GetSourceFileOfNode(d) == ast.GetSourceFileOfNode(e) && d.Pos() >= e.Pos() && d.End() <= e.End()
		})
	}
	read := func(n *ast.Node, v Value) {
		if (value.Root != nil || value.This != nil) && v.Within(value) {
			rest := v.Path[len(value.Path):]
			switch {
			case tag != "" && len(rest) > 0 && rest[0] == tag:
			case len(rest) == 0 || len(rest) == 1 && callee(n):
				found.Whole = true
			case !slices.Contains(found.Fields, rest[0]):
				found.Fields = append(found.Fields, rest[0])
			}
			return
		}
		if v.This != nil {
			found.Other = true
			return
		}
		symbol := v.Root
		if symbol != nil && symbol.Flags&ast.SymbolFlagsAlias != 0 {
			symbol = tc.GetAliasedSymbol(symbol)
		}
		if _, constant := m.Unit(n); constant {
			return
		}
		if symbol == nil || len(symbol.Declarations) == 0 || inside(symbol) || m.code(symbol) {
			return
		}
		found.Other = true
	}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsTypeNode(n) || declaresName(n) || n != e && (ast.IsFunctionLike(n) || ast.IsClassLike(n)) {
			return false
		}
		switch n.Kind {
		case ast.KindIdentifier, ast.KindThisKeyword, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
			if own(n) {
				return false
			}
			v := m.ValueOf(n)
			if p := n.Parent; n.Kind == ast.KindIdentifier && p != nil && p.Kind == ast.KindShorthandPropertyAssignment {
				v = Value{Node: n, Root: tc.GetShorthandAssignmentValueSymbol(p), Path: []string{}}
			}
			if v.Base == nil {
				read(n, v)
				return false
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(e)
	return found
}

// declaresName reports whether n is a name written in a declaration or a property name, not a read.
func declaresName(n *ast.Node) bool {
	p := n.Parent
	if n.Kind != ast.KindIdentifier || p == nil || p.Name() != n {
		return false
	}
	switch p.Kind {
	case ast.KindPropertyAccessExpression, ast.KindPropertyAssignment, ast.KindParameter, ast.KindVariableDeclaration, ast.KindBindingElement,
		ast.KindMethodDeclaration, ast.KindPropertyDeclaration, ast.KindGetAccessor, ast.KindSetAccessor, ast.KindFunctionExpression, ast.KindFunctionDeclaration,
		ast.KindClassExpression, ast.KindClassDeclaration:
		return true
	}
	return false
}

// callee reports whether n is the function a call or construction invokes.
func callee(n *ast.Node) bool {
	for n.Parent != nil && n.Parent.Kind == ast.KindParenthesizedExpression {
		n = n.Parent
	}
	p := n.Parent
	return p != nil && (p.Kind == ast.KindCallExpression || p.Kind == ast.KindNewExpression) && p.Expression() == n
}

// own reports whether n invokes code bound where it is written: a function called by name, or a method of this.
func own(n *ast.Node) bool {
	if !callee(n) {
		return false
	}
	n = Unwrap(n)
	return n.Kind == ast.KindIdentifier || n.Kind == ast.KindPropertyAccessExpression && Unwrap(n.Expression()).Kind == ast.KindThisKeyword
}

// code reports whether a symbol names code rather than data: a function, class, method, namespace or enum, a value
// whose type is callable or constructable, a constant initialized with an object literal of functions, such as a
// helper object, or a value declared only in declaration files, such as a global of the runtime. An object type alone
// cannot tell code from data: an instance of Promise or Map has only methods too.
func (m *Analyzer) code(symbol *ast.Symbol) bool {
	if symbol.Flags&(ast.SymbolFlagsFunction|ast.SymbolFlagsClass|ast.SymbolFlagsMethod|ast.SymbolFlagsEnum|ast.SymbolFlagsValueModule|ast.SymbolFlagsNamespaceModule) != 0 {
		return true
	}
	if t := m.ctx.TypeChecker.GetTypeOfSymbol(symbol); t != nil && m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(t) {
		return true
	}
	if declaration := symbol.ValueDeclaration; declaration != nil && declaration.Kind == ast.KindVariableDeclaration && ast.IsVarConst(declaration) {
		if literal := Unwrap(declaration.Initializer()); literal != nil && literal.Kind == ast.KindObjectLiteralExpression && len(literal.Properties()) > 0 &&
			!slices.ContainsFunc(literal.Properties(), func(p *ast.Node) bool {
				return p.Kind != ast.KindMethodDeclaration && (p.Kind != ast.KindPropertyAssignment || !ast.IsFunctionLike(Unwrap(p.Initializer())))
			}) {
			return true
		}
	}
	return len(symbol.Declarations) > 0 && !slices.ContainsFunc(symbol.Declarations, func(d *ast.Node) bool { return !ast.GetSourceFileOfNode(d).IsDeclarationFile })
}
