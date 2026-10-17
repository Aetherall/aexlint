package model

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Origin names where a vocabulary read gets its values: a type alias or enum (rank 0), a property (rank 1), or a
// variable, parameter, or function (rank 2), or nothing known (rank 3).
type Origin struct {
	Label       string
	Declaration *ast.Node
	Rank        int
}

// Origins returns where the vocabulary n reads comes from. A property read through a union of object types has one
// origin per declaration.
func (m *Analyzer) Origins(n *ast.Node) []Origin {
	tc := m.ctx.TypeChecker
	n = Unwrap(n)
	if t := m.Declared(n); t != nil {
		for _, candidate := range []*checker.Type{t, tc.GetNonNullableType(t)} {
			if alias := candidate.Alias(); alias != nil {
				return []Origin{{alias.Symbol().Name, firstDeclaration(alias.Symbol()), 0}}
			}
			if symbol := candidate.Symbol(); symbol != nil && symbol.Flags&ast.SymbolFlagsEnum != 0 {
				return []Origin{{symbol.Name, firstDeclaration(symbol), 0}}
			}
		}
	}
	switch n.Kind {
	case ast.KindPropertyAccessExpression:
		if symbol := tc.GetSymbolAtLocation(n.Name()); symbol != nil {
			if symbol.Parent != nil && !strings.HasPrefix(symbol.Parent.Name, ast.InternalSymbolNamePrefix) {
				return []Origin{{symbol.Parent.Name + "." + symbol.Name, symbol.ValueDeclaration, 1}}
			}
			result := []Origin{}
			for _, declaration := range symbol.Declarations {
				label := symbol.Name
				if container := container(declaration); container != "" {
					label = container + "." + label
				}
				result = append(result, Origin{label, declaration, 1})
			}
			if len(result) > 0 {
				return result
			}
			return []Origin{{symbol.Name, nil, 1}}
		}
	case ast.KindIdentifier:
		if symbol := tc.GetSymbolAtLocation(n); symbol != nil {
			return []Origin{{symbol.Name, symbol.ValueDeclaration, 2}}
		}
	case ast.KindCallExpression:
		if signature := tc.GetResolvedSignature(n); signature != nil && signature.Declaration() != nil && signature.Declaration().Name() != nil {
			return []Origin{{signature.Declaration().Name().Text() + "()", signature.Declaration(), 2}}
		}
	}
	return []Origin{{"value", nil, 3}}
}

// container returns the name of the type alias, interface, or class enclosing declaration, when no function, variable, or other named declaration intervenes.
func container(declaration *ast.Node) string {
	for n := declaration.Parent; n != nil && n.Kind != ast.KindSourceFile; n = n.Parent {
		switch n.Kind {
		case ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration, ast.KindClassDeclaration, ast.KindClassExpression:
			if n.Name() != nil {
				return n.Name().Text()
			}
			return ""
		}
		if n.Name() != nil {
			return ""
		}
	}
	return ""
}

func firstDeclaration(symbol *ast.Symbol) *ast.Node {
	if len(symbol.Declarations) == 0 {
		return nil
	}
	return symbol.Declarations[0]
}

// FromDependency reports whether the vocabulary n reads is declared only outside project code: every origin has a
// declaration, in a declaration file or inside node_modules.
func (m *Analyzer) FromDependency(n *ast.Node) bool {
	origins := m.Origins(n)
	return len(origins) > 0 && !slices.ContainsFunc(origins, func(o Origin) bool {
		return o.Declaration == nil || ProjectFile(ast.GetSourceFileOfNode(o.Declaration))
	})
}
