package model

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
)

// Test is a place where code learns a claim about a value: Claim holds where Node is true. Named is the vocabulary
// values a test of literal values written as a whole condition writes, sorted and joined.
type Test struct {
	Node  *ast.Node
	Value Value
	Claim Claim
	Named string
}

func booleanCall(n *ast.Node) (*ast.Node, bool) {
	if n = Unwrap(n); n.Kind == ast.KindCallExpression && n.Expression().Kind == ast.KindIdentifier && n.Expression().Text() == "Boolean" && len(n.Arguments()) == 1 {
		return n.Arguments()[0], true
	}
	return nil, false
}

// TestOf reads n as a test of literal values or of presence. A bare read is a presence test only where it is used as a
// condition or converted with Boolean.
func (m *Analyzer) TestOf(n *ast.Node) (Test, bool) {
	bare := Condition(n)
	if argument, ok := booleanCall(n); ok {
		n, bare = argument, true
	}
	if t, ok := m.literalTest(n); ok {
		return t, true
	}
	if p, ok := PresenceTest(n); ok && (bare || Unwrap(n) != p.Subject) {
		return Test{Node: n, Value: m.ValueOf(p.Subject), Claim: presenceClaim(p)}, true
	}
	return Test{}, false
}

func operand(n *ast.Node) bool {
	switch parent := n.Parent; {
	case parent == nil:
		return false
	case parent.Kind == ast.KindArrayLiteralExpression:
		return true
	case parent.Kind == ast.KindBinaryExpression:
		comparison, _ := equality(parent.AsBinaryExpression().OperatorToken.Kind)
		return comparison
	}
	return false
}

