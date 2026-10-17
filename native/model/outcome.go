package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Source is the data an arm's result depends on, relative to the value its decision tests.
type Source string

const (
	// NoResult is an arm that produces no value: it leaves, or only runs commands.
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

// Outcome is what an arm produces when it runs: the data its result depends on, and whether it runs commands. Values
// identify a constant result: the literals it picks, or its form.
type Outcome struct {
	Source Source
	// Commands is whether the arm runs something for what it does rather than for a value: an expression statement whose
	// value is discarded, such as a call, an update or an assignment to data other than a local of its function, or a
	// returned call typed void. It is not a purity analysis: calls in declarations and conditions are not commands.
	Commands bool
	Values   []string
}

// Fallback reports whether the arm does no work: it produces nothing new and runs no commands.
func (o Outcome) Fallback() bool {
	return !o.Commands && (o.Source == NoResult || o.Source == EmptyResult)
}

// Lookup reports whether the arm only picks a constant.
func (o Outcome) Lookup() bool { return !o.Commands && o.Source == ConstantResult }

// OutcomeOf reads what an arm produces. value is the value its decision tests, and tag the property of value it tests,
// or "" when it tests value itself.
func (m *Analyzer) OutcomeOf(body []*ast.Node, value Value, tag string) Outcome {
	if len(body) == 1 && body[0] != nil && ast.IsExpression(body[0]) {
		return m.result([]*ast.Node{body[0]}, value, tag, false)
	}
	statements := armStatements(body)
	return m.result(m.returned(statements), value, tag, m.commands(statements))
}

func armStatements(body []*ast.Node) []*ast.Node {
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
	return statements
}

// AssignsLocal reports whether an arm leaves a result in a local of its function declared outside it, rather than returning
// it: the function goes on to use what the arm produced.
func (m *Analyzer) AssignsLocal(body []*ast.Node) bool {
	statements := armStatements(body)
	found := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if found || ast.IsFunctionLike(n) || ast.IsClassLike(n) || n.Kind == ast.KindReturnStatement {
			return found
		}
		if m.kept(n, statements) != nil {
			found = true
			return true
		}
		n.ForEachChild(walk)
		return found
	}
	for _, statement := range statements {
		walk(statement)
	}
	return found
}

// returned lists the expressions return statements return, and the values assignments to a local of the arm's function
// declared outside the arm leave in it, outside the functions the statements declare.
func (m *Analyzer) returned(statements []*ast.Node) []*ast.Node {
	found := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindReturnStatement && n.Expression() != nil {
			found = append(found, n.Expression())
			return false
		}
		if value := m.kept(n, statements); value != nil {
			found = append(found, value)
			return false
		}
		n.ForEachChild(walk)
		return false
	}
	for _, statement := range statements {
		walk(statement)
	}
	return found
}

// kept returns the value an expression statement assigns to a local variable of its function declared outside
// statements, which they leave as their result, or nil.
func (m *Analyzer) kept(n *ast.Node, statements []*ast.Node) *ast.Node {
	if n.Kind != ast.KindExpressionStatement {
		return nil
	}
	e := Unwrap(n.Expression())
	if e.Kind != ast.KindBinaryExpression || e.AsBinaryExpression().OperatorToken.Kind != ast.KindEqualsToken || Unwrap(e.AsBinaryExpression().Left).Kind != ast.KindIdentifier {
		return nil
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(Unwrap(e.AsBinaryExpression().Left))
	if symbol == nil || symbol.ValueDeclaration == nil || symbol.ValueDeclaration.Kind != ast.KindVariableDeclaration {
		return nil
	}
	declaration := symbol.ValueDeclaration
	function := enclosingFunction(declaration)
	if function == nil || function != enclosingFunction(n) || slices.ContainsFunc(statements, func(s *ast.Node) bool { return encloses(s, declaration) }) {
		return nil
	}
	return e.AsBinaryExpression().Right
}

func enclosingFunction(n *ast.Node) *ast.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if ast.IsFunctionLike(p) {
			return p
		}
	}
	return nil
}

// commands reports whether statements run an expression statement, such as a call or an assignment, outside the
// functions they declare: what it does, rather than any value, is why it runs. Calls inside declarations and conditions
// compute values; whether they also have effects is not known. A call returning never leaves, like a throw, and an
// assignment leaving a result in a local produces it.
func (m *Analyzer) commands(statements []*ast.Node) bool {
	found := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if found || ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return found
		}
		if n.Kind == ast.KindExpressionStatement && !m.exits(n) && m.kept(n, statements) == nil {
			found = true
			return true
		}
		n.ForEachChild(walk)
		return found
	}
	for _, statement := range statements {
		walk(statement)
	}
	return found
}

