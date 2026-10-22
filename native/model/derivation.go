package model

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// Derivation is a value computed from one property of data, outside the property's owner: a call whose only argument
// reads the property, making no effect, by a callee that changes no data outside it (see changesOutside), whose result is used and is not a boolean, which a test would establish (see
// Facts). The property is an instance's (see instanceData), and the callee a function computing from its arguments
// alone (see computesFromArguments), not a method of an instance, whose data it would also read. Callee is the declaration the call runs (see Implementation), Property the property's declaration, every
// declaration of it belonging to Owner (see MemberOwner), and Reader the function the call is written in, or its file.
type Derivation struct {
	Node     *ast.Node
	Callee   *ast.Node
	Property *ast.Node
	Owner    *ast.Symbol
	Reader   *ast.Node
}

// DerivationOf reads n as a derivation.
func (m *Analyzer) DerivationOf(n *ast.Node) (Derivation, bool) {
	if derivedName(n) == "" || discarded(n) {
		return Derivation{}, false
	}
	tc := m.ctx.TypeChecker
	callee := Implementation(tc.GetResolvedSignature(n))
	symbol := tc.GetSymbolAtLocation(Unwrap(n.Arguments()[0]).Name())
	if callee == nil || symbol == nil || len(symbol.Declarations) == 0 {
		return Derivation{}, false
	}
	property := symbol.Declarations[0]
	if !computesFromArguments(callee) || !instanceData(property) || len(tc.GetSignaturesOfType(tc.GetNonNullableType(tc.GetTypeOfSymbol(symbol)), checker.SignatureKindCall)) > 0 {
		return Derivation{}, false
	}
	owner := m.MemberOwner(property)
	if owner == nil || m.InsideDeclaringOwner(n, property) {
		return Derivation{}, false
	}
	for _, d := range symbol.Declarations[1:] {
		if m.MemberOwner(d) != owner {
			return Derivation{}, false
		}
	}
	result := tc.GetTypeAtLocation(n)
	if result == nil || result.Flags()&(checker.TypeFlagsBooleanLike|checker.TypeFlagsVoid|checker.TypeFlagsNever) != 0 || len(m.EffectsIn([]*ast.Node{n})) > 0 || m.followed(n) != nil && m.changesOutside(callee) {
		return Derivation{}, false
	}
	reader := enclosingFunction(n)
	if reader == nil {
		reader = ast.GetSourceFileOfNode(n).AsNode()
	}
	return Derivation{Node: n, Callee: callee, Property: property, Owner: owner, Reader: reader}, true
}

// changesOutside reports whether the body of a function makes an effect on data declared outside it, such as module
// state or, for a static method, its class. Effects of the functions it calls are read through their summaries, which
// list only those on their receiver and parameters.
func (m *Analyzer) changesOutside(fn *ast.Node) bool {
	for _, e := range m.EffectsIn([]*ast.Node{fn.Body()}) {
		if e.Receiver.This != nil || e.Receiver.Root != nil && e.Receiver.Root.ValueDeclaration != nil && !encloses(fn, e.Receiver.Root.ValueDeclaration) {
			return true
		}
	}
	return false
}

// computesFromArguments reports whether a callee is a function of its arguments alone: a function declaration or
// expression, an arrow function, or a static method or one of an object literal, rather than a method or call signature
// of an instance.
func computesFromArguments(callee *ast.Node) bool {
	switch callee.Kind {
	case ast.KindFunctionDeclaration, ast.KindFunctionExpression, ast.KindArrowFunction:
		return true
	case ast.KindMethodDeclaration:
		return callee.Parent.Kind == ast.KindObjectLiteralExpression || ast.HasSyntacticModifier(callee, ast.ModifierFlagsStatic)
	}
	return false
}

// instanceData reports whether a property declaration holds data of each instance: a property signature, or a property
// declaration or getter that is not static.
func instanceData(property *ast.Node) bool {
	switch property.Kind {
	case ast.KindPropertySignature:
		return true
	case ast.KindPropertyDeclaration, ast.KindGetAccessor:
		return !ast.HasSyntacticModifier(property, ast.ModifierFlagsStatic)
	}
	return false
}

// derivedName returns the name of the property a call may derive a value from, read from syntax alone: its only
// argument's property name, or "".
func derivedName(n *ast.Node) string {
	if n.Kind != ast.KindCallExpression || len(n.Arguments()) != 1 {
		return ""
	}
	if read := Unwrap(n.Arguments()[0]); read.Kind == ast.KindPropertyAccessExpression {
		return read.Name().Text()
	}
	return ""
}

// derivationSites lists the calls of a program's project files that may be derivations, by property name. It is built
// once per program from syntax alone and never mutated afterwards.
type derivationSites struct {
	byName map[string][]*ast.Node
}

var derivationIndex = NewShared(func(program *compiler.Program) *derivationSites {
	_, partial := Collect(program, func(n *ast.Node) bool { return derivedName(n) != "" })
	index := &derivationSites{byName: map[string][]*ast.Node{}}
	for _, sites := range partial {
		for _, site := range sites {
			index.byName[derivedName(site)] = append(index.byName[derivedName(site)], site)
		}
	}
	return index
})

// DerivationsLike returns the derivations by d's callee from d's property across the program's project files, d
// included, in program order.
func (m *Analyzer) DerivationsLike(d Derivation) []Derivation {
	found := []Derivation{}
	for _, site := range derivationIndex.For(m.ctx.Program).byName[derivedName(d.Node)] {
		if other, ok := m.DerivationOf(site); ok && other.Property == d.Property && other.Callee == d.Callee {
			found = append(found, other)
		}
	}
	return found
}