func (m *Analyzer) named(test *ast.Node, vocabulary *Vocabulary) string {
	found := []string{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if operand(n) {
			if key, ok := m.Unit(n); ok && slices.Contains(vocabulary.Values, key) {
				found = append(found, key)
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(test)
	slices.Sort(found)
	return strings.Join(slices.Compact(found), ",")
}

// Property returns the project property t tests as resolved where it is read, so on a narrowed receiver it is the
// narrowed member's, whose type is what the read can hold; it is nil when t tests something else. Field identity
// across code uses FieldOf instead.
func (m *Analyzer) Property(t Test) *ast.Symbol {
	n := t.Value.Node
	if n.Kind != ast.KindPropertyAccessExpression || n.Name().Kind != ast.KindIdentifier {
		return nil
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n.Name())
	if symbol == nil || symbol.Flags&(ast.SymbolFlagsProperty|ast.SymbolFlagsGetAccessor) == 0 || !ProjectSymbol(symbol) {
		return nil
	}
	return symbol
}

// Receiver returns the object whose field t tests.
func (t Test) Receiver() *ast.Node { return Unwrap(t.Value.Node.Expression()) }

// FieldTest reads n as a test of a project property that rules out a value the property's declared type can hold.
func (m *Analyzer) FieldTest(n *ast.Node) (Test, bool) {
	t, ok := m.TestOf(n)
	if !ok || m.Property(t) == nil || !m.DeclaredClaim(t).Splits() {
		return Test{}, false
	}
	return t, true
}

// DeclaredClaim restricts t's claim to what the declared type of its value can hold: the property's as resolved where it
// is read, or otherwise the variable's or the expression's.
func (m *Analyzer) DeclaredClaim(t Test) Claim {
	if !t.Claim.Presence() {
		return t.Claim
	}
	declared := m.Declared(t.Value.Node)
	if property := m.Property(t); property != nil {
		declared = m.ctx.TypeChecker.GetTypeOfSymbol(property)
	}
	if declared == nil {
		return t.Claim
	}
	return t.Claim.Under(m.HeldBy(declared))
}

// shared restricts the declared claims of two tests to the values both declared types can hold.
func (m *Analyzer) shared(a, b Test) (Claim, Claim) {
	x, y := m.DeclaredClaim(a), m.DeclaredClaim(b)
	both := within(x.Domain, y.Domain)
	return x.Under(both), y.Under(both)
}

// SameQuestion reports whether two field tests ask the same question: the same field, split the same way in either
// direction among the values both can hold, with the same values written.
func (m *Analyzer) SameQuestion(a, b Test) bool {
	x, y := m.shared(a, b)
	if !x.comparable(y) || a.Named != b.Named || !x.Splits() || !(x.Same(y) || x.Same(y.Not())) {
		return false
	}
	return m.FieldOf(a.Value.Node).Same(m.FieldOf(b.Value.Node))
}

// Opposite reports whether two tests asking the same question accept complementary values.
func (m *Analyzer) Opposite(a, b Test) bool {
	x, y := m.shared(a, b)
	return !x.Same(y)
}

// Relied reports whether a later read of t's value in the same function relies on the narrowing a presence test
// brings: its type there no longer holds a missing value t splits off.
func (m *Analyzer) Relied(t Test) bool {
	scope := t.Node.Parent
	for scope != nil && !ast.IsFunctionLike(scope) && scope.Kind != ast.KindSourceFile {
		scope = scope.Parent
	}
	if scope == nil || !t.Claim.Presence() {
		return false
	}
	claim := m.DeclaredClaim(t)
	split := claim.Accepted
	if slices.Contains(split, Present) {
		split = claim.Rejected()
	}
	found := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if found {
			return true
		}
		if (n.Kind == ast.KindPropertyAccessExpression || n.Kind == ast.KindIdentifier) && n.Pos() >= t.Node.End() && m.ValueOf(n).Same(t.Value) && !m.presenceRead(n) {
			if held, ok := m.Held(n); ok && len(minus(split, held)) > 0 {
				found = true
				return true
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(scope)
	return found
}

func (m *Analyzer) presenceRead(n *ast.Node) bool {
	if Condition(n) {
		return true
	}
	q := outer(n).Parent
	if q == nil {
		return false
	}
	if _, ok := booleanCall(q); ok {
		return true
	}
	p, ok := PresenceTest(q)
	return ok && p.Subject == n
}

func outer(n *ast.Node) *ast.Node {
	for n.Parent != nil && n.Parent.Kind == ast.KindParenthesizedExpression {
		n = n.Parent
	}
	return n
}

// retested reports whether n is read to be tested: compared, switched on, looked up with includes, or used as a
// condition.
func retested(n *ast.Node) bool {
	child, p := parent(n)
	switch p.Kind {
	case ast.KindSwitchStatement:
		return p.Expression() == child
	case ast.KindBinaryExpression:
		if comparison, _ := equality(p.AsBinaryExpression().OperatorToken.Kind); comparison {
			return true
		}
	case ast.KindCallExpression:
		if _, ok := includes(p); ok {
			return true
		}
	}
	return Condition(n)
}

// Boolean reports whether n's value is used only for its truthiness: as a condition, negated, or through && and || whose
// own value is used that way.
func Boolean(n *ast.Node) bool { return decides(n, true) }

// Condition reports whether n's value is used as a condition.
func Condition(n *ast.Node) bool { return decides(n, false) }

// decides reports whether n's value is used as a condition, or with only, used only for its truthiness. The left operand
// of && and || is a condition, since it decides whether the right one runs, but its value can be the result: it is used
// only for its truthiness when the whole is.
func decides(n *ast.Node, only bool) bool {
	child, p := parent(n)
	if p == nil {
		return false
	}
	switch p.Kind {
	case ast.KindIfStatement, ast.KindWhileStatement, ast.KindDoStatement:
		return p.Expression() == child
	case ast.KindForStatement:
		return p.AsForStatement().Condition == child
	case ast.KindConditionalExpression:
		return p.AsConditionalExpression().Condition == child
	case ast.KindPrefixUnaryExpression:
		return p.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken
	case ast.KindBinaryExpression:
		switch p.AsBinaryExpression().OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
			return !only && p.AsBinaryExpression().Left == child || decides(p, only)
		}
	}
	return false
}
