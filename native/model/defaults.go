package model

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// Default is where code decides what an absent optional property stands for: a `??` or `??=` whose left operand reads
// the property, or a default of an object binding pattern destructuring it, with a constant (see Provenance) of the
// property's own type, without undefined and null, as Value. Property is the declaration of the property, every
// declaration of it belonging to Owner (see MemberOwner), Type that type where it is read, as the receiver's type
// instantiates it, and Inside tells whether the code is written inside that owner (see InsideDeclaringOwner). Outside
// it, the meaning of absence is not part of the type: another reader of the property can decide it otherwise.
type Default struct {
	Node     *ast.Node
	Name     string
	Property *ast.Node
	Value    *ast.Node
	Owner    *ast.Symbol
	Type     *checker.Type
	Inside   bool
}

// DefaultOf reads n as a default of an optional property.
func (m *Analyzer) DefaultOf(n *ast.Node) (Default, bool) {
	if known, ok := m.defaults[n]; ok {
		return known.Default, known.ok
	}
	d, ok := m.defaultOf(n)
	m.defaults[n] = knownDefault{d, ok}
	return d, ok
}

type knownDefault struct {
	Default
	ok bool
}

func (m *Analyzer) defaultOf(n *ast.Node) (Default, bool) {
	var symbol *ast.Symbol
	var value *ast.Node
	tc := m.ctx.TypeChecker
	switch n.Kind {
	case ast.KindBinaryExpression:
		b := n.AsBinaryExpression()
		read := Unwrap(b.Left)
		if b.OperatorToken.Kind != ast.KindQuestionQuestionToken && b.OperatorToken.Kind != ast.KindQuestionQuestionEqualsToken || read.Kind != ast.KindPropertyAccessExpression {
			return Default{}, false
		}
		symbol, value = tc.GetSymbolAtLocation(read.Name()), Unwrap(b.Right)
	case ast.KindBindingElement:
		pattern := n.Parent
		element := n.AsBindingElement()
		if pattern.Kind != ast.KindObjectBindingPattern || element.Initializer == nil || element.DotDotDotToken != nil {
			return Default{}, false
		}
		name := element.PropertyName
		if name == nil {
			name = n.Name()
		}
		if name.Kind != ast.KindIdentifier {
			return Default{}, false
		}
		object := tc.GetTypeAtLocation(pattern)
		if object == nil {
			return Default{}, false
		}
		symbol, value = tc.GetPropertyOfType(tc.GetNonNullableType(object), name.Text()), Unwrap(element.Initializer)
	default:
		return Default{}, false
	}
	if symbol == nil || len(symbol.Declarations) == 0 || !m.unit(value) {
		return Default{}, false
	}
	declared := tc.GetTypeOfSymbol(symbol)
	if declared == nil || declared.Flags()&checker.TypeFlagsAnyOrUnknown != 0 || symbol.Flags&ast.SymbolFlagsOptional == 0 && !m.missableType(declared) {
		return Default{}, false
	}
	present := tc.GetNonNullableType(declared)
	if !checker.Checker_isTypeAssignableTo(tc, tc.GetTypeAtLocation(value), present) {
		return Default{}, false
	}
	property := symbol.Declarations[0]
	owner := m.MemberOwner(property)
	if owner == nil {
		return Default{}, false
	}
	for _, d := range symbol.Declarations[1:] {
		if m.MemberOwner(d) != owner {
			return Default{}, false
		}
	}
	return Default{Node: n, Name: symbol.Name, Property: property, Value: value, Owner: owner, Type: present, Inside: m.InsideDeclaringOwner(n, property)}, true
}

// defaultName returns the name of the property a default may be of, read from syntax alone.
func defaultName(n *ast.Node) string {
	switch n.Kind {
	case ast.KindBinaryExpression:
		b := n.AsBinaryExpression()
		operator := b.OperatorToken.Kind
		if read := ast.SkipParentheses(b.Left); (operator == ast.KindQuestionQuestionToken || operator == ast.KindQuestionQuestionEqualsToken) && read.Kind == ast.KindPropertyAccessExpression && read.Name().Kind == ast.KindIdentifier {
			return read.Name().Text()
		}
	case ast.KindBindingElement:
		element := n.AsBindingElement()
		if n.Parent.Kind != ast.KindObjectBindingPattern || element.Initializer == nil {
			return ""
		}
		name := element.PropertyName
		if name == nil {
			name = n.Name()
		}
		if name.Kind == ast.KindIdentifier {
			return name.Text()
		}
	}
	return ""
}

// defaultSites lists the places of a program's project files that may be defaults, by property name. It is built once
// per program from syntax alone and never mutated afterwards.
type defaultSites struct {
	byName map[string][]*ast.Node
}

var defaultIndex = NewShared(func(program *compiler.Program) *defaultSites {
	_, partial := Collect(program, func(n *ast.Node) bool { return defaultName(n) != "" })
	index := &defaultSites{byName: map[string][]*ast.Node{}}
	for _, sites := range partial {
		for _, site := range sites {
			index.byName[defaultName(site)] = append(index.byName[defaultName(site)], site)
		}
	}
	return index
})

// DefaultsLike returns the defaults of d's property, with the type it has at d, across the program's project files, d
// included, in program order.
func (m *Analyzer) DefaultsLike(d Default) []Default {
	found := []Default{}
	for _, site := range defaultIndex.For(m.ctx.Program).byName[defaultName(d.Node)] {
		if other, ok := m.DefaultOf(site); ok && other.Property == d.Property && other.Type == d.Type {
			found = append(found, other)
		}
	}
	return found
}

// SameDefault reports whether two defaults give the same value.
func (m *Analyzer) SameDefault(a, b Default) bool {
	tc := m.ctx.TypeChecker
	return tc.GetTypeAtLocation(a.Value) == tc.GetTypeAtLocation(b.Value)
}
