package model

import (
	"slices"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Value is data reached from a root through property reads. The root is a variable (Root), this in a function or class
// (This), or an expression evaluated once (Base), such as a call: two calls are different values even when written
// alike.
type Value struct {
	Node *ast.Node
	Root *ast.Symbol
	This *ast.Node
	Base *ast.Node
	Path []string
	// Partial marks some unknown part of the data at Path, such as what a method call or a computed key reads.
	Partial bool
}

// ValueOf reads n as a value: a root followed by property reads, optional or not, and element reads with a literal
// key. A variable that only names other data is read as that data.
func (m *Analyzer) ValueOf(n *ast.Node) Value {
	n = Unwrap(n)
	v := Value{Node: n, Path: []string{}}
	c := n
	for ; ; c = Unwrap(c.Expression()) {
		if c.Kind == ast.KindPropertyAccessExpression {
			v.Path = append(v.Path, c.Name().Text())
			continue
		}
		if c.Kind == ast.KindElementAccessExpression {
			if key := Unwrap(c.AsElementAccessExpression().ArgumentExpression); ast.IsStringLiteralLike(key) || key.Kind == ast.KindNumericLiteral {
				v.Path = append(v.Path, key.Text())
				continue
			}
		}
		break
	}
	slices.Reverse(v.Path)
	switch c.Kind {
	case ast.KindThisKeyword:
		v.This = ast.GetThisContainer(c, false, false)
	case ast.KindIdentifier:
		v.Root = m.ctx.TypeChecker.GetSymbolAtLocation(c)
		if p := c.Parent; p != nil && p.Kind == ast.KindShorthandPropertyAssignment && p.Name() == c {
			v.Root = m.ctx.TypeChecker.GetShorthandAssignmentValueSymbol(p)
		}
	}
	if v.Root == nil && v.This == nil {
		v.Base = c
	}
	return m.through(v)
}

// through returns v read through the variable at its root when that variable only names other data (see alias).
func (m *Analyzer) through(v Value) Value {
	if v.Root == nil {
		return v
	}
	if named, ok := m.alias(v.Root); ok {
		v.Root, v.This, v.Path = named.Root, named.This, append(slices.Clone(named.Path), v.Path...)
	}
	return v
}

// alias returns the value a variable only names, so reading the variable reads that value: a const initialized with a
// read of a variable or this, or a binding destructured without a default from such a read or from a parameter, by
// property name or, from an array or tuple, by position, and never assigned.
func (m *Analyzer) alias(symbol *ast.Symbol) (Value, bool) {
	if found, ok := m.aliases[symbol]; ok {
		if found == nil {
			return Value{}, false
		}
		return *found, true
	}
	m.aliases[symbol] = nil
	v, ok := m.aliasOf(symbol)
	if ok {
		m.aliases[symbol] = &v
	}
	return v, ok
}

func (m *Analyzer) aliasOf(symbol *ast.Symbol) (Value, bool) {
	d := symbol.ValueDeclaration
	if d == nil || symbol.Flags&ast.SymbolFlagsVariable == 0 {
		return Value{}, false
	}
	path := []string{}
	for d.Kind == ast.KindBindingElement {
		element := d.AsBindingElement()
		if element.DotDotDotToken != nil || element.Initializer != nil {
			return Value{}, false
		}
		pattern := d.Parent
		switch pattern.Kind {
		case ast.KindObjectBindingPattern:
			key := d.PropertyName()
			if key == nil {
				key = d.Name()
			}
			if key.Kind != ast.KindIdentifier && !ast.IsStringOrNumericLiteralLike(key) {
				return Value{}, false
			}
			path = append(path, key.Text())
		case ast.KindArrayBindingPattern:
			if !m.positional(pattern) {
				return Value{}, false
			}
			path = append(path, strconv.Itoa(slices.Index(pattern.AsBindingPattern().Elements.Nodes, d)))
		default:
			return Value{}, false
		}
		d = pattern.Parent
	}
	slices.Reverse(path)
	var base Value
	switch {
	case d.Kind == ast.KindParameter && len(path) > 0 && d.Symbol() != nil:
		if m.assigned(symbol) {
			return Value{}, false
		}
		base = Value{Root: d.Symbol(), Path: []string{}}
	case d.Kind == ast.KindVariableDeclaration && ast.IsVarConst(d) && d.Initializer() != nil && !m.computed(d.Initializer()):
		base = m.ValueOf(d.Initializer())
		if base.Base != nil || base.Root == symbol || base.Root != nil && base.Root.Flags&ast.SymbolFlagsVariable == 0 {
			return Value{}, false
		}
	default:
		return Value{}, false
	}
	base.Node = nil
	base.Path = append(slices.Clone(base.Path), path...)
	return base, true
}

// positional reports whether n's type, without null and undefined, holds arrays or tuples only, whose elements are
// read by position.
func (m *Analyzer) positional(n *ast.Node) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil {
		return false
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	return !slices.ContainsFunc(members(t), func(member *checker.Type) bool {
		return !checker.IsTupleType(member) && !checker.Checker_isArrayType(m.ctx.TypeChecker, member)
	})
}

// computed reports whether a read goes through an accessor, which computes its value on each read.
func (m *Analyzer) computed(n *ast.Node) bool {
	for c := Unwrap(n); c.Kind == ast.KindPropertyAccessExpression || c.Kind == ast.KindElementAccessExpression; c = Unwrap(c.Expression()) {
		name := c
		if c.Kind == ast.KindPropertyAccessExpression {
			name = c.Name()
		}
		if symbol := m.ctx.TypeChecker.GetSymbolAtLocation(name); symbol != nil && symbol.Flags&ast.SymbolFlagsAccessor != 0 {
			return true
		}
	}
	return false
}

// assigned reports whether code in the function declaring symbol assigns it, by assignment, increment, decrement,
// destructuring assignment or as the variable of a for-in or for-of loop.
func (m *Analyzer) assigned(symbol *ast.Symbol) bool {
	scope := enclosingFunction(symbol.ValueDeclaration)
	if scope == nil {
		return false
	}
	tc := m.ctx.TypeChecker
	found := false
	var target func(*ast.Node)
	target = func(n *ast.Node) {
		n = Unwrap(n)
		switch n.Kind {
		case ast.KindIdentifier:
			found = found || tc.GetSymbolAtLocation(n) == symbol
		case ast.KindShorthandPropertyAssignment:
			found = found || tc.GetShorthandAssignmentValueSymbol(n) == symbol
		case ast.KindArrayLiteralExpression, ast.KindObjectLiteralExpression, ast.KindPropertyAssignment, ast.KindSpreadElement, ast.KindSpreadAssignment:
			n.ForEachChild(func(c *ast.Node) bool {
				if n.Kind != ast.KindPropertyAssignment || c == n.Initializer() {
					target(c)
				}
				return found
			})
		}
	}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		switch {
		case ast.IsAssignmentExpression(n, false):
			target(n.AsBinaryExpression().Left)
		case n.Kind == ast.KindPrefixUnaryExpression && (n.AsPrefixUnaryExpression().Operator == ast.KindPlusPlusToken || n.AsPrefixUnaryExpression().Operator == ast.KindMinusMinusToken):
			target(n.AsPrefixUnaryExpression().Operand)
		case n.Kind == ast.KindPostfixUnaryExpression:
			target(n.AsPostfixUnaryExpression().Operand)
		case n.Kind == ast.KindForInStatement || n.Kind == ast.KindForOfStatement:
			if initializer := n.AsForInOrOfStatement().Initializer; initializer.Kind != ast.KindVariableDeclarationList {
				target(initializer)
			}
		}
		if !found {
			n.ForEachChild(walk)
		}
		return found
	}
	walk(scope)
	return found
}

