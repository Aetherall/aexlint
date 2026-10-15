package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Restriction is a necessary comparison in a predicate's body that rejects a value its declared target permits.
type Restriction struct {
	Node     *ast.Node
	Subject  *ast.Node
	Rejected string
}

// PredicateContract is what a written type predicate's body requires of its declared target.
type PredicateContract struct {
	Target       *checker.Type
	Restrictions []Restriction
}

func members(t *checker.Type) []*checker.Type {
	if t.Flags()&checker.TypeFlagsUnion != 0 {
		return t.Types()
	}
	return []*checker.Type{t}
}

// known reports whether t, and every part of a union or intersection, is neither any, unknown, never nor generic.
func known(t *checker.Type) bool {
	if t == nil || t.Flags()&(checker.TypeFlagsAnyOrUnknown|checker.TypeFlagsNever|checker.TypeFlagsInstantiable) != 0 {
		return false
	}
	if t.Flags()&(checker.TypeFlagsUnion|checker.TypeFlagsIntersection) != 0 {
		return !slices.ContainsFunc(t.Types(), func(part *checker.Type) bool { return !known(part) })
	}
	return true
}

// accessor reports whether a property symbol has a getter, setter or auto-accessor declaration, including one merged with
// a data property.
func accessor(symbol *ast.Symbol) bool {
	return symbol != nil && (symbol.Flags&(ast.SymbolFlagsGetAccessor|ast.SymbolFlagsSetAccessor) != 0 ||
		slices.ContainsFunc(symbol.Declarations, func(declaration *ast.Node) bool {
			return declaration.Kind == ast.KindGetAccessor || declaration.Kind == ast.KindSetAccessor || declaration.ModifierFlags()&ast.ModifierFlagsAccessor != 0
		}))
}

func (m *Analyzer) parameterRead(n *ast.Node, parameter *ast.Symbol) bool {
	n = ast.SkipParentheses(n)
	return n.Kind == ast.KindIdentifier && m.ctx.TypeChecker.GetSymbolAtLocation(n) == parameter
}

// pure reports whether evaluating n reads only data properties of known types, so it cannot change what later
// conjuncts read.
func (m *Analyzer) pure(n *ast.Node) bool {
	switch n = ast.SkipParentheses(n); n.Kind {
	case ast.KindIdentifier:
		return known(m.ctx.TypeChecker.GetTypeAtLocation(n))
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindNullKeyword:
		return true
	case ast.KindPropertyAccessExpression:
		return !accessor(m.ctx.TypeChecker.GetSymbolAtLocation(n.Name())) &&
			known(m.ctx.TypeChecker.GetTypeAtLocation(n)) && m.pure(n.Expression()) &&
			(n.QuestionDotToken() != nil || m.holds(m.ctx.TypeChecker.GetTypeAtLocation(n.Expression()))&(AbsentNull|AbsentUndefined) == 0)
	case ast.KindElementAccessExpression:
		key := ast.SkipParentheses(n.AsElementAccessExpression().ArgumentExpression)
		t := m.ctx.TypeChecker.GetTypeAtLocation(n.Expression())
		if !ast.IsStringLiteralLike(key) || !known(t) {
			return false
		}
		property := m.ctx.TypeChecker.GetPropertyOfType(m.ctx.TypeChecker.GetNonNullableType(t), key.Text())
		return property != nil && !accessor(property) && known(m.ctx.TypeChecker.GetTypeAtLocation(n)) && m.pure(n.Expression()) &&
			(n.QuestionDotToken() != nil || m.holds(t)&(AbsentNull|AbsentUndefined) == 0)
	case ast.KindPrefixUnaryExpression:
		unary := n.AsPrefixUnaryExpression()
		return unary.Operator == ast.KindExclamationToken && m.pure(unary.Operand)
	case ast.KindBinaryExpression:
		switch n.AsBinaryExpression().OperatorToken.Kind {
		case ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsEqualsToken, ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
			return m.pure(n.AsBinaryExpression().Left) && m.pure(n.AsBinaryExpression().Right)
		}
	}
	return false
}

// admits reports whether n is a presence test of the parameter that every value of target passes.
func (m *Analyzer) admits(n *ast.Node, parameter *ast.Symbol, target *checker.Type) bool {
	p, ok := PresenceTest(n)
	if !ok || p.Missing || !m.parameterRead(p.Subject, parameter) {
		return false
	}
	return m.holds(target)&p.Removes == 0
}

// domain returns the literal value keys of a property's type, and whether it is an open string.
func domain(t *checker.Type) (keys []string, open bool, ok bool) {
	if !known(t) {
		return nil, false, false
	}
	for _, member := range members(t) {
		switch {
		case member.Flags()&checker.TypeFlagsString != 0:
			open = true
		case literal(member):
			keys = append(keys, valueKey(member))
		case member.Flags()&(checker.TypeFlagsNull|checker.TypeFlagsUndefined) != 0:
		default:
			return nil, false, false
		}
	}
	slices.Sort(keys)
	return slices.Compact(keys), open, len(keys) > 0 || open
}

