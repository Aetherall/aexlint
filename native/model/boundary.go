package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Boundary is where an argument passes into a parameter of the called function.
type Boundary struct {
	Call      *ast.Node
	Argument  *ast.Node
	Callee    *ast.Node
	Parameter *ast.Node
	signature *checker.Signature
	index     int
}

// BoundaryOf returns the boundary argument crosses at call, when the called function has a body in a project file.
func (m *Analyzer) BoundaryOf(call, argument *ast.Node) (Boundary, bool) {
	arguments := call.Arguments()
	index := slices.Index(arguments, argument)
	if index < 0 || slices.ContainsFunc(arguments[:index], func(a *ast.Node) bool { return a.Kind == ast.KindSpreadElement }) {
		return Boundary{}, false
	}
	signature := m.ctx.TypeChecker.GetResolvedSignature(call)
	if signature == nil {
		return Boundary{}, false
	}
	fn := checker.Signature_declaration(signature)
	if fn == nil || !ast.IsFunctionLike(fn) || fn.Body() == nil || !ProjectFile(ast.GetSourceFileOfNode(fn)) {
		return Boundary{}, false
	}
	parameters := fn.Parameters()
	if len(parameters) > 0 && ast.IsThisParameter(parameters[0]) {
		parameters = parameters[1:]
	}
	if index >= len(parameters) || parameters[index].AsParameterDeclaration().DotDotDotToken != nil {
		return Boundary{}, false
	}
	return Boundary{Call: call, Argument: argument, Callee: fn, Parameter: parameters[index], signature: signature, index: index}, true
}

// Keeps reports whether the parameter across b keeps what t establishes about the argument: the argument as written,
// without the narrowing t brings, is not assignable to the parameter, so passing it untested would not type-check. A
// default value accepts undefined as it arrives.
func (m *Analyzer) Keeps(b Boundary, t Test) bool {
	tc := m.ctx.TypeChecker
	parameters := b.signature.Parameters()
	untested := m.untested(b.Argument, t)
	if b.index >= len(parameters) || untested == nil {
		return false
	}
	if b.Parameter.Initializer() != nil {
		defined := slices.DeleteFunc(slices.Clone(members(untested)), func(member *checker.Type) bool { return member.Flags()&checker.TypeFlagsUndefined != 0 })
		if len(defined) == 0 {
			return false
		}
		untested = tc.GetUnionType(defined)
	}
	return !checker.Checker_isTypeAssignableTo(tc, untested, tc.GetTypeOfSymbol(parameters[b.index]))
}

func (m *Analyzer) untested(argument *ast.Node, t Test) *checker.Type {
	written := ast.SkipParentheses(argument)
	switch written.Kind {
	case ast.KindAsExpression, ast.KindTypeAssertionExpression, ast.KindNonNullExpression:
		return m.ctx.TypeChecker.GetTypeAtLocation(written)
	}
	value := m.ValueOf(argument)
	for c := t.Value.Node; ; c = Unwrap(c.Expression()) {
		if m.ValueOf(c).Same(value) {
			return m.ctx.TypeChecker.GetTypeAtLocation(c)
		}
		if c.Kind != ast.KindPropertyAccessExpression && c.Kind != ast.KindElementAccessExpression {
			return nil
		}
	}
}

// arrive returns the values the parameter across b receives when values are passed: a default value replaces undefined.
func (b Boundary) arrive(values []string) []string {
	if b.Parameter.Initializer() == nil || !slices.Contains(values, MissingUndefined) {
		return values
	}
	return append(minus(values, []string{MissingUndefined}), Present)
}

// Inward returns the value v, read from the argument, becomes in the called function.
func (m *Analyzer) Inward(b Boundary, v Value) (Value, bool) {
	argument := m.ValueOf(b.Argument)
	if !v.Within(argument) {
		return Value{}, false
	}
	return Value{Node: b.Parameter, Root: b.Parameter.Symbol(), Path: v.Path[len(argument.Path):]}, true
}

