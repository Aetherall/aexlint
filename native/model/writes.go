package model

import (
	"github.com/microsoft/typescript-go/shim/ast"
)

// WriteKind is how a write changes the data at its target.
type WriteKind int

const (
	// Assignment replaces the target with Source, or with a part of what a destructuring assignment or a for-in or
	// for-of loop takes apart, when Source is nil.
	Assignment WriteKind = iota
	// Update replaces the target with a value computed from it: a compound assignment, whose Source is its right side,
	// or an increment or decrement.
	Update
	// Mutation calls a method on the target, which may change it.
	Mutation
)

// Write is a place where code changes data: Node is the assignment, update, loop or call, Target the expression it
// writes, and Source the expression assigned, when there is one.
type Write struct {
	Node   *ast.Node
	Target *ast.Node
	Source *ast.Node
	Kind   WriteKind
}

// Rebinds reports whether the write replaces its target rather than calling a method on it.
func (w Write) Rebinds() bool { return w.Kind != Mutation }

// Discarded reports whether nothing receives the write's own result: a mutation run as a command, or any assignment
// or update.
func (w Write) Discarded() bool { return w.Kind != Mutation || discarded(w.Node) }

// WritesAt lists the writes n itself makes, not those of the nodes inside it.
func WritesAt(n *ast.Node) []Write {
	switch {
	case ast.IsAssignmentExpression(n, false):
		binary := n.AsBinaryExpression()
		if binary.OperatorToken.Kind != ast.KindEqualsToken {
			return []Write{{Node: n, Target: binary.Left, Source: binary.Right, Kind: Update}}
		}
		if pattern := Unwrap(binary.Left); pattern.Kind == ast.KindArrayLiteralExpression || pattern.Kind == ast.KindObjectLiteralExpression {
			return destructured(n, pattern)
		}
		return []Write{{Node: n, Target: binary.Left, Source: binary.Right, Kind: Assignment}}
	case n.Kind == ast.KindPrefixUnaryExpression:
		if operator := n.AsPrefixUnaryExpression().Operator; operator == ast.KindPlusPlusToken || operator == ast.KindMinusMinusToken {
			return []Write{{Node: n, Target: n.AsPrefixUnaryExpression().Operand, Kind: Update}}
		}
	case n.Kind == ast.KindPostfixUnaryExpression:
		return []Write{{Node: n, Target: n.AsPostfixUnaryExpression().Operand, Kind: Update}}
	case n.Kind == ast.KindForInStatement || n.Kind == ast.KindForOfStatement:
		if initializer := n.AsForInOrOfStatement().Initializer; initializer.Kind != ast.KindVariableDeclarationList {
			return destructured(n, initializer)
		}
	case n.Kind == ast.KindCallExpression:
		if callee := Unwrap(n.Expression()); callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression {
			return []Write{{Node: n, Target: callee.Expression(), Kind: Mutation}}
		}
	}
	return nil
}

// destructured lists the targets a destructuring pattern written as an expression assigns, as assignments of parts.
func destructured(node, pattern *ast.Node) []Write {
	writes := []Write{}
	var target func(*ast.Node)
	target = func(n *ast.Node) {
		n = Unwrap(n)
		switch n.Kind {
		case ast.KindArrayLiteralExpression, ast.KindObjectLiteralExpression:
			n.ForEachChild(func(c *ast.Node) bool {
				target(c)
				return false
			})
		case ast.KindPropertyAssignment:
			target(n.Initializer())
		case ast.KindShorthandPropertyAssignment:
			writes = append(writes, Write{Node: node, Target: n.Name(), Kind: Assignment})
		case ast.KindSpreadElement, ast.KindSpreadAssignment:
			target(n.Expression())
		case ast.KindBinaryExpression:
			if ast.IsAssignmentExpression(n, true) {
				target(n.AsBinaryExpression().Left)
			}
		case ast.KindOmittedExpression:
		default:
			writes = append(writes, Write{Node: node, Target: n, Kind: Assignment})
		}
	}
	target(pattern)
	return writes
}

// writesIn lists the writes nodes make, including inside the functions they declare when nested is true.
func writesIn(nodes []*ast.Node, nested bool) []Write {
	writes := []Write{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if !nested && (ast.IsFunctionLike(n) || ast.IsClassLike(n)) {
			return false
		}
		writes = append(writes, WritesAt(n)...)
		n.ForEachChild(walk)
		return false
	}
	for _, n := range nodes {
		if n != nil {
			walk(n)
		}
	}
	return writes
}

// WriteValue reads the data a write changes as a value, or false when its target is not a value code can name again.
func (m *Analyzer) WriteValue(w Write) (Value, bool) {
	v := m.ValueOf(w.Target)
	return v, v.Named() && (v.Root != nil || v.This != nil)
}
