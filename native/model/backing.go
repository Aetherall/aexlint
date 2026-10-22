package model

import (
	"math"
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/jsnum"
)

// Backing is the check a non-null assertion of a container's entry relies on, which the checker cannot connect to it.
// Assertion asserts an element read or lookup of Container whose type the checker leaves possibly undefined or null
// while the type of the entries it holds is not, and is neither any, unknown nor never, so only the entry's absence is
// asserted away, and Check is a test that, where the assertion runs, holds Container larger than the position read, or
// holding the key looked up: a guard earlier in a statement list around the assertion, whose branch leaves when the
// test fails, which calls an assertion function declared `asserts condition` with the test, or which calls a project
// function guaranteeing it (see guaranteed), or the condition of a branch around it (see guarantees), through the
// variables, never rebound, holding a size or a test. A position is a literal index, a negative literal given to `at`,
// or the container's size minus a literal; a key is the same value as the one tested with `has`. The check may be
// outside a callback around the assertion that a global collection method calls with its elements, which runs during
// the call, once per element. Nothing between the check and the assertion, including later runs of a loop or such a
// callback around the assertion but not the check, removes entries from the container, changes it by a method whose
// body is unknown, or rebinds it. Guard tells whether the check is such a guard, which the assertion only follows,
// rather than the condition of a branch holding it.
type Backing struct {
	Assertion *ast.Node
	Check     *ast.Node
	Container Value
	Guard     bool
}

// guarantee is what a test holds about a container where it has a given truth: a size of at least minimum, or holding
// key.
type guarantee struct {
	node      *ast.Node
	container Value
	minimum   int
	key       *ast.Node
}

// BackingOf returns the check a non-null assertion relies on.
func (m *Analyzer) BackingOf(assertion *ast.Node) (Backing, bool) {
	if assertion.Kind != ast.KindNonNullExpression || !m.missable(assertion.Expression()) {
		return Backing{}, false
	}
	inner := Unwrap(assertion.Expression())
	lookup, ok := m.LookupOf(inner)
	if !ok || lookup.Key == nil || !lookup.Container.Named() || lookup.Container.Partial {
		return Backing{}, false
	}
	keyed := inner.Kind == ast.KindCallExpression && m.keyed(Unwrap(inner.Expression()).Expression()) ||
		inner.Kind == ast.KindElementAccessExpression && !m.array(inner) && !m.isInteger(lookup.Key)
	if entry := m.entryType(inner, keyed); entry == nil || m.missableType(entry) || entry.Flags()&(checker.TypeFlagsNever|checker.TypeFlagsAnyOrUnknown) != 0 {
		return Backing{}, false
	}
	need := 0
	if !keyed {
		if need = m.needed(inner, lookup); need == 0 {
			return Backing{}, false
		}
	}
	holds := func(g guarantee) bool {
		if !g.container.Same(lookup.Container) {
			return false
		}
		if keyed {
			return g.key != nil && m.SameValue(g.key, lookup.Key)
		}
		return g.key == nil && g.minimum >= need
	}
	for c := assertion; c.Parent != nil; c = c.Parent {
		p := c.Parent
		if ast.IsFunctionLike(p) && !m.synchronous(p) {
			break
		}
		var found []guarantee
		guard := false
		switch {
		case p.Kind == ast.KindIfStatement && c == p.AsIfStatement().ThenStatement:
			found = m.guarantees(p.Expression(), true)
		case p.Kind == ast.KindIfStatement && c == p.AsIfStatement().ElseStatement:
			found = m.guarantees(p.Expression(), false)
		case p.Kind == ast.KindConditionalExpression && c == p.AsConditionalExpression().WhenTrue:
			found = m.guarantees(p.AsConditionalExpression().Condition, true)
		case p.Kind == ast.KindConditionalExpression && c == p.AsConditionalExpression().WhenFalse:
			found = m.guarantees(p.AsConditionalExpression().Condition, false)
		case p.Kind == ast.KindBinaryExpression && c == p.AsBinaryExpression().Right:
			switch p.AsBinaryExpression().OperatorToken.Kind {
			case ast.KindAmpersandAmpersandToken:
				found = m.guarantees(p.AsBinaryExpression().Left, true)
			case ast.KindBarBarToken, ast.KindQuestionQuestionToken:
				found = m.guarantees(p.AsBinaryExpression().Left, false)
			}
		default:
			guard = true
			list := statementList(p)
			for i := indexOf(list, c) - 1; i >= 0; i-- {
				if guard := list[i]; guard.Kind == ast.KindIfStatement && guard.AsIfStatement().ElseStatement == nil && m.exits(guard.AsIfStatement().ThenStatement) {
					found = append(found, m.guarantees(guard.Expression(), false)...)
				} else if asserted := m.asserted(guard); asserted != nil {
					found = append(found, m.guarantees(asserted, true)...)
				} else {
					found = append(found, m.ensured(guard)...)
				}
			}
		}
		for _, g := range found {
			if holds(g) && m.intact(lookup.Container, g.node, assertion) {
				return Backing{Assertion: assertion, Check: g.node, Container: lookup.Container, Guard: guard}, true
			}
		}
	}
	return Backing{}, false
}

