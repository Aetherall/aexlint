package model

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
)

// destination is where a literal built in an arm goes: a callee, an argument of a call, a return, an assignment, a
// variable, or another expression.
func (m *Analyzer) destination(literal *ast.Node, boundary map[*ast.Node]bool, source Value) string {
	c := literal
	for p := c.Parent; p != nil && !boundary[c]; c, p = p, p.Parent {
		switch p.Kind {
		case ast.KindParenthesizedExpression, ast.KindAsExpression, ast.KindSatisfiesExpression, ast.KindConditionalExpression:
			continue
		case ast.KindCallExpression, ast.KindNewExpression:
			if p.Expression() == c {
				return "callee"
			}
			return fmt.Sprintf("argument %d of %s", slices.Index(p.Arguments(), c), m.Form(p.Expression(), source))
		case ast.KindReturnStatement, ast.KindArrowFunction:
			return "return"
		case ast.KindBinaryExpression:
			if ast.IsAssignmentExpression(p, true) {
				return "assignment to " + m.Form(p.AsBinaryExpression().Left, source)
			}
			return "expression"
		case ast.KindVariableDeclaration:
			return "variable"
		}
		return "expression"
	}
	return "value"
}

type Construction struct {
	Node      *ast.Node
	Signature string
}

// Constructions returns, for each value the dispatch's branches handle, the objects and arrays a branch builds from
// fields of the dispatch's receiver, by their form relative to the receiver, with where each one goes.
func (m *Analyzer) Constructions(d *Decision) map[string][]Construction {
	out := map[string][]Construction{}
	if d.Receiver == nil || slices.ContainsFunc(d.Arms, func(a Branch) bool { return !a.Complete }) {
		return out
	}
	source := m.ValueOf(d.Receiver)
	for _, arm := range d.Arms {
		boundary := map[*ast.Node]bool{}
		for _, n := range arm.Body {
			if n != nil {
				boundary[n] = true
			}
		}
		found := []Construction{}
		var walk func(*ast.Node) bool
		walk = func(n *ast.Node) bool {
			if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
				return false
			}
			if (n.Kind == ast.KindObjectLiteralExpression || n.Kind == ast.KindArrayLiteralExpression) && len(m.fieldsIn(n, d.Receiver, "")) > 0 {
				signature := m.destination(n, boundary, source) + ": " + m.Form(n, source)
				if !slices.ContainsFunc(found, func(c Construction) bool { return c.Signature == signature }) {
					found = append(found, Construction{Node: n, Signature: signature})
				}
				return false
			}
			n.ForEachChild(walk)
			return false
		}
		for _, n := range arm.Body {
			if n != nil {
				walk(n)
			}
		}
		if len(found) == 0 {
			continue
		}
		slices.SortFunc(found, func(a, b Construction) int { return strings.Compare(a.Signature, b.Signature) })
		for _, value := range arm.Claim.Accepted {
			out[value] = found
		}
	}
	return out
}

// Shape renders an object literal's keys and constant values, sorted, as a syntax-only key for finding candidate copies.
func Shape(n *ast.Node) string {
	parts := []string{}
	for _, property := range n.Properties() {
		switch property.Kind {
		case ast.KindPropertyAssignment:
			part := written(property.Name())
			switch value := Unwrap(property.Initializer()); value.Kind {
			case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindNullKeyword:
				part += "=" + written(value)
			}
			parts = append(parts, part)
		case ast.KindSpreadAssignment:
			parts = append(parts, "...")
		default:
			parts = append(parts, written(property.Name()))
		}
	}
	slices.Sort(parts)
	return "{" + strings.Join(parts, ",") + "}"
}
