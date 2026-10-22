package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// Read is a place where code reads data: Value is the data read, with everything inside it, Key the key of an element
// read when one expression selects it, and Size whether only the size of a collection is read.
type Read struct {
	Node  *ast.Node
	Value Value
	Key   *ast.Node
	Size  bool
}

// ReadsIn lists the reads nodes make, including in the functions they declare, which may run. A variable or this read
// through property reads and literal-key element reads is one read of that data, except where it is:
//   - the target an assignment replaces, or the object of an element an assignment replaces;
//   - the object of an element read with a computed key, a read of that element;
//   - the object of a size read, a read of its size only;
//   - the receiver of a global collection method that stores, appends or removes without returning what it removes,
//     not a read; of a method returning an entry by key, a read of that element;
//   - the receiver, or an argument as a whole, of a call to a project function with a body, which reads what its
//     summary says, mapped onto the call; an argument its summary cannot name, given to a parameter it rebinds,
//     destructures or gathers, or after a spread, is read as a whole;
//   - the object of a read through a project getter, or of an assignment through a project setter, with a body,
//     which also reads what the accessor's summary says of its receiver.
func (m *Analyzer) ReadsIn(nodes []*ast.Node) []Read {
	reads := []Read{}
	targets := map[*ast.Node]bool{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		for _, w := range WritesAt(c) {
			if w.Kind == Assignment {
				targets[Unwrap(w.Target)] = true
			}
		}
		if fn := m.followed(c); fn != nil {
			reads = append(reads, m.readsApplied(c, fn)...)
		}
		if c.Kind == ast.KindPropertyAccessExpression {
			accessor := m.getter(c)
			if targets[c] {
				accessor = m.setter(c)
			}
			if accessor != nil {
				reads = append(reads, m.accessorReads(c, accessor)...)
			}
		}
		if c.Kind == ast.KindIdentifier || c.Kind == ast.KindThisKeyword {
			if read, ok := m.readAt(c, targets); ok {
				reads = append(reads, read)
			}
			return false
		}
		c.ForEachChild(walk)
		return false
	}
	for _, n := range nodes {
		if n != nil {
			walk(n)
		}
	}
	for i, r := range reads {
		if r.Value.Root != nil && r.Value.Root.ValueDeclaration != nil && r.Value.Root.ValueDeclaration.Kind == ast.KindParameter {
			callback := r.Value.Root.ValueDeclaration.Parent
			inside := slices.ContainsFunc(nodes, func(n *ast.Node) bool { return n != nil && n != callback.Body() && encloses(n, callback) })
			if inside && m.synchronous(callback) {
				reads[i].Value = m.throughElement(callback, r.Value)
			}
		}
		reads[i].Value = m.throughLoop(reads[i].Value)
	}
	return reads
}

// readAt reads the reference c as one read, or false when it is not a read of data or another read covers it.
func (m *Analyzer) readAt(c *ast.Node, targets map[*ast.Node]bool) (Read, bool) {
	if c.Kind == ast.KindIdentifier && (c.Parent == nil || c.Parent.Name() == c && c.Parent.Kind != ast.KindShorthandPropertyAssignment) {
		return Read{}, false
	}
	read := c
	for p := read.Parent; p != nil && (p.Kind == ast.KindPropertyAccessExpression || p.Kind == ast.KindElementAccessExpression && ast.IsStringOrNumericLiteralLike(Unwrap(p.AsElementAccessExpression().ArgumentExpression))) && p.Expression() == read; p = read.Parent {
		if call := p.Parent; p.Kind == ast.KindPropertyAccessExpression && call != nil && call.Kind == ast.KindCallExpression && call.Expression() == p {
			break
		}
		if m.size(p) && p.Kind == ast.KindPropertyAccessExpression {
			return Read{Node: p, Value: m.ValueOf(read), Size: true}, true
		}
		read = p
	}
	if targets[read] {
		return Read{}, false
	}
	value := m.ValueOf(read)
	whole := Read{Node: read, Value: value}
	p := read.Parent
	for p != nil && (p.Kind == ast.KindParenthesizedExpression || p.Kind == ast.KindNonNullExpression || p.Kind == ast.KindAsExpression) {
		read, p = p, p.Parent
	}
	if p == nil {
		return whole, true
	}
	switch {
	case p.Kind == ast.KindElementAccessExpression && p.Expression() == read:
		if targets[p] {
			return Read{}, false
		}
		return Read{Node: p, Value: value, Key: p.AsElementAccessExpression().ArgumentExpression}, true
	case p.Kind == ast.KindPropertyAccessExpression && p.Expression() == read && p.Parent != nil && p.Parent.Kind == ast.KindCallExpression && p.Parent.Expression() == p:
		call := p.Parent
		if m.followed(call) != nil {
			return Read{}, false
		}
		if owners, name, ok := m.libraryMember(p.Name()); ok {
			if effect, found := agreed(owners, name, libraryEffects); found && (effect.kind == Store || effect.kind == Append || name == "delete" || name == "clear") {
				return Read{}, false
			}
			if lookup, ok := m.LookupOf(call); ok && lookup.Key != nil {
				return Read{Node: call, Value: value, Key: lookup.Key}, true
			}
		}
		return Read{Node: call, Value: value}, true
	case p.Kind == ast.KindCallExpression && p.Expression() != read && slices.Contains(p.Arguments(), read):
		if m.followed(p) != nil {
			return Read{}, false
		}
	case p.Kind == ast.KindNewExpression && slices.Contains(p.Arguments(), read):
		if m.followed(p) != nil {
			return Read{}, false
		}
	}
	return whole, true
}

