package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// UnaddressedKey is an origin of a key a local map stores entries under that no keyed access to the map shares, while
// keyed accesses do reach entries stored under other origins: Entry stores under a key with Origin among its origins, and no lookup, check
// or removal by key in the function has Origin among the origins of its key. Entries stored that way are only reached
// by going through the whole map, as a list would be.
type UnaddressedKey struct {
	Container Value
	Entry     Entry
	Origin    *ast.Node
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
		if e.Kind != Store || slices.ContainsFunc(seen, e.Receiver.Same) {
			continue
		}
		seen = append(seen, e.Receiver)
		contents, ok := m.Contents(e.Receiver, e.Node)
		if !ok || e.Receiver.Node == nil || !m.keyed(e.Receiver.Node) {
			continue
		}
		addressed := m.addressed(e.Receiver, reads, effects)
		keys := map[*ast.Node][]*ast.Node{}
		reached := false
		for _, entry := range contents.Entries {
			if entry.Key != nil {
				keys[entry.Node] = m.OriginsOf(entry.Key)
				reached = reached || slices.ContainsFunc(keys[entry.Node], func(o *ast.Node) bool { return slices.Contains(addressed, o) })
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
func (m *Analyzer) addressed(container Value, reads []Read, effects []Effect) []*ast.Node {
	found := []*ast.Node{}
	add := func(key *ast.Node) { found = unionNodes(found, m.OriginsOf(key)) }
	for _, r := range reads {
		if r.Value.Same(container) && r.Key != nil {
			add(r.Key)
		}
		if r.Value.Same(container) && r.Node.Kind == ast.KindCallExpression && len(r.Node.Arguments()) > 0 {
			if callee := Unwrap(r.Node.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
				if owners, name, ok := m.libraryMember(callee.Name()); ok && name == "has" && !slices.Contains(owners, "String") {
					add(r.Node.Arguments()[0])
				}
			}
		}
	}
	for _, e := range effects {
		if e.Kind == Remove && e.Key != nil && e.Receiver.Same(container) {
			add(e.Key)
		}
	}
	return found
}

// Fabrication is what a branch of a decision makes up only so the code after it can treat the branch's result like
// the other branches': placeholders appended in the branch that only the code after the decision replaces, and
// unaddressed keys first defined in the branch that the code after the decision stores entries under.
type Fabrication struct {
	Placeholders []Placeholder
	Keys         []UnaddressedKey
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
	for _, h := range m.Placeholders(fn) {
		if inBranch(h.Append.Node) && !slices.ContainsFunc(h.Fills, func(w Effect) bool { return w.Node.Pos() < decision.End() }) {
			fabrication.Placeholders = append(fabrication.Placeholders, h)
		}
	}
	for _, k := range m.UnaddressedKeys(fn) {
		if inBranch(k.Origin) && k.Entry.Node.Pos() >= decision.End() {
			fabrication.Keys = append(fabrication.Keys, k)
		}
	}
	return fabrication
}

// Fabricated reports whether a branch makes up anything.
func (f Fabrication) Fabricated() bool { return len(f.Placeholders) > 0 || len(f.Keys) > 0 }