// Outward returns the value v, read from the parameter in the called function, is at the call.
func (m *Analyzer) Outward(b Boundary, v Value) (Value, bool) {
	if v.Root != b.Parameter.Symbol() || v.This != nil || v.Base != nil {
		return Value{}, false
	}
	argument := m.ValueOf(b.Argument)
	return Value{Node: b.Argument, Root: argument.Root, This: argument.This, Base: argument.Base, Path: append(slices.Clone(argument.Path), v.Path...)}, true
}

// Rechecks returns the entry guards with which the function across b rejects again, as they arrive, every value that
// c rejects about t's value. c is what holds after t where the call is made.
func (m *Analyzer) Rechecks(b Boundary, t Test, c Claim) []Guard {
	received, ok := m.Inward(b, t.Value)
	if !ok {
		return nil
	}
	rejected := b.arrive(c.Rejected())
	guards := []Guard{}
	for _, e := range m.Entries(b.Callee) {
		if e.Test.Value.Same(received) && e.Claim.comparable(c) && len(within(e.Claim.Accepted, rejected)) == 0 {
			guards = append(guards, e)
		}
	}
	return guards
}

// Guard is a test after which code does Work only where Claim holds, and otherwise leaves without using the tested
// variable.
type Guard struct {
	Test  Test
	Claim Claim
	Work  []*ast.Node
	// Whole reports whether the test is the guard's whole condition.
	Whole bool
	// Fallback identifies the constant the guard leaves with, or is "" when it leaves with anything else.
	Fallback string
	Shown    string
}

func (m *Analyzer) guard(n *ast.Node, exits, whole bool, exit *ast.Node, work []*ast.Node, fallback, shown string) (Guard, bool) {
	t, ok := m.TestOf(n)
	if !ok || !t.Value.Named() {
		return Guard{}, false
	}
	variable := Value{Node: t.Value.Node, Root: t.Value.Root, This: t.Value.This}
	if exit != nil && m.usesBeyond(exit, variable, t.Value, true) || !slices.ContainsFunc(work, func(w *ast.Node) bool { return m.usesBeyond(w, variable, t.Value, false) }) {
		return Guard{}, false
	}
	claim := t.Claim
	if exits {
		claim = claim.Not()
	}
	return Guard{Test: t, Claim: claim, Work: work, Whole: whole, Fallback: fallback, Shown: shown}, true
}

func constant(n *ast.Node) (string, string) {
	if fallback, ok := Constant(n); ok {
		return fallback, shown(n)
	}
	return "", ""
}

// falsy identifies the value a test evaluates to when it fails: false for a boolean, or the only missing value its
// type allows for a truthiness test.
func (m *Analyzer) falsy(n *ast.Node) string {
	if t := m.ctx.TypeChecker.GetTypeAtLocation(n); t != nil && t.Flags()&checker.TypeFlagsBooleanLike != 0 {
		return "false"
	}
	if p, ok := PresenceTest(n); ok && p.Truthy && !p.Missing {
		if held, ok := m.Holds(p.Subject); ok {
			if keys := missing(held); len(keys) == 1 && keys[0] != MissingFalsy {
				return keys[0]
			}
		}
	}
	return ""
}

// Guards returns the guards an expression makes: `a && b ? work : exit`, `a || b ? exit : work`, and `a && b && work`.
func (m *Analyzer) Guards(e *ast.Node) []Guard {
	guards := []Guard{}
	add := func(g Guard, ok bool) {
		if ok {
			guards = append(guards, g)
		}
	}
	switch e = Unwrap(e); {
	case e.Kind == ast.KindConditionalExpression:
		c := e.AsConditionalExpression()
		fallback, display := constant(c.WhenFalse)
		conjunction := parts(c.Condition, ast.KindAmpersandAmpersandToken)
		for _, p := range conjunction {
			add(m.guard(p, false, len(conjunction) == 1, c.WhenFalse, []*ast.Node{c.WhenTrue}, fallback, display))
		}
		fallback, display = constant(c.WhenTrue)
		disjunction := parts(c.Condition, ast.KindBarBarToken)
		for _, p := range disjunction {
			add(m.guard(p, true, len(disjunction) == 1, c.WhenTrue, []*ast.Node{c.WhenFalse}, fallback, display))
		}
	case e.Kind == ast.KindBinaryExpression && e.AsBinaryExpression().OperatorToken.Kind == ast.KindAmpersandAmpersandToken:
		chain := parts(e, ast.KindAmpersandAmpersandToken)
		condition := Condition(e)
		for i, p := range chain[:len(chain)-1] {
			work := chain[i+1:]
			if condition {
				work = slices.DeleteFunc(slices.Clone(work), func(w *ast.Node) bool {
					_, test := m.TestOf(w)
					return test
				})
			}
			fallback := m.falsy(p)
			add(m.guard(p, false, len(chain) == 2, nil, work, fallback, fallback))
		}
	}
	return guards
}

