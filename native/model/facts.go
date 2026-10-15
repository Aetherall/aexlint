package model

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// testPosition reports whether read is where a test can use it: a condition, an operand of a comparison, or the
// argument of Boolean or includes.
func testPosition(read *ast.Node) bool {
	if Condition(read) {
		return true
	}
	child, p := parent(read)
	if p == nil {
		return false
	}
	switch p.Kind {
	case ast.KindBinaryExpression:
		comparison, _ := equality(p.AsBinaryExpression().OperatorToken.Kind)
		return comparison
	case ast.KindCallExpression:
		if argument, ok := booleanCall(p); ok {
			return argument == child
		}
		if _, ok := includes(p); ok {
			return p.Arguments()[0] == child
		}
	}
	return false
}

func logical(n *ast.Node) bool {
	if n == nil || n.Kind != ast.KindBinaryExpression {
		return false
	}
	op := n.AsBinaryExpression().OperatorToken.Kind
	return op == ast.KindAmpersandAmpersandToken || op == ast.KindBarBarToken
}

func negations(n *ast.Node) *ast.Node {
	for {
		q := outer(n).Parent
		if q == nil || q.Kind != ast.KindPrefixUnaryExpression || q.AsPrefixUnaryExpression().Operator != ast.KindExclamationToken {
			return n
		}
		n = q
	}
}

// widest returns n widened through negations to the && and || chains read as one test.
func (m *Analyzer) widest(n *ast.Node) *ast.Node {
	for {
		q := outer(n).Parent
		for q != nil && q.Kind == ast.KindPrefixUnaryExpression && q.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken {
			q = outer(q).Parent
		}
		if _, ok := m.TestOf(q); !logical(q) || !ok {
			return n
		}
		n = q
	}
}

// WholeTest returns the test read is part of: read with its negations when it is a condition, otherwise its comparison
// or its Boolean or includes call, widened to the && and || chains read as one test. It returns nil
// when read is not where a test can use it.
func (m *Analyzer) WholeTest(read *ast.Node) *ast.Node {
	if Condition(read) {
		return negations(read)
	}
	if !testPosition(read) {
		return nil
	}
	_, p := parent(read)
	return m.widest(p)
}

// About reports whether t asks about values a property declaration can hold: a test of literal values names at least
// one value of the declaration's type, so on a union of item types, `item.kind === "tool"` is about the member whose
// kind is "tool". Presence tests, and declarations whose type is not made of literal values, always qualify.
func (m *Analyzer) About(t Test, declaration *ast.Node) bool {
	if t.Claim.Presence() || declaration.Symbol() == nil {
		return true
	}
	tc := m.ctx.TypeChecker
	declared := tc.GetNonNullableType(tc.GetTypeOfSymbol(declaration.Symbol()))
	members := []*checker.Type{declared}
	if declared.Flags()&checker.TypeFlagsUnion != 0 {
		members = declared.Types()
	}
	named := strings.Split(t.Named, ",")
	for _, member := range members {
		if !literal(member) || slices.Contains(named, valueKey(member)) {
			return true
		}
	}
	return false
}

// testedReads lists the property reads of a program's project files that are where a test can use them, by property
// name. It is built once per program from syntax alone and never mutated afterwards.
type testedReads struct {
	byName map[string][]*ast.Node
}

var tested = NewShared(func(program *compiler.Program) *testedReads {
	_, partial := Collect(program, func(n *ast.Node) bool {
		return n.Kind == ast.KindPropertyAccessExpression && n.Name().Kind == ast.KindIdentifier && testPosition(n)
	})
	index := &testedReads{byName: map[string][]*ast.Node{}}
	for _, reads := range partial {
		for _, read := range reads {
			index.byName[read.Name().Text()] = append(index.byName[read.Name().Text()], read)
		}
	}
	return index
})

// TestedReads returns the reads of properties named name, across the program's project files, that are where a test
// can use them. Callers must not modify the result.
func (m *Analyzer) TestedReads(name string) []*ast.Node {
	return tested.For(m.ctx.Program).byName[name]
}
