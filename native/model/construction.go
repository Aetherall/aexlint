package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

func (m *Analyzer) fieldsIn(root, receiver *ast.Node, tag string) []string {
	fields := []string{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindPropertyAccessExpression && n.Name().Kind == ast.KindIdentifier && m.SameValue(n.Expression(), receiver) {
			name := n.Name().Text()
			if name != tag && !slices.Contains(fields, name) {
				fields = append(fields, name)
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(root)
	return fields
}

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

func (m *Analyzer) ConstructionFields(root, receiver *ast.Node, tag string) []string {
	fields := []string{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindObjectLiteralExpression || n.Kind == ast.KindArrayLiteralExpression {
			for _, field := range m.fieldsIn(n, receiver, tag) {
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

// constructs reports whether n builds an object from fields of receiver other than tag.
func (m *Analyzer) constructs(n, receiver *ast.Node, tag string) bool {
	return receiver != nil && m.ObjectResult(n) && len(m.ConstructionFields(n, receiver, tag)) > 0
}

// Built returns the objects an arm of d builds from fields of d's receiver other than the one d tests: its value, and
// what it returns, assigns or declares.
func (m *Analyzer) Built(d *Decision, b Branch) []*ast.Node {
	return m.Results(b.Body, d.Receiver, d.Field.Name)
}

func (m *Analyzer) Results(nodes []*ast.Node, receiver *ast.Node, tag string) []*ast.Node {
	results := []*ast.Node{}
	add := func(expression *ast.Node) {
		if expression != nil && m.constructs(expression, receiver, tag) {
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
