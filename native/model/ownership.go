package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// ReceiverType returns the declared type of a read, so narrowing does not change it, or its type where it is read when
// it has no declared type.
func (m *Analyzer) ReceiverType(n *ast.Node) *checker.Type {
	if t := m.Declared(n); t != nil {
		return t
	}
	return m.ctx.TypeChecker.GetTypeAtLocation(Unwrap(n))
}

func (m *Analyzer) ownerFromSymbol(symbol *ast.Symbol, seen map[*ast.Symbol]bool) *ast.Symbol {
	if symbol != nil && symbol.Flags&ast.SymbolFlagsAlias != 0 {
		symbol = m.ctx.TypeChecker.GetAliasedSymbol(symbol)
	}
	if symbol == nil || seen[symbol] {
		return nil
	}
	seen[symbol] = true
	if symbol.Flags&ast.SymbolFlagsClass != 0 || symbol.Flags&ast.SymbolFlagsType != 0 && symbol.Flags&ast.SymbolFlagsValue != 0 {
		return symbol
	}
	if symbol.Flags&ast.SymbolFlagsTypeAlias != 0 {
		for _, declaration := range symbol.Declarations {
			n := declaration.Type()
			for n != nil && n.Kind == ast.KindParenthesizedType {
				n = n.Type()
			}
			if n != nil && n.Kind == ast.KindTypeReference {
				return m.ownerFromTypeNode(n, seen)
			}
		}
		return symbol
	}
	if symbol.Flags&ast.SymbolFlagsInterface != 0 {
		return symbol
	}
	return nil
}

func (m *Analyzer) ownerFromTypeNode(n *ast.Node, seen map[*ast.Symbol]bool) *ast.Symbol {
	if n == nil {
		return nil
	}
	if n.Kind == ast.KindParenthesizedType {
		return m.ownerFromTypeNode(n.Type(), seen)
	}
	if n.Kind != ast.KindTypeReference {
		return nil
	}
	return m.ownerFromSymbol(m.ctx.TypeChecker.GetSymbolAtLocation(n.AsTypeReferenceNode().TypeName), seen)
}