// Named reports whether v can be read again by name: its root is a variable or this.
func (v Value) Named() bool { return v.Base == nil }

func (v Value) extend(key string) Value {
	v.Path = append(slices.Clone(v.Path), key)
	return v
}

func (v Value) root() bool {
	return v.Root != nil || v.This != nil || v.Base != nil
}

func related(a, b []string) bool {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// overlaps reports whether v and o can denote the same data: one path contains the other. A partial value overlaps a
// longer path only when that path is partial too: an unknown part of `actor` is not known to be `actor.id`.
func (v Value) overlaps(o Value) bool {
	if v.Root != o.Root || v.This != o.This || v.Base != o.Base || !related(v.Path, o.Path) {
		return false
	}
	if v.Partial && !o.Partial && len(o.Path) > len(v.Path) {
		return false
	}
	return !(o.Partial && !v.Partial && len(v.Path) > len(o.Path))
}

// Overlapping reports whether any value of a can denote the same data as any value of b.
func Overlapping(a, b []Value) bool {
	return slices.ContainsFunc(a, func(v Value) bool { return slices.ContainsFunc(b, v.overlaps) })
}

// Definite reports whether any value of a is known to share data with any value of b: two unknown parts of one value,
// such as the results of two getters on it, are not known to be the same.
func Definite(a, b []Value) bool {
	return slices.ContainsFunc(a, func(v Value) bool {
		return slices.ContainsFunc(b, func(o Value) bool { return !(v.Partial && o.Partial) && v.overlaps(o) })
	})
}

// Exact keeps the values that denote known data, not an unknown part of it.
func Exact(values []Value) []Value {
	return slices.DeleteFunc(slices.Clone(values), func(v Value) bool { return v.Partial })
}

// String describes a value as written: its root and path, prefixed with "part of" when partial.
func (v Value) String() string {
	root, path := "this", v.Path
	switch {
	case v.Root != nil:
		root = v.Root.Name
		if d := v.Root.ValueDeclaration; d != nil && d.Kind == ast.KindParameter && ast.IsBindingPattern(d.Name()) {
			root, path = bound(d.Name(), path)
		}
	case v.Base != nil:
		root = Text(v.Base, 40)
	}
	text := strings.Join(append([]string{root}, path...), ".")
	if v.Partial {
		return "part of " + text
	}
	return text
}

// bound names the data at path in a destructured parameter by the variable the pattern binds to it, as written, or by
// the pattern when it binds none.
func bound(pattern *ast.Node, path []string) (string, []string) {
	for current, rest := pattern, path; len(rest) > 0 && ast.IsBindingPattern(current); {
		var next *ast.Node
		for position, element := range current.AsBindingPattern().Elements.Nodes {
			if element.Kind != ast.KindBindingElement || element.Name() == nil {
				continue
			}
			key := strconv.Itoa(position)
			if current.Kind == ast.KindObjectBindingPattern {
				name := element.PropertyName()
				if name == nil {
					name = element.Name()
				}
				if name.Kind != ast.KindIdentifier && !ast.IsStringOrNumericLiteralLike(name) {
					continue
				}
				key = name.Text()
			}
			if key == rest[0] {
				next = element.Name()
			}
		}
		if next == nil {
			break
		}
		current, rest = next, rest[1:]
		if current.Kind == ast.KindIdentifier {
			return current.Text(), rest
		}
	}
	return Text(pattern, 40), path
}

// Same reports whether two values are the same data.
func (v Value) Same(o Value) bool {
	return v.Root == o.Root && v.This == o.This && v.Base == o.Base && slices.Equal(v.Path, o.Path)
}

// Within reports whether v is o or data read from it.
func (v Value) Within(o Value) bool {
	return v.Root == o.Root && v.This == o.This && v.Base == o.Base && len(v.Path) >= len(o.Path) && slices.Equal(v.Path[:len(o.Path)], o.Path)
}

// Reads reports whether n is a value within v, or a property or element read from one.
func (m *Analyzer) Reads(n *ast.Node, v Value) bool {
	for c := Unwrap(n); c != nil; c = Unwrap(c.Expression()) {
		if m.ValueOf(c).Within(v) {
			return true
		}
		if c.Kind != ast.KindPropertyAccessExpression && c.Kind != ast.KindElementAccessExpression {
			return false
		}
	}
	return false
}

// SameValue reports whether two expressions are the same value.
func (m *Analyzer) SameValue(a, b *ast.Node) bool {
	return m.ValueOf(a).Same(m.ValueOf(b))
}

// Uses reports whether evaluating n reads v.
func (m *Analyzer) Uses(n *ast.Node, v Value) bool {
	return m.usesBeyond(n, v, Value{}, false)
}

// usesBeyond reports whether evaluating n reads v, or data v is read from, other than by reading tested, a value within
// v that a test examines, to test it again, or at all when mentions is true.
func (m *Analyzer) usesBeyond(n *ast.Node, v Value, tested Value, mentions bool) bool {
	found := false
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if found {
			return true
		}
		if c.Kind == ast.KindIdentifier || c.Kind == ast.KindThisKeyword {
			read := c
			for p := read.Parent; p != nil && (p.Kind == ast.KindPropertyAccessExpression || p.Kind == ast.KindElementAccessExpression && ast.IsStringOrNumericLiteralLike(Unwrap(p.AsElementAccessExpression().ArgumentExpression))) && p.Expression() == read; p = read.Parent {
				read = p
			}
			value := m.ValueOf(read)
			if value.Within(v) || v.Within(value) {
				found = !value.Same(tested) || !mentions && !m.retested(read)
			}
			return found
		}
		c.ForEachChild(walk)
		return found
	}
	walk(n)
	return found
}

