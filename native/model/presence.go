package model

import (
	"slices"
	"strconv"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/jsnum"
)

// Absence is a set of missing values: undefined, null, and other falsy values.
type Absence uint8

const (
	AbsentUndefined Absence = 1 << iota
	AbsentNull
	AbsentFalsy
	AbsentAny = AbsentUndefined | AbsentNull | AbsentFalsy
)

// Presence is a test of whether Subject holds a value. Missing is true when the test holds for a missing value.
type Presence struct {
	Subject *ast.Node
	Removes Absence
	Missing bool
	Truthy  bool
}

func missingLiteral(n *ast.Node) (Absence, bool) {
	switch n = Unwrap(n); n.Kind {
	case ast.KindNullKeyword:
		return AbsentNull, true
	case ast.KindVoidExpression:
		return AbsentUndefined, true
	case ast.KindIdentifier:
		return AbsentUndefined, n.Text() == "undefined"
	}
	return 0, false
}

func presenceSubject(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindIdentifier:
		return n.Text() != "undefined"
	case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
		return true
	}
	return false
}

// PresenceTest reads n as a truthiness test of a variable, property or element read, or as a comparison of any
// expression with null or undefined, possibly negated with !.
func PresenceTest(n *ast.Node) (Presence, bool) {
	negated := false
	for n = Unwrap(n); n.Kind == ast.KindPrefixUnaryExpression && n.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken; n = Unwrap(n.AsPrefixUnaryExpression().Operand) {
		negated = !negated
	}
	if presenceSubject(n) {
		return Presence{Subject: n, Removes: AbsentAny, Missing: negated, Truthy: true}, true
	}
	if n.Kind != ast.KindBinaryExpression {
		return Presence{}, false
	}
	b := n.AsBinaryExpression()
	comparison, unequal := equality(b.OperatorToken.Kind)
	if !comparison {
		return Presence{}, false
	}
	strict := b.OperatorToken.Kind == ast.KindEqualsEqualsEqualsToken || b.OperatorToken.Kind == ast.KindExclamationEqualsEqualsToken
	subject, other := Unwrap(b.Left), b.Right
	removes, ok := missingLiteral(other)
	if !ok {
		subject, other = Unwrap(b.Right), b.Left
		removes, ok = missingLiteral(other)
	}
	if _, absent := missingLiteral(subject); !ok || absent {
		return Presence{}, false
	}
	if !strict {
		removes = AbsentUndefined | AbsentNull
	}
	return Presence{Subject: subject, Removes: removes, Missing: unequal == negated}, true
}

func (m *Analyzer) holds(t *checker.Type) Absence {
	if t.Flags()&checker.TypeFlagsInstantiable != 0 {
		constraint := checker.Checker_getBaseConstraintOfType(m.ctx.TypeChecker, t)
		if constraint == nil {
			return AbsentAny
		}
		t = constraint
	}
	switch flags := t.Flags(); {
	case flags&(checker.TypeFlagsAny|checker.TypeFlagsUnknown) != 0:
		return AbsentAny
	case flags&(checker.TypeFlagsUndefined|checker.TypeFlagsVoid) != 0:
		return AbsentUndefined
	case flags&checker.TypeFlagsNull != 0:
		return AbsentNull
	case flags&checker.TypeFlagsUnion != 0:
		held := Absence(0)
		for _, member := range t.Types() {
			held |= m.holds(member)
		}
		return held
	}
	if m.truthy(t) {
		return 0
	}
	return AbsentFalsy
}

// truthy reports whether every value of t is truthy: a non-empty string, a number other than 0 and NaN, a bigint
// other than 0, true, a symbol, or an object type that admits no primitive and is not document.all. An intersection
// with primitive members holds their values; one of object types is an object type.
func (m *Analyzer) truthy(t *checker.Type) bool {
	if known, ok := m.truthiness[t]; ok {
		return known
	}
	result := false
	switch flags := t.Flags(); {
	case flags&checker.TypeFlagsStringLiteral != 0:
		value, ok := t.AsLiteralType().Value().(string)
		result = ok && value != ""
	case flags&checker.TypeFlagsNumberLiteral != 0:
		value, ok := t.AsLiteralType().Value().(jsnum.Number)
		result = ok && value != 0 && !value.IsNaN()
	case flags&checker.TypeFlagsBigIntLiteral != 0:
		value, ok := t.AsLiteralType().Value().(jsnum.PseudoBigInt)
		result = ok && value.Sign() != 0
	case flags&checker.TypeFlagsBooleanLiteral != 0:
		result = t.AsLiteralType().Value() == true
	case flags&checker.TypeFlagsESSymbolLike != 0:
		result = true
	case flags&checker.TypeFlagsIntersection != 0:
		primitives := slices.DeleteFunc(slices.Clone(t.Types()), func(member *checker.Type) bool {
			return member.Flags()&(checker.TypeFlagsObject|checker.TypeFlagsNonPrimitive) != 0
		})
		if len(primitives) > 0 {
			result = !slices.ContainsFunc(primitives, func(member *checker.Type) bool { return !m.truthy(member) })
		} else {
			result = m.admitsNoFalsy(t)
		}
	case flags&(checker.TypeFlagsObject|checker.TypeFlagsNonPrimitive) != 0:
		result = m.admitsNoFalsy(t)
	}
	m.truthiness[t] = result
	return result
}

