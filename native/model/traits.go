package model

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

func (m *Analyzer) objectLike(t *checker.Type) bool {
	return t.Flags()&checker.TypeFlagsObject != 0 &&
		!m.ctx.TypeChecker.IsArrayType(t) && !checker.IsTupleType(t) &&
		!m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(t)
}

// Owned reports whether the project declares t: its alias, otherwise its symbol, otherwise every member.
func (m *Analyzer) Owned(t *checker.Type) bool {
	if alias := t.Alias(); alias != nil {
		return ProjectSymbol(alias.Symbol())
	}
	if t.Flags()&(checker.TypeFlagsUnion|checker.TypeFlagsIntersection) != 0 {
		for _, member := range t.Types() {
			if !m.Owned(member) {
				return false
			}
		}
		return true
	}
	return ProjectSymbol(t.Symbol())
}

// Forgeable reports whether t is an object type, or a union or intersection of them, that is not a class instance
// type and has no private, protected, #private, or unique symbol members: other code can produce one without the
// checks of the code that made it, so the type does not keep them.
func (m *Analyzer) Forgeable(t *checker.Type) bool {
	if t.Flags()&(checker.TypeFlagsUnion|checker.TypeFlagsIntersection) != 0 {
		for _, member := range t.Types() {
			if !m.Forgeable(member) {
				return false
			}
		}
		return true
	}
	if !m.objectLike(t) {
		return false
	}
	if symbol := t.Symbol(); symbol != nil && symbol.Flags&ast.SymbolFlagsClass != 0 {
		return false
	}
	for _, property := range m.ctx.TypeChecker.GetPropertiesOfType(t) {
		if checker.GetDeclarationModifierFlagsFromSymbol(property)&(ast.ModifierFlagsPrivate|ast.ModifierFlagsProtected) != 0 {
			return false
		}
		for _, declaration := range property.Declarations {
			name := declaration.Name()
			if name == nil {
				continue
			}
			if name.Kind == ast.KindPrivateIdentifier {
				return false
			}
			if m.brand(declaration) {
				return false
			}
		}
	}
	return true
}

// Unwrap replaces a promise with its awaited type and removes null and undefined.
func (m *Analyzer) Unwrap(t *checker.Type) *checker.Type {
	if t == nil {
		return nil
	}
	if awaited := checker.Checker_getAwaitedType(m.ctx.TypeChecker, t); awaited != nil {
		t = awaited
	}
	return m.ctx.TypeChecker.GetNonNullableType(t)
}

func (m *Analyzer) objects(t *checker.Type) bool {
	if t.Flags()&(checker.TypeFlagsUnion|checker.TypeFlagsIntersection) != 0 {
		for _, member := range t.Types() {
			if !m.objects(member) {
				return false
			}
		}
		return true
	}
	return m.objectLike(t)
}

// Unforgeable reports whether t is a project-owned object type that only code with access to its private state or
// brand can produce, so a check made while producing it is kept by the type.
func (m *Analyzer) Unforgeable(t *checker.Type) bool {
	return t != nil && m.objects(t) && m.Owned(t) && !m.Forgeable(t)
}

// Stateful reports whether t holds data: an index signature, or a property that is not a method and whose type is
// not a function.
func (m *Analyzer) Stateful(t *checker.Type) bool {
	if len(m.ctx.TypeChecker.GetIndexInfosOfType(t)) > 0 {
		return true
	}
	for _, property := range m.ctx.TypeChecker.GetPropertiesOfType(t) {
		if property.Flags&ast.SymbolFlagsMethod != 0 {
			continue
		}
		value := m.ctx.TypeChecker.GetNonNullableType(m.ctx.TypeChecker.GetTypeOfSymbol(property))
		if !m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(value) {
			return true
		}
	}
	return false
}

// Result returns what a function produces: its return type, awaited and without null and undefined.
func (m *Analyzer) Result(fn *ast.Node) *checker.Type {
	signature := m.ctx.TypeChecker.GetSignatureFromDeclaration(fn)
	if signature == nil {
		return nil
	}
	return m.Unwrap(m.ctx.TypeChecker.GetReturnTypeOfSignature(signature))
}

// brand reports whether a member declaration is keyed by a unique symbol.
func (m *Analyzer) brand(declaration *ast.Node) bool {
	name := declaration.Name()
	if name == nil || name.Kind != ast.KindComputedPropertyName {
		return false
	}
	key := m.ctx.TypeChecker.GetTypeAtLocation(name.Expression())
	return key != nil && key.Flags()&checker.TypeFlagsUniqueESSymbol != 0
}

// phantom reports whether every member of t is a brand. Such a type describes no runtime shape: opaque types use it to
// stand for primitives.
func (m *Analyzer) phantom(t *checker.Type) bool {
	properties := m.ctx.TypeChecker.GetPropertiesOfType(t)
	if len(properties) == 0 || m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(t) {
		return false
	}
	for _, property := range properties {
		if len(property.Declarations) == 0 {
			return false
		}
		for _, declaration := range property.Declarations {
			if !m.brand(declaration) {
				return false
			}
		}
	}
	return true
}