// Field is what a value names across code: the declarations of the property it ends at, or of the variable it is.
// Symbol is that property or variable, or nil when it is not found.
type Field struct {
	Name         string
	Declarations []*ast.Node
	Symbol       *ast.Symbol
}

// Same reports whether two fields have the same name and share a declaration.
func (f Field) Same(o Field) bool {
	return f.Name == o.Name && slices.ContainsFunc(f.Declarations, func(d *ast.Node) bool { return slices.Contains(o.Declarations, d) })
}

// FieldOf returns the field n reads. A property is looked up on its object's declared type, so narrowing the object
// does not change it, or on the narrowed type when only some members of the declared type have it.
func (m *Analyzer) FieldOf(n *ast.Node) Field {
	n = Unwrap(n)
	tc := m.ctx.TypeChecker
	if n.Kind == ast.KindPropertyAccessExpression {
		name := n.Name().Text()
		if n.Name().Kind == ast.KindPrivateIdentifier {
			if property := tc.GetSymbolAtLocation(n.Name()); property != nil && len(property.Declarations) > 0 {
				return Field{Name: name, Declarations: declarations(property.Declarations), Symbol: property}
			}
			return Field{Name: name}
		}
		for _, t := range []*checker.Type{m.ReceiverType(n.Expression()), tc.GetTypeAtLocation(n.Expression())} {
			if t == nil {
				continue
			}
			if property := tc.GetPropertyOfType(tc.GetNonNullableType(t), name); property != nil && len(property.Declarations) > 0 {
				return Field{Name: name, Declarations: declarations(property.Declarations), Symbol: property}
			}
		}
		return Field{Name: name}
	}
	if symbol := tc.GetSymbolAtLocation(n); symbol != nil && symbol.ValueDeclaration != nil {
		if field, ok := m.bound(symbol.ValueDeclaration); ok {
			return field
		}
		return Field{Name: symbol.Name, Declarations: []*ast.Node{symbol.ValueDeclaration}, Symbol: symbol}
	}
	return Field{}
}