// asserted returns the condition an expression statement calling an assertion function, declared `asserts condition`,
// makes hold after it.
func (m *Analyzer) asserted(statement *ast.Node) *ast.Node {
	if statement.Kind != ast.KindExpressionStatement {
		return nil
	}
	call := Unwrap(statement.Expression())
	if call.Kind != ast.KindCallExpression {
		return nil
	}
	signature := m.ctx.TypeChecker.GetResolvedSignature(call)
	if signature == nil {
		return nil
	}
	predicate := m.ctx.TypeChecker.GetTypePredicateOfSignature(signature)
	if predicate == nil || checker.TypePredicate_kind(predicate) != checker.TypePredicateKindAssertsIdentifier || checker.TypePredicate_t(predicate) != nil {
		return nil
	}
	index := int(checker.TypePredicate_parameterIndex(predicate))
	if arguments := call.Arguments(); index >= 0 && index < len(arguments) {
		return arguments[index]
	}
	return nil
}

// ensured returns what an expression statement calling a project function guarantees about the containers it gives it
// (see guaranteed), with the call as the check.
func (m *Analyzer) ensured(statement *ast.Node) []guarantee {
	if statement.Kind != ast.KindExpressionStatement {
		return nil
	}
	call := Unwrap(statement.Expression())
	if call.Kind == ast.KindAwaitExpression {
		call = Unwrap(call.Expression())
	}
	fn := m.followed(call)
	if fn == nil || call.Kind != ast.KindCallExpression {
		return nil
	}
	var receiver *ast.Node
	if callee := Unwrap(call.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
		receiver = callee.Expression()
	}
	arguments := call.Arguments()
	given := func(a Anchor) (*ast.Node, bool) {
		var n *ast.Node
		switch {
		case a.Parameter < 0:
			n = receiver
		case a.Parameter < len(arguments) && !slices.ContainsFunc(arguments[:a.Parameter+1], func(x *ast.Node) bool { return x.Kind == ast.KindSpreadElement }):
			n = arguments[a.Parameter]
		}
		return n, n != nil && len(a.Path) == 0
	}
	found := []guarantee{}
	for _, g := range m.Summary(fn).Guarantees {
		container, ok := given(Anchor{Parameter: g.Container.Parameter})
		if !ok {
			continue
		}
		v := m.ValueOf(container)
		v.Node, v.Path = nil, append(slices.Clone(v.Path), g.Container.Path...)
		held := guarantee{node: call, container: v, minimum: g.Minimum}
		if g.Key != nil {
			if held.key, ok = given(*g.Key); !ok {
				continue
			}
		}
		found = append(found, held)
	}
	return found
}