// restriction reads n as a strict comparison of the parameter, or one of its properties, with a literal constant, and
// returns the first value of target it rejects. It reports false when n is not such a comparison.
func (m *Analyzer) restriction(n *ast.Node, parameter *ast.Symbol, target *checker.Type) (Restriction, bool) {
	n = ast.SkipParentheses(n)
	if n.Kind != ast.KindBinaryExpression {
		return Restriction{}, false
	}
	operator := n.AsBinaryExpression().OperatorToken.Kind
	if operator != ast.KindEqualsEqualsEqualsToken && operator != ast.KindExclamationEqualsEqualsToken {
		return Restriction{}, false
	}
	constantNode := func(n *ast.Node) bool {
		switch n.Kind {
		case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral:
			return true
		case ast.KindIdentifier:
			_, unit := m.Unit(n)
			return unit && m.pure(n)
		}
		return false
	}
	subject, constant := ast.SkipParentheses(n.AsBinaryExpression().Left), ast.SkipParentheses(n.AsBinaryExpression().Right)
	if !constantNode(constant) {
		subject, constant = constant, subject
	}
	if !constantNode(constant) {
		return Restriction{}, false
	}
	property := ""
	if !m.parameterRead(subject, parameter) {
		if subject.Kind != ast.KindPropertyAccessExpression || subject.Name().Kind != ast.KindIdentifier || !m.parameterRead(subject.Expression(), parameter) {
			return Restriction{}, false
		}
		property = subject.Name().Text()
	}
	tc := m.ctx.TypeChecker
	if !m.pure(subject) || !checker.Checker_isTypeAssignableTo(tc, tc.GetTypeAtLocation(constant), tc.GetTypeAtLocation(subject)) {
		return Restriction{}, false
	}
	compared, ok := m.Unit(constant)
	if !ok {
		return Restriction{}, false
	}
	found := Restriction{Node: n, Subject: subject}
	parts := 0
	for _, member := range members(target) {
		part := member
		if property != "" {
			if member.Flags()&(checker.TypeFlagsNull|checker.TypeFlagsUndefined) != 0 {
				continue
			}
			symbol := tc.GetPropertyOfType(member, property)
			if symbol == nil || accessor(symbol) {
				continue
			}
			part = tc.GetTypeOfSymbol(symbol)
		}
		keys, open, supported := domain(part)
		if !supported {
			return Restriction{}, false
		}
		parts++
		rejected := ""
		if operator == ast.KindExclamationEqualsEqualsToken {
			if slices.Contains(keys, compared) || open && tc.GetTypeAtLocation(constant).Flags()&checker.TypeFlagsStringLiteral != 0 {
				rejected = compared
			}
		} else {
			for _, key := range keys {
				if key != compared {
					rejected = key
					break
				}
			}
			if rejected == "" && open {
				rejected = "s:"
				if compared == "s:" {
					rejected = "s:other"
				}
			}
		}
		if found.Rejected == "" {
			found.Rejected = rejected
		}
	}
	return found, parts > 0
}

// PredicateContractOf reads a single-expression type predicate on an identifier parameter. Its restrictions are the
// conjuncts, preceded only by pure conjuncts, that reject a value of the declared target.
func (m *Analyzer) PredicateContractOf(fn *ast.Node) *PredicateContract {
	if !ast.IsFunctionLike(fn) || !declaresPredicate(fn) || len(fn.TypeParameters()) > 0 || ast.GetFunctionFlags(fn)&(ast.FunctionFlagsAsync|ast.FunctionFlagsGenerator) != 0 {
		return nil
	}
	expression := result(fn)
	signature := m.ctx.TypeChecker.GetSignatureFromDeclaration(fn)
	if expression == nil || signature == nil {
		return nil
	}
	predicate := m.ctx.TypeChecker.GetTypePredicateOfSignature(signature)
	if predicate == nil || checker.TypePredicate_kind(predicate) != checker.TypePredicateKindIdentifier {
		return nil
	}
	index := int(checker.TypePredicate_parameterIndex(predicate))
	parameters := fn.Parameters()
	if index < 0 || index >= len(parameters) || index >= len(signature.Parameters()) || slices.ContainsFunc(parameters, func(p *ast.Node) bool {
		return ast.IsThisParameter(p) || p.Name() == nil || p.Name().Kind != ast.KindIdentifier || p.Initializer() != nil
	}) || parameters[index].AsParameterDeclaration().DotDotDotToken != nil {
		return nil
	}
	input := m.ctx.TypeChecker.GetTypeOfSymbol(signature.Parameters()[index])
	target := checker.TypePredicate_t(predicate)
	if !known(input) || !known(target) || !checker.Checker_isTypeAssignableTo(m.ctx.TypeChecker, target, input) {
		return nil
	}
	parameter := m.ctx.TypeChecker.GetSymbolAtLocation(parameters[index].Name())
	if parameter == nil {
		return nil
	}
	contract := &PredicateContract{Target: target}
	for _, conjunct := range parts(expression, ast.KindAmpersandAmpersandToken) {
		if m.admits(conjunct, parameter, target) {
			continue
		}
		if r, ok := m.restriction(conjunct, parameter, target); ok && r.Rejected != "" {
			contract.Restrictions = append(contract.Restrictions, r)
		}
		if !m.pure(conjunct) {
			break
		}
	}
	return contract
}
