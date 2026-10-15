package model

import (
	"slices"
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
// key.
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
	}
	if v.Root == nil && v.This == nil {
		v.Base = c
	}
	return v
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
	root := "this"
	switch {
	case v.Root != nil:
		root = v.Root.Name
	case v.Base != nil:
		root = Text(v.Base, 40)
	}
	text := strings.Join(append([]string{root}, v.Path...), ".")
	if v.Partial {
		return "part of " + text
	}
	return text
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

// usesBeyond reports whether evaluating n reads v other than by reading tested, a value within v that a test examines,
// to test it again, or at all when mentions is true.
func (m *Analyzer) usesBeyond(n *ast.Node, v Value, tested Value, mentions bool) bool {
	found := false
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if found {
			return true
		}
		if c.Kind == ast.KindIdentifier || c.Kind == ast.KindThisKeyword {
			if m.ValueOf(c).Within(v) {
				r := m.reread(c, tested)
				found = r == nil || !mentions && !retested(r)
				return found
			}
		}
		c.ForEachChild(walk)
		return found
	}
	walk(n)
	return found
}

func (m *Analyzer) reread(root *ast.Node, tested Value) *ast.Node {
	for c := root; c.Parent != nil && (c.Parent.Kind == ast.KindPropertyAccessExpression || c.Parent.Kind == ast.KindElementAccessExpression) && c.Parent.Expression() == c; c = c.Parent {
		if m.ValueOf(c.Parent).Same(tested) {
			return c.Parent
		}
	}
	return nil
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
		return Field{Name: symbol.Name, Declarations: []*ast.Node{symbol.ValueDeclaration}, Symbol: symbol}
	}
	return Field{}
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