// guaranteed returns what holds about the containers fn's caller gives it once fn returns: the checks of guards in its
// body's statement list whose branch only throws, of assertion functions it calls there, and what the project
// functions it calls there guarantee in turn, on a container read from
// a parameter or its receiver, with a key that is such data as a whole, when fn makes no effect on that container that
// removes entries or changes it by a method whose body is unknown.
func (m *Analyzer) guaranteed(fn *ast.Node, anchor func(Value) (Anchor, bool), effects []SummaryEffect) []SummaryGuarantee {
	body := fn.Body()
	if body == nil || body.Kind != ast.KindBlock {
		return nil
	}
	found := []SummaryGuarantee{}
	for _, statement := range statementList(body) {
		var held []guarantee
		if statement.Kind == ast.KindIfStatement && statement.AsIfStatement().ElseStatement == nil {
			e := m.flowControl().of(statement.AsIfStatement().ThenStatement)
			if e.end == nil && !e.returns && len(e.breaks) == 0 && len(e.continues) == 0 && e.throws {
				held = m.guarantees(statement.Expression(), false)
			}
		} else if asserted := m.asserted(statement); asserted != nil {
			held = m.guarantees(asserted, true)
		} else {
			held = m.ensured(statement)
		}
		for _, g := range held {
			target, ok := anchor(g.container)
			if !ok || slices.ContainsFunc(effects, func(e SummaryEffect) bool {
				return e.Target.Parameter == target.Parameter && (e.Kind == Remove || e.Kind == Change)
			}) {
				continue
			}
			summary := SummaryGuarantee{Container: target, Minimum: g.minimum}
			if g.key != nil {
				key, ok := anchor(m.ValueOf(g.key))
				if !ok || len(key.Path) > 0 {
					continue
				}
				summary.Key = &key
			}
			found = append(found, summary)
		}
	}
	return found
}

// runsDuring reports whether fn runs while the expression it is given to runs: a callback a global collection method
// calls during the call (see synchronous), the executor a Promise construction runs at once, or a callback another
// default library function calls during the call (see callingAtOnce).
func (m *Analyzer) runsDuring(fn *ast.Node) bool {
	return m.synchronous(fn) || m.executor(fn) || m.callingAtOnce(fn)
}

// atOnce names the default library functions, other than collection methods, calling a callback they are given during
// the call, by the global interface declaring them.
var atOnce = map[string][]string{
	"String":           {"replace", "replaceAll"},
	"JSON":             {"parse", "stringify"},
	"ArrayConstructor": {"from"},
}