// Entries returns the guards fn makes on its parameters before its work, with the constant fn returns when it leaves.
func (m *Analyzer) Entries(fn *ast.Node) []Guard {
	body := fn.Body()
	if body == nil || len(fn.Parameters()) == 0 {
		return nil
	}
	wrapped := ast.GetFunctionFlags(fn)&(ast.FunctionFlagsAsync|ast.FunctionFlagsGenerator) != 0
	entries := []Guard{}
	add := func(g Guard, ok bool) {
		if !ok {
			return
		}
		if declaration := g.Test.Value.Root; declaration == nil || declaration.ValueDeclaration == nil || declaration.ValueDeclaration.Kind != ast.KindParameter || declaration.ValueDeclaration.Parent != fn {
			return
		}
		if wrapped {
			g.Fallback, g.Shown = "", ""
		}
		entries = append(entries, g)
	}
	if first := firstStatement(body); first != nil && first.Kind == ast.KindIfStatement {
		statements := body.Statements()
		s := first.AsIfStatement()
		switch {
		case s.ElseStatement != nil:
		case m.exits(s.ThenStatement):
			fallback, display := "", ""
			if value, ok := returnedValue(s.ThenStatement); ok {
				fallback, display = constant(value)
			}
			disjunction := parts(s.Expression, ast.KindBarBarToken)
			for _, p := range disjunction {
				add(m.guard(p, true, len(disjunction) == 1, s.ThenStatement, statements[1:], fallback, display))
			}
		case len(statements) == 1:
			conjunction := parts(s.Expression, ast.KindAmpersandAmpersandToken)
			for _, p := range conjunction {
				add(m.guard(p, false, len(conjunction) == 1, nil, []*ast.Node{s.ThenStatement}, MissingUndefined, MissingUndefined))
			}
		}
	}
	expression, returned := entryExpression(body)
	if expression == nil {
		return entries
	}
	for _, g := range m.Guards(expression) {
		if !returned && g.Fallback != "" {
			g.Fallback, g.Shown = MissingUndefined, MissingUndefined
		}
		add(g, true)
	}
	return entries
}

func firstStatement(body *ast.Node) *ast.Node {
	if body.Kind != ast.KindBlock || len(body.Statements()) == 0 {
		return nil
	}
	return body.Statements()[0]
}

// entryExpression returns the expression a function body starts with, and whether the function returns its value.
func entryExpression(body *ast.Node) (*ast.Node, bool) {
	if body.Kind != ast.KindBlock {
		return body, true
	}
	statements := body.Statements()
	switch {
	case len(statements) == 0:
	case statements[0].Kind == ast.KindReturnStatement:
		return statements[0].Expression(), true
	case statements[0].Kind == ast.KindExpressionStatement && len(statements) == 1:
		return statements[0].Expression(), false
	}
	return nil, false
}

func (m *Analyzer) parameterOf(fn, n *ast.Node) (*ast.Node, bool) {
	for c := Unwrap(n); c != nil; c = Unwrap(c.Expression()) {
		switch c.Kind {
		case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
			continue
		case ast.KindIdentifier:
			symbol := m.ctx.TypeChecker.GetSymbolAtLocation(c)
			if symbol == nil || symbol.ValueDeclaration == nil || symbol.ValueDeclaration.Kind != ast.KindParameter || symbol.ValueDeclaration.Parent != fn {
				return nil, false
			}
			return symbol.ValueDeclaration, true
		}
		return nil, false
	}
	return nil, false
}
