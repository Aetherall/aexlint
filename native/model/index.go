package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// Index lists the roots of a program's project source files by file, by the literal values they mention and by the outcome text they lead to,
// whether their question is true or, for an early-exit guard, false.
// Roots comparing a value with anything but a literal token, such as a constant or an enum member, are listed in other. It is built once per program from syntax alone and never mutated afterwards.
type Index struct {
	byFile    map[*ast.SourceFile][]*ast.Node
	byKey     map[string][]*ast.Node
	byOutcome map[string][]*ast.Node
	other     []*ast.Node
}

var indexes = NewShared(buildIndex)

// IndexFor returns the program's shared index.
func IndexFor(program *compiler.Program) *Index { return indexes.For(program) }

func tokenKey(n *ast.Node) (string, bool) {
	switch n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return "s:" + n.Text(), true
	case ast.KindNumericLiteral:
		if value, ok := numberValue(n); ok {
			return NumberKey(value), true
		}
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindMinusToken && unary.Operand.Kind == ast.KindNumericLiteral {
			if value, ok := numberValue(unary.Operand); ok {
				return NumberKey(-value), true
			}
		}
	}
	return "", false
}

// keys returns the literal tokens a root mentions, and whether it compares a value with something other than a
// literal token, such as a constant or an enum member, whose value only a checker knows.
func keys(root *ast.Node) ([]string, bool) {
	scope := []*ast.Node{root}
	opaque := false
	literal := func(n *ast.Node) bool {
		_, ok := tokenKey(ast.SkipParentheses(n))
		return ok
	}
	if root.Kind == ast.KindCaseClause {
		scope = nil
		clauses := root.Parent.AsCaseBlock().Clauses.Nodes
		for _, clause := range clauses[indexOf(clauses, root):] {
			if clause.Kind == ast.KindDefaultClause {
				break
			}
			scope = append(scope, clause.Expression())
			opaque = opaque || !literal(clause.Expression())
			if len(clause.Statements()) > 0 {
				break
			}
		}
	}
	if root.Kind == ast.KindArrayLiteralExpression {
		opaque = slices.ContainsFunc(root.Elements(), func(e *ast.Node) bool { return !literal(e) })
	}
	found := map[string]bool{}
	result := []string{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if key, ok := tokenKey(n); ok {
			if !found[key] {
				found[key] = true
				result = append(result, key)
			}
			return false
		}
		switch n.Kind {
		case ast.KindBinaryExpression:
			if comparison, _ := equality(n.AsBinaryExpression().OperatorToken.Kind); comparison && !literal(n.AsBinaryExpression().Left) && !literal(n.AsBinaryExpression().Right) {
				opaque = true
			}
		case ast.KindCallExpression:
			if receiver, ok := membership(n); ok {
				receiver = ast.SkipParentheses(receiver)
				opaque = opaque || receiver.Kind != ast.KindArrayLiteralExpression || slices.ContainsFunc(receiver.Elements(), func(e *ast.Node) bool { return !literal(e) })
			}
		}
		n.ForEachChild(walk)
		return false
	}
	for _, n := range scope {
		walk(n)
	}
	return result, opaque
}

func buildIndex(program *compiler.Program) *Index {
	files, partial := Collect(program, Root)
	index := &Index{}
	index.byFile = map[*ast.SourceFile][]*ast.Node{}
	index.byKey = map[string][]*ast.Node{}
	index.byOutcome = map[string][]*ast.Node{}
	for i, file := range files {
		index.byFile[file] = partial[i]
		for _, root := range partial[i] {
			found, opaque := keys(root)
			if len(found) == 0 || opaque {
				index.other = append(index.other, root)
			}
			for _, key := range found {
				index.byKey[key] = append(index.byKey[key], root)
			}
			for _, outcome := range []string{OutcomeText(root), RestText(root)} {
				if outcome != "" {
					index.byOutcome[outcome] = append(index.byOutcome[outcome], root)
				}
			}
		}
	}
	return index
}

// File returns the roots of file. Callers must not modify the result.
func (index *Index) File(file *ast.SourceFile) []*ast.Node { return index.byFile[file] }

// Keyed returns the roots mentioning the literal value key. Callers must not modify the result.
func (index *Index) Keyed(key string) []*ast.Node { return index.byKey[key] }

// Outcome returns the roots leading to the outcome text, when their question is true or false. Callers must not modify the result.
func (index *Index) Outcome(outcome string) []*ast.Node { return index.byOutcome[outcome] }

// Other returns the roots mentioning no literal token. Callers must not modify the result.
func (index *Index) Other() []*ast.Node { return index.other }