// callingAtOnce reports whether fn is given to a default library function calling it during the call (see atOnce).
func (m *Analyzer) callingAtOnce(fn *ast.Node) bool {
	call := fn.Parent
	if call == nil || call.Kind != ast.KindCallExpression || !slices.Contains(call.Arguments(), fn) {
		return false
	}
	callee := Unwrap(call.Expression())
	if callee.Kind != ast.KindPropertyAccessExpression {
		return false
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(callee.Name())
	if symbol == nil || len(symbol.Declarations) == 0 {
		return false
	}
	for owner, names := range atOnce {
		if slices.Contains(names, symbol.Name) && slices.ContainsFunc(symbol.Declarations, func(d *ast.Node) bool { return m.ownedBy(d, owner) }) {
			return true
		}
	}
	return false
}

// executor reports whether fn is the executor a construction of the global Promise runs at once.
func (m *Analyzer) executor(fn *ast.Node) bool {
	call := fn.Parent
	return call != nil && call.Kind == ast.KindNewExpression && len(call.Arguments()) > 0 && call.Arguments()[0] == fn && m.typeIs(call, "Promise")
}

// synchronous reports whether fn is a callback a global collection method calls with its elements during the call.
func (m *Analyzer) synchronous(fn *ast.Node) bool {
	call := fn.Parent
	if call == nil || call.Kind != ast.KindCallExpression || !slices.Contains(call.Arguments(), fn) {
		return false
	}
	callee := Unwrap(call.Expression())
	if callee.Kind != ast.KindPropertyAccessExpression {
		return false
	}
	owners, name, ok := m.libraryMember(callee.Name())
	_, found := agreed(owners, name, elementCallbacks)
	return ok && found
}

// missable reports whether the checker types n as possibly undefined or null.
func (m *Analyzer) missable(n *ast.Node) bool {
	return m.missableType(m.ctx.TypeChecker.GetTypeAtLocation(n))
}

// entryType returns the type of the entries the container an element read or lookup reads holds: a map's value type
// argument, or the type of its elements by number.
func (m *Analyzer) entryType(n *ast.Node, keyed bool) *checker.Type {
	object := n.Expression()
	if n.Kind == ast.KindCallExpression {
		object = Unwrap(object).Expression()
	}
	tc := m.ctx.TypeChecker
	t := tc.GetNonNullableType(tc.GetTypeAtLocation(object))
	if keyed && n.Kind == ast.KindCallExpression {
		if arguments := tc.GetTypeArguments(t); len(arguments) == 2 {
			return arguments[1]
		}
		return nil
	}
	if keyed {
		key := tc.GetStringType()
		if tc.GetTypeAtLocation(n.AsElementAccessExpression().ArgumentExpression).Flags()&checker.TypeFlagsNumberLike != 0 {
			key = tc.GetNumberType()
		}
		return checker.Checker_getIndexTypeOfType(tc, t, key)
	}
	return checker.Checker_getIndexTypeOfType(tc, t, tc.GetNumberType())
}

// missableType reports whether t may hold undefined or null (see holds).
func (m *Analyzer) missableType(t *checker.Type) bool {
	return t != nil && m.holds(t)&(AbsentUndefined|AbsentNull) != 0
}

// needed returns the size an element read or `at` lookup needs its container to have for the entry to exist, or 0
// when its position is not known.
func (m *Analyzer) needed(n *ast.Node, lookup Lookup) int {
	key := Unwrap(lookup.Key)
	if value, ok := m.integer(key); ok {
		switch {
		case value >= 0:
			return value + 1
		case n.Kind == ast.KindCallExpression:
			return -value
		}
		return 0
	}
	if key.Kind == ast.KindBinaryExpression && key.AsBinaryExpression().OperatorToken.Kind == ast.KindMinusToken {
		size := Unwrap(key.AsBinaryExpression().Left)
		if value, ok := m.integer(key.AsBinaryExpression().Right); ok && value > 0 && m.sizeOf(size, lookup.Container) {
			return value
		}
	}
	return 0
}

// isInteger reports whether n's type is an integer literal.
func (m *Analyzer) isInteger(n *ast.Node) bool {
	_, ok := m.integer(Unwrap(n))
	return ok
}

// integer returns the integer n's type is a literal of.
func (m *Analyzer) integer(n *ast.Node) (int, bool) {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil || t.Flags()&checker.TypeFlagsNumberLiteral == 0 {
		return 0, false
	}
	value, ok := t.AsLiteralType().Value().(jsnum.Number)
	if !ok || float64(value) != math.Trunc(float64(value)) {
		return 0, false
	}
	return int(value), true
}

// sizeOf reports whether n is a size read of container.
func (m *Analyzer) sizeOf(n *ast.Node, container Value) bool {
	n = Unwrap(n)
	return n.Kind == ast.KindPropertyAccessExpression && m.size(n) && m.ValueOf(n.Expression()).Same(container)
}

// guarantees returns what a test holds about containers where it has the given truth: a truthy size read holds a size
// of at least one; a comparison of a size read with an integer literal the sizes its operator leaves; a global
// collection's has, an in test, Object.hasOwn or hasOwnProperty the key it is given; a negation the other truth; `&&` true, or `||` false, what both sides hold. A
// const of the function gives what its initializer gives, as a size read or as a test.
func (m *Analyzer) guarantees(n *ast.Node, truth bool) []guarantee {
	n = Unwrap(n)
	if initializer := m.initialized(n); initializer != nil {
		return m.guarantees(initializer, truth)
	}
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if n.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken {
			return m.guarantees(n.AsPrefixUnaryExpression().Operand, !truth)
		}
	case ast.KindPropertyAccessExpression:
		if truth && m.size(n) {
			return []guarantee{{node: n, container: m.ValueOf(n.Expression()), minimum: 1}}
		}
	case ast.KindCallExpression:
		callee := Unwrap(n.Expression())
		if !truth || callee.Kind != ast.KindPropertyAccessExpression || len(n.Arguments()) == 0 {
			return nil
		}
		if owners, name, ok := m.libraryMember(callee.Name()); ok && name == "has" && !slices.Contains(owners, "String") {
			return []guarantee{{node: n, container: m.ValueOf(callee.Expression()), key: n.Arguments()[0]}}
		}
		if symbol := m.ctx.TypeChecker.GetSymbolAtLocation(callee.Name()); symbol != nil && len(symbol.Declarations) > 0 {
			switch d := symbol.Declarations[0]; {
			case symbol.Name == "hasOwn" && m.ownedBy(d, "ObjectConstructor") && len(n.Arguments()) == 2:
				return []guarantee{{node: n, container: m.ValueOf(n.Arguments()[0]), key: n.Arguments()[1]}}
			case symbol.Name == "hasOwnProperty" && m.ownedBy(d, "Object"):
				return []guarantee{{node: n, container: m.ValueOf(callee.Expression()), key: n.Arguments()[0]}}
			}
		}
	case ast.KindBinaryExpression:
		b := n.AsBinaryExpression()
		operator := b.OperatorToken.Kind
		if operator == ast.KindInKeyword {
			if truth {
				return []guarantee{{node: n, container: m.ValueOf(b.Right), key: b.Left}}
			}
			return nil
		}
		switch {
		case operator == ast.KindAmpersandAmpersandToken && truth, operator == ast.KindBarBarToken && !truth:
			return append(m.guarantees(b.Left, truth), m.guarantees(b.Right, truth)...)
		case operator == ast.KindAmpersandAmpersandToken || operator == ast.KindBarBarToken:
			return nil
		}
		size, literal := m.sized(b.Left), b.Right
		if !m.isSize(size) {
			size, literal, operator = m.sized(b.Right), b.Left, flipped(operator)
		}
		value, ok := m.integer(literal)
		if !m.isSize(size) || !ok {
			return nil
		}
		if !truth {
			operator = negated(operator)
		}
		if minimum, ok := atLeast(operator, value); ok && minimum > 0 {
			return []guarantee{{node: n, container: m.ValueOf(size.Expression()), minimum: minimum}}
		}
	}
	return nil
}