// bound returns the property a destructured variable is bound to: in const { a } = value or function f({ a }: T), a is
// the field a of the destructured type. Rest elements and computed names bind no single property.
func (m *Analyzer) bound(declaration *ast.Node) (Field, bool) {
	if declaration.Kind != ast.KindBindingElement || declaration.Parent == nil || declaration.Parent.Kind != ast.KindObjectBindingPattern || declaration.AsBindingElement().DotDotDotToken != nil {
		return Field{}, false
	}
	key := declaration.PropertyName()
	if key == nil {
		key = declaration.Name()
	}
	if key.Kind != ast.KindIdentifier && !ast.IsStringLiteralLike(key) {
		return Field{}, false
	}
	tc := m.ctx.TypeChecker
	t := tc.GetTypeAtLocation(declaration.Parent)
	if t == nil {
		return Field{}, false
	}
	property := tc.GetPropertyOfType(tc.GetNonNullableType(t), key.Text())
	if property == nil || len(property.Declarations) == 0 {
		return Field{}, false
	}
	return Field{Name: key.Text(), Declarations: declarations(property.Declarations), Symbol: property}, true
}

// declarations returns nodes once each, by file and position: a property of a union lists a declaration once per member
// that has it, and how many members the checker has resolved can vary.
func declarations(nodes []*ast.Node) []*ast.Node {
	sorted := slices.Clone(nodes)
	slices.SortFunc(sorted, func(a, b *ast.Node) int {
		if byFile := strings.Compare(ast.GetSourceFileOfNode(a).FileName(), ast.GetSourceFileOfNode(b).FileName()); byFile != 0 {
			return byFile
		}
		return a.Pos() - b.Pos()
	})
	return slices.Compact(sorted)
}