// accessorReads returns the reads running a project accessor makes of its object, from the accessor's summary.
func (m *Analyzer) accessorReads(access, accessor *ast.Node) []Read {
	object := Unwrap(access.Expression())
	receiver := m.ValueOf(object)
	if object.Kind == ast.KindSuperKeyword {
		receiver = Value{This: ast.GetThisContainer(access, false, false), Path: []string{}}
	}
	reads := []Read{}
	for _, r := range m.Summary(accessor).Reads {
		if r.Target.Parameter != -1 {
			continue
		}
		v := receiver
		v.Node, v.Path = nil, append(slices.Clone(v.Path), r.Target.Path...)
		reads = append(reads, Read{Node: access, Value: v, Size: r.Size})
	}
	return reads
}

// readsApplied returns the reads a call to fn makes, from its summary mapped onto the call as applied maps effects.
func (m *Analyzer) readsApplied(call, fn *ast.Node) []Read {
	var receiver *Value
	if call.Kind == ast.KindCallExpression {
		switch callee := Unwrap(call.Expression()); {
		case callee.Kind == ast.KindSuperKeyword:
			receiver = &Value{This: ast.GetThisContainer(call, false, false), Path: []string{}}
		case callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression:
			object := Unwrap(callee.Expression())
			v := m.ValueOf(object)
			if object.Kind == ast.KindSuperKeyword {
				v = Value{This: ast.GetThisContainer(call, false, false), Path: []string{}}
			}
			receiver = &v
		}
	}
	arguments := call.Arguments()
	if spread := slices.IndexFunc(arguments, func(a *ast.Node) bool { return a.Kind == ast.KindSpreadElement }); spread >= 0 {
		arguments = arguments[:spread]
	}
	given := func(a Anchor) (Value, *ast.Node, bool) {
		var v Value
		var node *ast.Node
		switch {
		case a.Parameter < 0 && receiver != nil:
			v = *receiver
		case a.Parameter >= 0 && a.Parameter < len(arguments):
			v, node = m.ValueOf(arguments[a.Parameter]), arguments[a.Parameter]
		default:
			return Value{}, nil, false
		}
		if len(a.Path) > 0 {
			node = nil
		}
		v.Node, v.Path = nil, append(slices.Clone(v.Path), a.Path...)
		return v, node, true
	}
	reads := []Read{}
	for _, r := range m.Summary(fn).Reads {
		v, _, ok := given(r.Target)
		if !ok {
			continue
		}
		read := Read{Node: call, Value: v, Size: r.Size}
		if r.Key != nil {
			_, read.Key, _ = given(*r.Key)
		}
		reads = append(reads, read)
	}
	parameters := fn.Parameters()
	for i, argument := range call.Arguments() {
		summarized := i < len(arguments) && i < len(parameters) && parameters[i].Name().Kind == ast.KindIdentifier &&
			parameters[i].AsParameterDeclaration().DotDotDotToken == nil && parameters[i].Symbol() != nil && !m.assigned(parameters[i].Symbol())
		if !summarized {
			reads = append(reads, Read{Node: argument, Value: m.ValueOf(argument)})
		}
	}
	return reads
}
