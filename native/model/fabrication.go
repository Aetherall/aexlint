package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// UnaddressedKey is an origin of a key a local map stores entries under that no keyed access to the map shares, while
// keyed accesses do reach entries stored under other origins. A map is a global Map, a set, whose values are its keys,
// or an object created empty and used only by element access: Entry stores under a key with Origin among its origins,
// and no lookup, check or removal by key in the function has Origin among the origins of its key. Entries stored that
// way are only reached by going through the whole map, as a list would be.
type UnaddressedKey struct {
	Container Value
	Entry     Entry
	Origin    ValueSource
}

// UnaddressedKeys lists the unaddressed keys of the local maps of a function.
func (m *Analyzer) UnaddressedKeys(fn *ast.Node) []UnaddressedKey {
	if found, ok := m.unaddressed[fn]; ok {
		return found
	}
	found := []UnaddressedKey{}
	m.unaddressed[fn] = found
	if fn.Body() == nil {
		return found
	}
	reads := m.ReadsIn([]*ast.Node{fn.Body()})
	effects := m.EffectsIn([]*ast.Node{fn.Body()})
	seen := []Value{}
	for _, e := range effects {
		if e.Kind != Store && e.Kind != Append || slices.ContainsFunc(seen, e.Receiver.Same) {
			continue
		}
		seen = append(seen, e.Receiver)
		contents, ok := m.Contents(e.Receiver, e.Node)
		if !ok {
			continue
		}
		declaration := e.Receiver.Root.ValueDeclaration
		set := m.typeIs(declaration.Name(), "Set")
		if !set && !m.keyed(declaration.Name()) && !emptyObject(declaration.Initializer()) {
			continue
		}
		addressed := m.addressed(e.Receiver, reads, effects)
		keys := map[*ast.Node][]ValueSource{}
		reached := false
		for _, entry := range contents.Entries {
			key := entry.Key
			if set {
				key = entry.Value
			}
			if key != nil {
				keys[entry.Node] = m.OriginsOf(key)
				reached = reached || slices.ContainsFunc(keys[entry.Node], func(o ValueSource) bool { return slices.Contains(addressed, o) })
			}
		}
		if !reached {
			continue
		}
		for _, entry := range contents.Entries {
			for _, origin := range keys[entry.Node] {
				if !slices.Contains(addressed, origin) {
					found = append(found, UnaddressedKey{Container: e.Receiver, Entry: entry, Origin: origin})
				}
			}
		}
	}
	m.unaddressed[fn] = found
	return found
}

// addressed returns the origins of the keys a function reaches entries of a container by: element reads and lookups by
// key, removals by key, and checks of whether it holds a key.
func (m *Analyzer) addressed(container Value, reads []Read, effects []Effect) []ValueSource {
	found := []ValueSource{}
	add := func(key *ast.Node) { found = unionSources(found, m.OriginsOf(key)) }
	for _, r := range reads {
		if r.Value.Same(container) && r.Key != nil {
			add(r.Key)
		}
		if key := m.checked(r); r.Value.Same(container) && key != nil {
			add(key)
		}
	}
	for _, e := range effects {
		if e.Kind == Remove && e.Receiver.Same(container) {
			if e.Key != nil {
				add(e.Key)
			} else if e.Stored != nil {
				add(e.Stored)
			}
		}
	}
	return found
}

// checked returns the key a read checks its collection holds, when it is a call to a global collection's has.
func (m *Analyzer) checked(r Read) *ast.Node {
	if r.Node.Kind != ast.KindCallExpression || len(r.Node.Arguments()) == 0 {
		return nil
	}
	if callee := Unwrap(r.Node.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
		if owners, name, ok := m.libraryMember(callee.Name()); ok && name == "has" && !slices.Contains(owners, "String") {
			return r.Node.Arguments()[0]
		}
	}
	return nil
}

// Fabrication is what a branch of a decision makes up only so the code after it can treat the branch's result like
// the other branches': stores the branch makes that only the code after the decision overwrites, such as a placeholder
// reserving a position, and unaddressed keys whose origin is in the branch that the code after the decision stores
// entries under.
type Fabrication struct {
	Stores []Overwritten
	Keys   []UnaddressedKey
}

// FabricationOf reads what the branches of a decision make up.
func (m *Analyzer) FabricationOf(decision *ast.Node) Fabrication {
	merge, ok := m.MergeOf(decision)
	fn := enclosingFunction(decision)
	if !ok || fn == nil {
		return Fabrication{}
	}
	inBranch := func(n *ast.Node) bool {
		return slices.ContainsFunc(merge.Arms, func(arm Arm) bool { return arm.Branch != nil && encloses(arm.Branch, n) })
	}
	fabrication := Fabrication{}
	for _, o := range m.OverwrittenIn(fn) {
		if inBranch(o.Store.Node) && !slices.ContainsFunc(o.Fills, func(w Effect) bool { return w.Node.Pos() < decision.End() }) {
			fabrication.Stores = append(fabrication.Stores, o)
		}
	}
	for _, k := range m.UnaddressedKeys(fn) {
		if k.Origin.Node != nil && inBranch(k.Origin.Node) && k.Entry.Node.Pos() >= decision.End() {
			fabrication.Keys = append(fabrication.Keys, k)
		}
	}
	return fabrication
}

// Fabricated reports whether a branch makes up anything.
func (f Fabrication) Fabricated() bool { return len(f.Stores) > 0 || len(f.Keys) > 0 }

// Fabricated reports whether an overwritten store is one a decision's branch makes up for the code after it (see
// FabricationOf).
func (m *Analyzer) Fabricated(o Overwritten) bool {
	fn := enclosingFunction(o.Store.Node)
	if fn == nil || fn.Body() == nil {
		return false
	}
	found := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if found || ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return found
		}
		if (n.Kind == ast.KindIfStatement || n.Kind == ast.KindSwitchStatement || n.Kind == ast.KindExpressionStatement) && encloses(n, o.Store.Node) {
			found = slices.ContainsFunc(m.FabricationOf(n).Stores, func(s Overwritten) bool { return s.Store.Node == o.Store.Node })
		}
		n.ForEachChild(walk)
		return found
	}
	fn.Body().ForEachChild(walk)
	return found
}