// initialized returns the initializer of the variable of n's function that n reads, when it is a const or a variable
// never rebound.
func (m *Analyzer) initialized(n *ast.Node) *ast.Node {
	if n = Unwrap(n); n.Kind != ast.KindIdentifier {
		return nil
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
	if symbol == nil || symbol.ValueDeclaration == nil {
		return nil
	}
	d := symbol.ValueDeclaration
	if d.Kind != ast.KindVariableDeclaration || d.Initializer() == nil || d.Name().Kind != ast.KindIdentifier || enclosingFunction(d) != enclosingFunction(n) {
		return nil
	}
	if !ast.IsVarConst(d) && m.assigned(symbol) {
		return nil
	}
	return Unwrap(d.Initializer())
}

// sized returns n, or the initializer of the const n reads.
func (m *Analyzer) sized(n *ast.Node) *ast.Node {
	n = Unwrap(n)
	if initializer := m.initialized(n); initializer != nil {
		return initializer
	}
	return n
}

func (m *Analyzer) isSize(n *ast.Node) bool {
	return n.Kind == ast.KindPropertyAccessExpression && m.size(n)
}

// flipped returns the operator comparing the operands the other way around.
func flipped(operator ast.Kind) ast.Kind {
	switch operator {
	case ast.KindLessThanToken:
		return ast.KindGreaterThanToken
	case ast.KindLessThanEqualsToken:
		return ast.KindGreaterThanEqualsToken
	case ast.KindGreaterThanToken:
		return ast.KindLessThanToken
	case ast.KindGreaterThanEqualsToken:
		return ast.KindLessThanEqualsToken
	}
	return operator
}

// negated returns the operator holding where the comparison fails.
func negated(operator ast.Kind) ast.Kind {
	switch operator {
	case ast.KindLessThanToken:
		return ast.KindGreaterThanEqualsToken
	case ast.KindLessThanEqualsToken:
		return ast.KindGreaterThanToken
	case ast.KindGreaterThanToken:
		return ast.KindLessThanEqualsToken
	case ast.KindGreaterThanEqualsToken:
		return ast.KindLessThanToken
	case ast.KindEqualsEqualsEqualsToken:
		return ast.KindExclamationEqualsEqualsToken
	case ast.KindExclamationEqualsEqualsToken:
		return ast.KindEqualsEqualsEqualsToken
	case ast.KindEqualsEqualsToken:
		return ast.KindExclamationEqualsToken
	case ast.KindExclamationEqualsToken:
		return ast.KindEqualsEqualsToken
	}
	return ast.KindUnknown
}

// atLeast returns the least size a comparison `size operator value` leaves, when it leaves a least one.
func atLeast(operator ast.Kind, value int) (int, bool) {
	switch operator {
	case ast.KindGreaterThanToken:
		return value + 1, true
	case ast.KindGreaterThanEqualsToken:
		return value, true
	case ast.KindEqualsEqualsEqualsToken, ast.KindEqualsEqualsToken:
		return value, true
	case ast.KindExclamationEqualsEqualsToken, ast.KindExclamationEqualsToken:
		if value == 0 {
			return 1, true
		}
	}
	return 0, false
}

// intact reports whether nothing between a check and an assertion, including later iterations of a loop around the
// assertion but not the check, removes entries from container, changes it by a method whose body is unknown, or
// rebinds it.
func (m *Analyzer) intact(container Value, check, assertion *ast.Node) bool {
	fn := enclosingFunction(check)
	if fn == nil || fn.Body() == nil {
		return false
	}
	start, end := check.End(), assertion.Pos()
	bodies := []*ast.Node{fn.Body()}
	for p := assertion.Parent; p != nil && p != fn; p = p.Parent {
		if ast.IsFunctionLike(p) {
			bodies = append(bodies, p.Body())
		}
		if (ast.IsIterationStatement(p, false) || ast.IsFunctionLike(p)) && !encloses(p, check) {
			end = p.End()
		}
	}
	for _, e := range m.EffectsIn(bodies) {
		if e.Node.Pos() < start || e.Node.End() > end || !e.Receiver.Within(container) && !container.Within(e.Receiver) {
			continue
		}
		if e.Kind == Remove || e.Kind == Change && !m.reorders(e) {
			return false
		}
	}
	for _, w := range writesIn([]*ast.Node{fn.Body()}, true) {
		if w.Node.Pos() < start || w.Node.End() > end || !w.Rebinds() {
			continue
		}
		if v, ok := m.WriteValue(w); ok && container.Within(v) {
			return false
		}
	}
	return true
}

// reorders reports whether a change is a default library collection method that keeps the collection's size.
func (m *Analyzer) reorders(e Effect) bool {
	if e.Node.Kind != ast.KindCallExpression {
		return false
	}
	callee := Unwrap(e.Node.Expression())
	if callee.Kind != ast.KindPropertyAccessExpression {
		return false
	}
	owners, name, ok := m.libraryMember(callee.Name())
	effect, found := agreed(owners, name, libraryEffects)
	return ok && found && effect.kind == Change
}
