package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// OwnKey is a method of a map's value given the key the value is stored under: Container is a local map (see Contents)
// whose every entry stores a new value, made by the function for the key it is stored under, Loop a for-of loop over
// its entries binding Key and Value, or a forEach callback taking Value and Key from a map, or the pair [Key, Value]
// from its entries, and Calls the calls in it of a project
// method on Value given Key. The value needs its own key to do its work, so every reader of the map has to pass the two
// together.
type OwnKey struct {
	Container Value
	Loop      *ast.Node
	Key       *ast.Symbol
	Value     *ast.Symbol
	Calls     []*ast.Node
}

// OwnKeyOf reads a for-of loop or a forEach call as one over a map whose values are given their own key.
func (m *Analyzer) OwnKeyOf(n *ast.Node) (OwnKey, bool) {
	var over Value
	var key, value *ast.Node
	var body *ast.Node
	switch n.Kind {
	case ast.KindForOfStatement:
		loop := n.AsForInOrOfStatement()
		container, part := m.iteration(loop.Expression)
		declarations := declaredBy(loop.Initializer)
		if part != entryParts || len(declarations) != 2 || loop.Initializer.Kind != ast.KindVariableDeclarationList {
			return OwnKey{}, false
		}
		pattern := loop.Initializer.AsVariableDeclarationList().Declarations.Nodes[0].Name()
		if pattern.Kind != ast.KindArrayBindingPattern || len(pattern.AsBindingPattern().Elements.Nodes) != 2 {
			return OwnKey{}, false
		}
		elements := pattern.AsBindingPattern().Elements.Nodes
		over, key, value, body = container, elements[0].Name(), elements[1].Name(), n.Statement()
	case ast.KindCallExpression:
		callee := Unwrap(n.Expression())
		if callee.Kind != ast.KindPropertyAccessExpression || len(n.Arguments()) == 0 {
			return OwnKey{}, false
		}
		if _, name, ok := m.libraryMember(callee.Name()); !ok || name != "forEach" {
			return OwnKey{}, false
		}
		callback := Unwrap(n.Arguments()[0])
		if !ast.IsFunctionLike(callback) || len(callback.Parameters()) == 0 {
			return OwnKey{}, false
		}
		container, part := m.iteration(callee.Expression())
		first := callback.Parameters()[0].Name()
		switch {
		case m.keyed(callee.Expression()) && len(callback.Parameters()) >= 2:
			over, value, key = m.ValueOf(callee.Expression()), first, callback.Parameters()[1].Name()
		case part == entryParts && first.Kind == ast.KindArrayBindingPattern && len(first.AsBindingPattern().Elements.Nodes) == 2:
			elements := first.AsBindingPattern().Elements.Nodes
			over, key, value = container, elements[0].Name(), elements[1].Name()
		default:
			return OwnKey{}, false
		}
		body = callback.Body()
	default:
		return OwnKey{}, false
	}
	if key == nil || value == nil || key.Kind != ast.KindIdentifier || value.Kind != ast.KindIdentifier || body == nil {
		return OwnKey{}, false
	}
	contents, ok := m.Contents(over, n)
	if !ok || len(contents.Entries) == 0 || !slices.ContainsFunc(contents.Entries, func(e Entry) bool { return e.Key != nil }) {
		return OwnKey{}, false
	}
	for _, e := range contents.Entries {
		if e.Key == nil || !m.madeFor(e) {
			return OwnKey{}, false
		}
	}
	found := OwnKey{Container: over, Loop: n, Key: key.Parent.Symbol(), Value: value.Parent.Symbol()}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if c.Kind == ast.KindCallExpression && m.givenOwnKey(c, found.Key, found.Value) {
			found.Calls = append(found.Calls, c)
		}
		c.ForEachChild(walk)
		return false
	}
	walk(body)
	return found, len(found.Calls) > 0 && found.Key != nil && found.Value != nil
}

// madeFor reports whether an entry stores a new value: every origin of the stored value is a new object, or the entry's
// own value read back through a lookup of the map.
func (m *Analyzer) madeFor(e Entry) bool {
	stored := Unwrap(e.Value)
	symbol := (*ast.Symbol)(nil)
	if stored.Kind == ast.KindIdentifier {
		symbol = m.ctx.TypeChecker.GetSymbolAtLocation(stored)
	}
	origins := m.OriginsOf(stored)
	return len(origins) > 0 && !slices.ContainsFunc(origins, func(o ValueSource) bool {
		if o.Node == nil || o.Path != "" {
			return true
		}
		if o.Node.Kind == ast.KindIdentifier && symbol != nil && m.ctx.TypeChecker.GetSymbolAtLocation(o.Node) == symbol {
			return false
		}
		return m.ProvenanceOf(o.Node) != FreshValue
	})
}

// givenOwnKey reports whether a call is of a project method on value given key as an argument.
func (m *Analyzer) givenOwnKey(call *ast.Node, key, value *ast.Symbol) bool {
	callee := Unwrap(call.Expression())
	if callee.Kind != ast.KindPropertyAccessExpression || m.followed(call) == nil {
		return false
	}
	receiver := Unwrap(callee.Expression())
	if receiver.Kind != ast.KindIdentifier || m.ctx.TypeChecker.GetSymbolAtLocation(receiver) != value {
		return false
	}
	return slices.ContainsFunc(call.Arguments(), func(a *ast.Node) bool {
		a = Unwrap(a)
		return a.Kind == ast.KindIdentifier && m.ctx.TypeChecker.GetSymbolAtLocation(a) == key
	})
}