// admitsNoFalsy reports whether an object type t admits no primitive and not document.all, and is not made only of
// brands, which can stand for primitives.
func (m *Analyzer) admitsNoFalsy(t *checker.Type) bool {
	tc := m.ctx.TypeChecker
	if !m.htmlAllResolved {
		m.htmlAllResolved = true
		if symbol := tc.ResolveName("HTMLAllCollection", nil, ast.SymbolFlagsType, false); symbol != nil {
			m.htmlAll = tc.GetDeclaredTypeOfSymbol(symbol)
		}
	}
	result := !m.phantom(t) && (m.htmlAll == nil || !tc.IsTypeAssignableTo(m.htmlAll, t))
	for _, primitive := range []*checker.Type{tc.GetStringType(), tc.GetNumberType(), tc.GetBooleanType(), tc.GetBigIntType()} {
		result = result && !tc.IsTypeAssignableTo(primitive, t)
	}
	return result
}

// present reports whether t holds a value other than undefined and null.
func (m *Analyzer) present(t *checker.Type) bool {
	if t.Flags()&checker.TypeFlagsUnion != 0 {
		return slices.ContainsFunc(t.Types(), m.present)
	}
	return t.Flags()&(checker.TypeFlagsUndefined|checker.TypeFlagsVoid|checker.TypeFlagsNull|checker.TypeFlagsNever) == 0
}

// Holds returns the missing values n can hold at its location, or false when its type is never.
func (m *Analyzer) Holds(n *ast.Node) (Absence, bool) {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil || t.Flags()&checker.TypeFlagsNever != 0 {
		return 0, false
	}
	return m.holds(t), true
}

// Constant identifies undefined, null, booleans, string and number literals, and empty array and object literals.
func Constant(n *ast.Node) (string, bool) {
	if n == nil {
		return "undefined", true
	}
	switch n = Unwrap(n); n.Kind {
	case ast.KindNullKeyword:
		return "null", true
	case ast.KindTrueKeyword:
		return "true", true
	case ast.KindFalseKeyword:
		return "false", true
	case ast.KindVoidExpression:
		return "undefined", true
	case ast.KindIdentifier:
		return "undefined", n.Text() == "undefined"
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return "s:" + n.Text(), true
	case ast.KindNumericLiteral:
		if value, err := strconv.ParseFloat(n.Text(), 64); err == nil {
			return NumberKey(value), true
		}
	case ast.KindArrayLiteralExpression:
		return "[]", len(n.Elements()) == 0
	case ast.KindObjectLiteralExpression:
		return "{}", len(n.Properties()) == 0
	}
	return "", false
}

func shown(n *ast.Node) string {
	if n == nil {
		return "undefined"
	}
	return Text(n, 40)
}

func returnedValue(n *ast.Node) (*ast.Node, bool) {
	if n.Kind == ast.KindBlock {
		if statements := n.Statements(); len(statements) == 1 {
			n = statements[0]
		}
	}
	if n.Kind != ast.KindReturnStatement {
		return nil, false
	}
	return n.Expression(), true
}

// WhenMissing evaluates test as if receiver, and every read through it, were missing. It reports false when the code does not determine the result.
func (m *Analyzer) WhenMissing(test *ast.Node, receiver *ast.Node) (bool, bool) {
	missing := m.ValueOf(receiver)
	var evaluate func(*ast.Node) (bool, bool)
	evaluate = func(n *ast.Node) (bool, bool) {
		n = Unwrap(n)
		if m.Reads(n, missing) {
			return false, true
		}
		switch n.Kind {
		case ast.KindPrefixUnaryExpression:
			if n.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken {
				value, ok := evaluate(n.AsPrefixUnaryExpression().Operand)
				return !value, ok
			}
		case ast.KindCallExpression:
			callee, arguments := Unwrap(n.Expression()), n.Arguments()
			if len(arguments) != 1 {
				return false, false
			}
			if callee.Kind == ast.KindIdentifier && callee.Text() == "Boolean" {
				return evaluate(arguments[0])
			}
			if array, ok := includes(n); ok && Unwrap(array).Kind == ast.KindArrayLiteralExpression && m.Reads(Unwrap(arguments[0]), missing) {
				for _, element := range Unwrap(array).Elements() {
					if _, absent := missingLiteral(element); absent {
						return false, false
					}
				}
				return false, true
			}
		case ast.KindBinaryExpression:
			b := n.AsBinaryExpression()
			switch b.OperatorToken.Kind {
			case ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
				left, ok := evaluate(b.Left)
				if !ok || left == (b.OperatorToken.Kind == ast.KindBarBarToken) {
					return left, ok
				}
				return evaluate(b.Right)
			}
			comparison, unequal := equality(b.OperatorToken.Kind)
			if !comparison {
				return false, false
			}
			other := b.Right
			switch {
			case m.Reads(Unwrap(b.Left), missing) && !m.Reads(Unwrap(b.Right), missing):
			case m.Reads(Unwrap(b.Right), missing) && !m.Reads(Unwrap(b.Left), missing):
				other = b.Left
			default:
				return false, false
			}
			equal := false
			if absent, ok := missingLiteral(other); ok {
				equal = absent == AbsentUndefined || b.OperatorToken.Kind == ast.KindEqualsEqualsToken || b.OperatorToken.Kind == ast.KindExclamationEqualsToken
			} else if _, ok := m.Unit(other); !ok {
				if _, ok := Constant(other); !ok {
					return false, false
				}
			}
			return equal != unequal, true
		}
		return false, false
	}
	return evaluate(test)
}
