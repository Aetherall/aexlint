package max_projection_spread

import (
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// DerivationIndex lists every candidate derivation in a program's project source files, by file and by syntactic
// form. It is built once per program from syntax alone and never mutated afterwards.
type DerivationIndex struct {
	byFile map[*ast.SourceFile][]*ast.Node
	byKey  map[string][]*ast.Node
}

var indexes = model.NewShared(buildDerivationIndex)

// IndexFor returns the program's shared index.
func IndexFor(program *compiler.Program) *DerivationIndex { return indexes.For(program) }

func literal(n *ast.Node) bool {
	switch model.Unwrap(n).Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindTrueKeyword,
		ast.KindFalseKeyword, ast.KindNullKeyword:
		return true
	}
	return false
}

// Chain is a derivation's value, read from its root identifier.
type Chain struct {
	Root *ast.Node
}

// ChainOf returns the chain of a derivation's value: an identifier followed by property accesses and method calls
// with literal or no arguments, including at least one call.
func ChainOf(value *ast.Node) (Chain, bool) {
	called := false
	n := model.Unwrap(value)
	for {
		switch n.Kind {
		case ast.KindCallExpression:
			for _, argument := range n.Arguments() {
				if !literal(argument) {
					return Chain{}, false
				}
			}
			if model.Unwrap(n.Expression()).Kind != ast.KindPropertyAccessExpression {
				return Chain{}, false
			}
			called = true
			n = model.Unwrap(n.Expression())
		case ast.KindPropertyAccessExpression:
			n = model.Unwrap(n.Expression())
		case ast.KindIdentifier:
			if !called {
				return Chain{}, false
			}
			return Chain{Root: n}, true
		default:
			return Chain{}, false
		}
	}
}

// KeyOf returns a property assignment's static key.
func KeyOf(property *ast.Node) (string, bool) {
	if property.Kind != ast.KindPropertyAssignment {
		return "", false
	}
	name := property.Name()
	switch name.Kind {
	case ast.KindIdentifier, ast.KindStringLiteral, ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return name.Text(), true
	}
	return "", false
}

// keyOf returns the index key of a candidate derivation: its property key and the syntactic form of its chain.
func keyOf(property *ast.Node) (string, bool) {
	key, ok := KeyOf(property)
	if !ok {
		return "", false
	}
	chain, ok := ChainOf(property.Initializer())
	if !ok {
		return "", false
	}
	return key + "=" + model.SyntacticForm(property.Initializer(), chain.Root), true
}

func buildDerivationIndex(program *compiler.Program) *DerivationIndex {
	index := &DerivationIndex{}
	files, partial := model.Collect(program, func(n *ast.Node) bool { _, ok := keyOf(n); return ok })
	index.byFile = map[*ast.SourceFile][]*ast.Node{}
	index.byKey = map[string][]*ast.Node{}
	for i, file := range files {
		index.byFile[file] = partial[i]
		for _, n := range partial[i] {
			key, _ := keyOf(n)
			index.byKey[key] = append(index.byKey[key], n)
		}
	}
	return index
}

// File returns the candidate derivations in file. Callers must not modify the result.
func (index *DerivationIndex) File(file *ast.SourceFile) []*ast.Node { return index.byFile[file] }

// Keyed returns the candidate derivations with the index key of property. Callers must not modify the result.
func (index *DerivationIndex) Keyed(property *ast.Node) []*ast.Node {
	key, _ := keyOf(property)
	return index.byKey[key]
}