// result reads what an arm produces from the expressions it can result in: nothing new when each is empty, the
// data they read together, or the constants they pick. A call returning void or undefined is a command, not a result.
func (m *Analyzer) result(expressions []*ast.Node, value Value, tag string, commands bool) Outcome {
	produced, emptied := []*ast.Node{}, false
	for _, e := range expressions {
		switch u := Unwrap(e); {
		case empty(e):
			emptied = true
		case (u.Kind == ast.KindCallExpression || u.Kind == ast.KindNewExpression || u.Kind == ast.KindAwaitExpression) && m.void(u):
			commands = true
		default:
			produced = append(produced, e)
		}
	}
	if len(produced) == 0 {
		if emptied {
			return Outcome{Source: EmptyResult, Commands: commands}
		}
		return Outcome{Source: NoResult, Commands: commands}
	}
	whole, parts, other := false, false, false
	for _, e := range produced {
		reads := m.ReadsOf(e, value, tag)
		whole, parts, other = whole || reads.Whole, parts || len(reads.Fields) > 0, other || reads.Other
	}
	switch {
	case (whole || parts) && other:
		return Outcome{Source: MixedResult, Commands: commands}
	case whole:
		return Outcome{Source: ValueResult, Commands: commands}
	case parts:
		return Outcome{Source: PartsResult, Commands: commands}
	case other:
		return Outcome{Source: OtherResult, Commands: commands}
	}
	values := []string{}
	for _, e := range produced {
		if !literals(e, &values) {
			if form := m.Form(e, value); !slices.Contains(values, form) {
				values = append(values, form)
			}
		}
	}
	return Outcome{Source: ConstantResult, Commands: commands, Values: values}
}

func (m *Analyzer) void(n *ast.Node) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	return t != nil && t.Flags()&(checker.TypeFlagsVoid|checker.TypeFlagsUndefined) != 0
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
				v = m.through(Value{Node: n, Root: tc.GetShorthandAssignmentValueSymbol(p), Path: []string{}})
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

// VariantReads returns the fields of d's receiver that an arm reads and that only some members of the receiver's type
// have: content of a variant, not fields every variant shares. It is empty when the receiver's type is not a union.
func (m *Analyzer) VariantReads(d *Decision, b Branch) []string {
	if d.Receiver == nil || d.Tag == "" {
		return nil
	}
	tc := m.ctx.TypeChecker
	t := tc.GetNonNullableType(tc.GetTypeAtLocation(d.Receiver))
	if t.Flags()&checker.TypeFlagsUnion == 0 && t.Flags()&checker.TypeFlagsInstantiable != 0 {
		if constraint := checker.Checker_getBaseConstraintOfType(tc, t); constraint != nil {
			t = tc.GetNonNullableType(constraint)
		}
	}
	if t.Flags()&checker.TypeFlagsUnion == 0 {
		return nil
	}
	shared := func(field string) bool {
		return !slices.ContainsFunc(t.Types(), func(member *checker.Type) bool { return tc.GetPropertyOfType(member, field) == nil })
	}
	reads := []string{}
	for _, n := range armStatements(b.Body) {
		for _, field := range m.ReadsOf(n, d.Object, d.Tag).Fields {
			if !shared(field) && !slices.Contains(reads, field) {
				reads = append(reads, field)
			}
		}
	}
	return reads
}

// OnlyReturned reports whether every local an arm of d keeps is read exactly once after d, inside what its function
// returns: the function builds its result through the local and is then itself the projection.
func (m *Analyzer) OnlyReturned(d *Decision, b Branch) bool {
	statements := armStatements(b.Body)
	symbols := []*ast.Symbol{}
	var collect func(*ast.Node) bool
	collect = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) || n.Kind == ast.KindReturnStatement {
			return false
		}
		if m.kept(n, statements) != nil {
			left := Unwrap(Unwrap(n.Expression()).AsBinaryExpression().Left)
			if symbol := m.ctx.TypeChecker.GetSymbolAtLocation(left); symbol != nil && !slices.Contains(symbols, symbol) {
				symbols = append(symbols, symbol)
			}
			return false
		}
		n.ForEachChild(collect)
		return false
	}
	for _, statement := range statements {
		collect(statement)
	}
	function := enclosingFunction(d.Root)
	if len(symbols) == 0 || function == nil || function.Body() == nil {
		return false
	}
	return !slices.ContainsFunc(symbols, func(symbol *ast.Symbol) bool { return !m.returnedOnce(symbol, d.Root, function) })
}

func (m *Analyzer) returnedOnce(symbol *ast.Symbol, decision, function *ast.Node) bool {
	uses := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if n == decision {
			return false
		}
		if n.Kind == ast.KindIdentifier && n.Pos() >= decision.End() && m.SymbolOf(n) == symbol {
			uses = append(uses, n)
		}
		n.ForEachChild(walk)
		return false
	}
	walk(function.Body())
	return len(uses) == 1 && returnedBy(uses[0], function)
}

func returnedBy(n, function *ast.Node) bool {
	for p := n.Parent; p != nil && p != function; p = p.Parent {
		if p.Kind == ast.KindReturnStatement || p == function.Body() && ast.IsExpression(p) {
			return true
		}
		if ast.IsFunctionLike(p) {
			return false
		}
	}
	return false
}