func (m *Analyzer) Owner(receiver *ast.Node) *ast.Symbol {
	t := m.ReceiverType(receiver)
	if t == nil {
		return nil
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	if t.Flags()&checker.TypeFlagsInstantiable != 0 {
		if constraint := checker.Checker_getBaseConstraintOfType(m.ctx.TypeChecker, t); constraint != nil {
			t = m.ctx.TypeChecker.GetNonNullableType(constraint)
		}
	}
	if t.Flags()&(checker.TypeFlagsAny|checker.TypeFlagsUnknown|checker.TypeFlagsNever) != 0 {
		return nil
	}
	t = m.mappedFrom(t)
	symbol := t.Symbol()
	if symbol == nil || symbol.Flags&ast.SymbolFlagsClass == 0 {
		symbol = m.ownerFromSymbol(t.Alias().Symbol(), map[*ast.Symbol]bool{})
		if symbol == nil {
			if receiverSymbol := m.ctx.TypeChecker.GetSymbolAtLocation(receiver); receiverSymbol != nil {
				for _, declaration := range receiverSymbol.Declarations {
					if candidate := m.ownerFromTypeNode(declaration.Type(), map[*ast.Symbol]bool{}); candidate != nil {
						symbol = candidate
						break
					}
				}
			}
		}
		if symbol == nil {
			symbol = m.ownerFromSymbol(t.Symbol(), map[*ast.Symbol]bool{})
		}
	}
	return m.established(symbol)
}

func (m *Analyzer) Destination(result *ast.Node) *ast.Symbol {
	result = Unwrap(result)
	var t *checker.Type
	if result.Kind == ast.KindObjectLiteralExpression || result.Kind == ast.KindArrayLiteralExpression {
		t = checker.Checker_getContextualType(m.ctx.TypeChecker, result, checker.ContextFlagsNone)
	} else {
		t = m.ctx.TypeChecker.GetTypeAtLocation(result)
	}
	if t == nil {
		return nil
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	if t.Flags()&(checker.TypeFlagsAny|checker.TypeFlagsUnknown|checker.TypeFlagsNever|checker.TypeFlagsInstantiable) != 0 {
		return nil
	}
	if symbol := t.Symbol(); symbol != nil && symbol.Flags&ast.SymbolFlagsClass != 0 {
		return m.established(symbol)
	}
	if symbol := m.ownerFromSymbol(t.Alias().Symbol(), map[*ast.Symbol]bool{}); symbol != nil {
		return m.established(symbol)
	}
	return m.established(m.ownerFromSymbol(t.Symbol(), map[*ast.Symbol]bool{}))
}

func (m *Analyzer) ProjectOwner(symbol *ast.Symbol) *ast.Symbol {
	return m.established(symbol)
}

func (m *Analyzer) established(symbol *ast.Symbol) *ast.Symbol {
	if !ProjectSymbol(symbol) {
		return nil
	}
	for _, declaration := range symbol.Declarations {
		file := ast.GetSourceFileOfNode(declaration)
		for parent := declaration; parent != nil; parent = parent.Parent {
			if parent.ModifierFlags()&ast.ModifierFlagsAmbient != 0 {
				return nil
			}
		}
		boundary := moduleBoundary(declaration)
		if symbol.Flags&ast.SymbolFlagsValue == 0 && (boundary == nil || !ast.IsExternalModule(file) && boundary.Kind == ast.KindSourceFile) {
			return nil
		}
	}
	return symbol
}

func moduleBoundary(n *ast.Node) *ast.Node {
	for n != nil {
		if n.Kind == ast.KindModuleDeclaration || n.Kind == ast.KindSourceFile {
			return n
		}
		n = n.Parent
	}
	return nil
}

func (m *Analyzer) sameModule(a, b *ast.Node) bool {
	a, b = moduleBoundary(a), moduleBoundary(b)
	if a == nil || b == nil {
		return false
	}
	if a == b {
		return true
	}
	if a.Kind != ast.KindModuleDeclaration || b.Kind != ast.KindModuleDeclaration {
		return false
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(a.Name())
	return symbol != nil && symbol == m.ctx.TypeChecker.GetSymbolAtLocation(b.Name())
}

func (m *Analyzer) InsideOwner(subject *ast.Node, owner *ast.Symbol) bool {
	if owner.Flags&ast.SymbolFlagsValue == 0 {
		for _, declaration := range owner.Declarations {
			if m.sameModule(subject, declaration) {
				return true
			}
		}
		return false
	}
	for parent := subject.Parent; parent != nil; parent = parent.Parent {
		if !ast.IsClassLike(parent) && parent.Kind != ast.KindVariableDeclaration && parent.Kind != ast.KindModuleDeclaration {
			continue
		}
		if name := parent.Name(); name != nil && m.ctx.TypeChecker.GetSymbolAtLocation(name) == owner {
			return true
		}
		if ast.IsClassLike(parent) {
			return false
		}
	}
	return false
}

// Extends reports whether class is base or derives from it, at any depth.
func (m *Analyzer) Extends(class *ast.Node, base *ast.Node) bool {
	seen := map[*ast.Node]bool{}
	for class != nil && !seen[class] {
		if class == base {
			return true
		}
		seen[class] = true
		heritage := ast.GetClassExtendsHeritageElement(class)
		if heritage == nil {
			return false
		}
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(heritage.Expression())
		if symbol == nil {
			return false
		}
		symbol = checker.SkipAlias(symbol, m.ctx.TypeChecker)
		class = nil
		for _, declaration := range symbol.Declarations {
			if ast.IsClassLike(declaration) {
				class = declaration
			}
		}
	}
	return false
}

// InsideMemberOwner reports whether n is written in a class that has member, declared there or inherited: the code of
// the member's owner or of a class extending it, mixins included. InsideOwner, by contrast, asks whether code is
// written inside an owner's own declaration.
func (m *Analyzer) InsideMemberOwner(n, member *ast.Node) bool {
	tc := m.ctx.TypeChecker
	for q := n.Parent; q != nil; q = q.Parent {
		if !ast.IsClassLike(q) || q.Symbol() == nil {
			continue
		}
		for _, property := range checker.Checker_getPropertiesOfType(tc, tc.GetDeclaredTypeOfSymbol(q.Symbol())) {
			if slices.Contains(property.Declarations, member) {
				return true
			}
		}
	}
	return false
}

// ObjectOwner returns the project declaration naming n's object type without null and undefined: its alias, otherwise
// its symbol, with that type. It returns nil when the type is not an object type or the project does not declare it.
func (m *Analyzer) ObjectOwner(n *ast.Node) (*ast.Symbol, *checker.Type) {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil {
		return nil, nil
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	if t.Flags()&checker.TypeFlagsObject == 0 {
		return nil, nil
	}
	symbol := t.Symbol()
	if alias := t.Alias(); alias != nil {
		symbol = alias.Symbol()
	}
	if !ProjectSymbol(symbol) {
		return nil, nil
	}
	return symbol, t
}

// ContextualOwner returns the established project owner of the type n is written for, and whether n has such a type.
func (m *Analyzer) ContextualOwner(n *ast.Node) (*ast.Symbol, bool) {
	t := checker.Checker_getContextualType(m.ctx.TypeChecker, n, checker.ContextFlagsNone)
	if t == nil {
		return nil, false
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	symbol := t.Alias().Symbol()
	if symbol == nil {
		symbol = t.Symbol()
	}
	return m.ProjectOwner(symbol), true
}

// Variable returns the variable n names, through import aliases, or nil.
func (m *Analyzer) Variable(n *ast.Node) *ast.Symbol {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
	if symbol == nil || checker.SkipAlias(symbol, m.ctx.TypeChecker).Flags&ast.SymbolFlagsVariable == 0 {
		return nil
	}
	return symbol
}

// mappedFrom returns the type a mapped alias over keyof one of its type parameters maps, such as T for Readonly<T> or
// Partial<T>, when the alias is not itself owned, repeating through nested mapped aliases. Other types are returned
// unchanged.
func (m *Analyzer) mappedFrom(t *checker.Type) *checker.Type {
	for range 4 {
		alias := t.Alias()
		if alias == nil || m.established(m.ownerFromSymbol(alias.Symbol(), map[*ast.Symbol]bool{})) != nil {
			return t
		}
		index := mappedParameter(alias.Symbol())
		if index < 0 || index >= len(alias.TypeArguments()) {
			return t
		}
		t = m.ctx.TypeChecker.GetNonNullableType(alias.TypeArguments()[index])
	}
	return t
}

// mappedParameter returns the position of the type parameter T that a type alias declared as { [P in keyof T]: ... }
// maps, or -1.
func mappedParameter(alias *ast.Symbol) int {
	if alias == nil || len(alias.Declarations) != 1 || alias.Declarations[0].Kind != ast.KindTypeAliasDeclaration {
		return -1
	}
	declaration := alias.Declarations[0]
	mapped := declaration.Type()
	if mapped == nil || mapped.Kind != ast.KindMappedType {
		return -1
	}
	constraint := mapped.AsMappedTypeNode().TypeParameter.AsTypeParameterDeclaration().Constraint
	if constraint == nil || constraint.Kind != ast.KindTypeOperator || constraint.AsTypeOperatorNode().Operator != ast.KindKeyOfKeyword {
		return -1
	}
	target := constraint.AsTypeOperatorNode().Type
	if target.Kind != ast.KindTypeReference || target.AsTypeReferenceNode().TypeName.Kind != ast.KindIdentifier {
		return -1
	}
	name := target.AsTypeReferenceNode().TypeName.Text()
	for i, parameter := range declaration.TypeParameters() {
		if parameter.Name().Text() == name {
			return i
		}
	}
	return -1
}
