package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// Buffer is a local container that only defers its entries to one for-of loop over it, or one forEach call on it, its
// drain: its contents are
// known (see Contents), every entry is put into it before the drain, which runs once per container, and the drain is
// its only read as a whole. Before the drain, the function only looks its entries up by key, checks whether it holds
// a key, and removes entries by key; it never reads its size, puts an entry before others, reorders it or removes all
// of it, and nothing reads or changes it once the drain starts, except the drain reading or changing the entry it is
// given. Entries are read by the origins of their keys (see
// BufferedEntry).
type Buffer struct {
	Contents Contents
	Drain    *ast.Node
	Entries  []BufferedEntry
}

// BufferedEntry is an entry of a buffer stored under keys from Keys, nil for an append. It is held when a lookup,
// check or removal by key shares an origin with Keys, and an append is held when any element of the container is
// read by key. Otherwise it is settled when every way from the store reaches the drain, no project call between them
// may throw, nothing but the drain reaches
// what it stores once stored (see settled), and, in a map, set or object, no other entry may replace it: the store
// runs once for each container, or its key is new each time it runs, a new object or a position reserved by
// appending to the collection whose size it reads (keys from different origins are taken to differ, as for
// unaddressed keys). Handing a settled entry over where it is stored would give the drain the same value. Ordered
// lists, for a settled entry, the reads and effects between the store and the drain that the deferral keeps in order
// with the drain's effects and reads: a read of data the drain changes, or an effect on data the drain reads or
// changes, unless both act on one collection under different literal keys, or any effect on data that outlives the
// function (see unreachable) when the drain may throw, which would stop the code after the store early. Exposed lists the code the model does not
// see (see Unseen), an await included, on one side, between the store and the drain or in the drain, while the other
// reads or changes data other than the buffer such code may reach (see reachable): the deferral keeps it apart from
// that data too.
type BufferedEntry struct {
	Entry   Entry
	Keys    []ValueSource
	Settled bool
	Ordered []*ast.Node
	Exposed []*ast.Node
}

// Buffers lists the buffers of a function.
func (m *Analyzer) Buffers(fn *ast.Node) []Buffer {
	if found, ok := m.buffers[fn]; ok {
		return found
	}
	found := []Buffer{}
	m.buffers[fn] = found
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
		drain := m.drainOf(contents, fn, reads, effects)
		if drain == nil {
			continue
		}
		b := Buffer{Contents: contents, Drain: drain}
		addressed := m.addressed(contents.Container, reads, effects)
		set := m.typeIs(contents.Container.Root.ValueDeclaration.Name(), "Set")
		for _, entry := range contents.Entries {
			key := entry.Key
			if set {
				key = entry.Value
			}
			if key == nil {
				held := len(addressed) > 0
				b.Entries = append(b.Entries, m.buffered(fn, b, entry, nil, held))
				continue
			}
			origins := m.OriginsOf(key)
			taken := slices.DeleteFunc(slices.Clone(origins), func(o ValueSource) bool { return !slices.Contains(addressed, o) })
			free := slices.DeleteFunc(slices.Clone(origins), func(o ValueSource) bool { return slices.Contains(addressed, o) })
			if len(taken) > 0 {
				b.Entries = append(b.Entries, BufferedEntry{Entry: entry, Keys: taken})
			}
			if len(free) > 0 {
				b.Entries = append(b.Entries, m.buffered(fn, b, entry, free, false))
			}
		}
		found = append(found, b)
	}
	m.buffers[fn] = found
	return found
}

// Deferred returns the buffer an unaddressed key's entry is settled in, under that key's origin.
func (m *Analyzer) Deferred(k UnaddressedKey) (Buffer, BufferedEntry, bool) {
	fn := enclosingFunction(k.Entry.Node)
	if fn == nil {
		return Buffer{}, BufferedEntry{}, false
	}
	for _, b := range m.Buffers(fn) {
		for _, e := range b.Entries {
			if e.Settled && e.Entry.Node == k.Entry.Node && slices.Contains(e.Keys, k.Origin) {
				return b, e, true
			}
		}
	}
	return Buffer{}, BufferedEntry{}, false
}

func (m *Analyzer) buffered(fn *ast.Node, b Buffer, entry Entry, keys []ValueSource, held bool) BufferedEntry {
	buffered := BufferedEntry{Entry: entry, Keys: keys}
	if held || !m.follows(entry.Node, b.Drain) || m.interrupted(entry.Node, b.Drain) || !m.unique(b, entry, keys) {
		return buffered
	}
	settled, own := m.settled(fn, b, entry, keys)
	if !settled {
		return buffered
	}
	buffered.Settled = true
	buffered.Ordered = m.ordered(b, entry)
	buffered.Exposed = m.exposure(fn, b, entry, own)
	return buffered
}

// interrupted reports whether a call between a store and the drain may throw (see Throwing), which drops the entry
// with the buffer while handing it over at the store would not.
func (m *Analyzer) interrupted(store, drain *ast.Node) bool {
	nodes, start, end := m.afterStore(store, drain)
	return slices.ContainsFunc(m.Throwing(nodes), func(n *ast.Node) bool { return n.Pos() >= start && n.End() <= end })
}

// unique reports whether no other entry of a map, set or object may replace an entry stored under keys: the store
// runs once for each container, or each of keys is a new object or a reserved position (see reserved).
func (m *Analyzer) unique(b Buffer, entry Entry, keys []ValueSource) bool {
	if keys == nil {
		return true
	}
	declaration := b.Contents.Container.Root.ValueDeclaration
	repeated := false
	for p := entry.Node.Parent; p != nil && !encloses(p, declaration); p = p.Parent {
		repeated = repeated || ast.IsIterationStatement(p, false)
	}
	return !repeated || !slices.ContainsFunc(keys, func(k ValueSource) bool {
		return k.Node == nil || k.Path != "" || m.ProvenanceOf(k.Node) != FreshValue && !m.reserved(k.Node, entry.Node)
	})
}

// reserved reports whether key is a size read of a collection that an append to it, on every way from the read to the
// store, makes the position of the appended element.
func (m *Analyzer) reserved(key, store *ast.Node) bool {
	key = Unwrap(key)
	if key.Kind != ast.KindPropertyAccessExpression || !m.size(key) {
		return false
	}
	collection := m.ValueOf(key.Expression())
	fn := enclosingFunction(key)
	return fn != nil && slices.ContainsFunc(m.EffectsIn([]*ast.Node{fn.Body()}), func(e Effect) bool {
		return e.Kind == Append && e.Receiver.Same(collection) && m.follows(key, e.Node) && m.follows(e.Node, store)
	})
}

// DrainParts returns what a buffer's drain iterates, the code it runs for each entry, and the variables binding the
// entry there.
func DrainParts(drain *ast.Node) (*ast.Node, *ast.Node, []*ast.Symbol) {
	if drain.Kind == ast.KindForOfStatement {
		loop := drain.AsForInOrOfStatement()
		return loop.Expression, loop.Statement, declaredBy(loop.Initializer)
	}
	callback := Unwrap(drain.Arguments()[0])
	given := []*ast.Symbol{}
	for _, parameter := range callback.Parameters() {
		given = append(given, declaredBy(parameter)...)
	}
	return Unwrap(drain.Expression()).Expression(), callback.Body(), given
}

// drains reports whether n is a for-of loop over container, or a forEach call on it, or on its entries, keys or
// values, given a callback.
func (m *Analyzer) drains(n *ast.Node, container Value) bool {
	switch n.Kind {
	case ast.KindForOfStatement:
		over, _ := m.iteration(n.AsForInOrOfStatement().Expression)
		return over.Same(container)
	case ast.KindCallExpression:
		callee := Unwrap(n.Expression())
		if callee.Kind != ast.KindPropertyAccessExpression || len(n.Arguments()) == 0 || !ast.IsFunctionLike(Unwrap(n.Arguments()[0])) {
			return false
		}
		_, name, ok := m.libraryMember(callee.Name())
		over, _ := m.iteration(callee.Expression())
		return ok && name == "forEach" && over.Same(container)
	}
	return false
}

// drainOf returns the drain of a local container, when it is a buffer.
func (m *Analyzer) drainOf(contents Contents, fn *ast.Node, reads []Read, effects []Effect) *ast.Node {
	container := contents.Container
	var drain *ast.Node
	several := false
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if m.drains(c, container) {
			several = several || drain != nil
			drain = c
		}
		c.ForEachChild(walk)
		return false
	}
	walk(fn.Body())
	if drain == nil || several {
		return nil
	}
	declaration := container.Root.ValueDeclaration
	for p := drain.Parent; p != nil && p != fn; p = p.Parent {
		if ast.IsIterationStatement(p, false) && !encloses(p, declaration) {
			return nil
		}
	}
	if slices.ContainsFunc(contents.Entries, func(e Entry) bool { return e.Node.End() > drain.Pos() }) {
		return nil
	}
	iterated, _, _ := DrainParts(drain)
	element := func(v Value) bool { return len(v.Path) > len(container.Path) && v.Path[len(container.Path)] == Element }
	for _, r := range reads {
		if !r.Value.Within(container) || encloses(iterated, r.Node) || r.Node == drain || encloses(drain, r.Node) && element(r.Value) {
			continue
		}
		if r.Node.End() > drain.Pos() || !r.Value.Same(container) || r.Size || r.Key == nil && m.checked(r) == nil {
			return nil
		}
	}
	for _, e := range effects {
		if !e.Receiver.Within(container) || encloses(drain, e.Node) && element(e.Receiver) {
			continue
		}
		switch {
		case e.Node.End() > drain.Pos():
			return nil
		case e.Kind == Append && m.prepends(e):
			return nil
		case e.Kind == Store || e.Kind == Append:
		case e.Kind == Remove && (e.Key != nil || e.Stored != nil):
		default:
			return nil
		}
	}
	return drain
}

// prepends reports whether an append puts its value before the others: Array.unshift.
func (m *Analyzer) prepends(e Effect) bool {
	if e.Node.Kind != ast.KindCallExpression {
		return false
	}
	callee := Unwrap(e.Node.Expression())
	if callee.Kind != ast.KindPropertyAccessExpression {
		return false
	}
	_, name, ok := m.libraryMember(callee.Name())
	return ok && name == "unshift"
}

// settled reports whether nothing but the drain reaches what an entry stores once stored: a primitive; or a value
// that, when the stored expression is a variable, the function does not reference between the store and the drain,
// before the store in a loop running again after it unless the variable is declared in that loop, in the functions it
// declares, or before the store other than as contained allows; and that is new or constant as defined with the key
// (see paired), or else is not exposed to any effect on data other than the container, or code the model does not
// see, between the store and the drain. Own reports whether the value is the entry's own: a primitive, or new or
// constant.
func (m *Analyzer) settled(fn *ast.Node, b Buffer, entry Entry, keys []ValueSource) (settled, own bool) {
	value := Unwrap(entry.Value)
	if m.primitive(value) {
		return true, true
	}
	var kinds Provenance
	if value.Kind == ast.KindIdentifier {
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(value)
		if symbol == nil || symbol.ValueDeclaration == nil || enclosingFunction(symbol.ValueDeclaration) != fn || !m.unaliased(symbol, fn, entry, b.Drain) {
			return false, false
		}
		for _, d := range m.paired(symbol, entry, keys) {
			kinds |= m.DefProvenance(d)
		}
	} else {
		kinds = m.ProvenanceOf(value)
	}
	if kinds&^(FreshValue|ConstantValue) == 0 {
		return true, true
	}
	nodes, start, end := m.afterStore(entry.Node, b.Drain)
	in := func(n *ast.Node) bool { return n.Pos() >= start && n.End() <= end }
	if slices.ContainsFunc(m.Unseen(nodes, false), in) {
		return false, false
	}
	return !slices.ContainsFunc(m.EffectsIn(nodes), func(e Effect) bool { return in(e.Node) && !e.Receiver.Within(b.Contents.Container) }), false
}

// unaliased reports whether the variable symbol, stored by entry, is referenced as settled allows.
func (m *Analyzer) unaliased(symbol *ast.Symbol, fn *ast.Node, entry Entry, drain *ast.Node) bool {
	declaration := symbol.ValueDeclaration
	store := entry.Node
	stored := Unwrap(entry.Value)
	unaliased := true
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if !unaliased {
			return true
		}
		if c.Kind == ast.KindIdentifier && c != stored && c.Parent != nil && m.referenced(c) == symbol {
			unaliased = m.unaliasedAt(c, fn, store, drain, declaration)
			return !unaliased
		}
		c.ForEachChild(walk)
		return !unaliased
	}
	walk(fn.Body())
	return unaliased
}

// referenced returns the variable an identifier reads, or nil when it names a declaration or property.
func (m *Analyzer) referenced(c *ast.Node) *ast.Symbol {
	switch p := c.Parent; {
	case p.Kind == ast.KindShorthandPropertyAssignment && p.Name() == c:
		return m.ctx.TypeChecker.GetShorthandAssignmentValueSymbol(p)
	case p.Name() == c:
		return nil
	}
	return m.ctx.TypeChecker.GetSymbolAtLocation(c)
}

func (m *Analyzer) unaliasedAt(c, fn, store, drain, declaration *ast.Node) bool {
	switch {
	case !m.runsIn(c, fn):
		return false
	case c.Pos() >= store.End():
		return c.Pos() >= drain.Pos()
	case c.Pos() >= store.Pos():
		return true
	}
	for p := store.Parent; p != nil && p != fn; p = p.Parent {
		if ast.IsIterationStatement(p, false) && encloses(p, c) && !encloses(p, declaration) {
			return false
		}
	}
	if p := c.Parent; ast.IsAssignmentExpression(p, true) && Unwrap(p.AsBinaryExpression().Left) == c {
		return true
	}
	return m.contained(c)
}

// paired returns the definitions of the variable symbol an entry stores that reach the store with its key from keys:
// when the key is a variable, the definitions of the value at or after a definition of the key giving one of keys,
// when one of them follows it on every way; all definitions reaching the store otherwise.
func (m *Analyzer) paired(symbol *ast.Symbol, entry Entry, keys []ValueSource) []Def {
	defs := m.reaching(entry.Node, Value{Root: symbol, Path: []string{}})
	key := Unwrap(entry.Key)
	if key == nil || key.Kind != ast.KindIdentifier || keys == nil {
		return defs
	}
	keySymbol := m.ctx.TypeChecker.GetSymbolAtLocation(key)
	if keySymbol == nil {
		return defs
	}
	found := []Def{}
	for _, k := range m.reaching(entry.Node, Value{Root: keySymbol, Path: []string{}}) {
		node, source := defNode(k)
		if source == nil || !slices.ContainsFunc(m.OriginsOf(source), func(o ValueSource) bool { return slices.Contains(keys, o) }) {
			continue
		}
		if node.End() > entry.Node.Pos() {
			return defs
		}
		later := slices.DeleteFunc(slices.Clone(defs), func(d Def) bool {
			n, _ := defNode(d)
			return n == nil || n.Pos() < node.End()
		})
		if !slices.ContainsFunc(later, func(d Def) bool { n, _ := defNode(d); return m.follows(node, n) }) {
			return defs
		}
		for _, d := range later {
			if !slices.Contains(found, d) {
				found = append(found, d)
			}
		}
	}
	if len(found) == 0 {
		return defs
	}
	return found
}

// defNode returns the declaration or write of a definition and the expression giving its value.
func defNode(d Def) (*ast.Node, *ast.Node) {
	switch {
	case d.Before:
		return nil, nil
	case d.Declaration != nil:
		return d.Declaration, d.Declaration.Initializer()
	}
	return d.Write.Node, d.Write.Source
}

// afterStore returns the statements running after a store and before the drain, with the positions between which
// they do: from the store, or from the start of the outermost loop holding it below the statement list holding both,
// whose later iterations run after it.
func (m *Analyzer) afterStore(store, drain *ast.Node) ([]*ast.Node, int, int) {
	common := commonList(store, drain)
	if common == nil {
		return nil, 0, 0
	}
	list := statementList(common)
	from := slices.IndexFunc(list, func(t *ast.Node) bool { return encloses(t, store) })
	to := slices.IndexFunc(list, func(t *ast.Node) bool { return encloses(t, drain) })
	start := store.End()
	for p := store.Parent; p != nil && p != common; p = p.Parent {
		if ast.IsIterationStatement(p, false) {
			start = p.Pos()
		}
	}
	return list[from : to+1], start, drain.Pos()
}

// ordered lists the reads and effects a settled entry's deferral keeps in order with the drain.
func (m *Analyzer) ordered(b Buffer, entry Entry) []*ast.Node {
	nodes, start, end := m.afterStore(entry.Node, b.Drain)
	in := func(n *ast.Node) bool { return n.Pos() >= start && n.End() <= end }
	container := b.Contents.Container
	reads := slices.DeleteFunc(m.ReadsIn(nodes), func(r Read) bool { return !in(r.Node) || r.Value.Within(container) })
	effects := slices.DeleteFunc(m.EffectsIn(nodes), func(e Effect) bool { return !in(e.Node) || e.Receiver.Within(container) })
	_, each, _ := DrainParts(b.Drain)
	body := []*ast.Node{each}
	drainEffects := m.EffectsIn(body)
	found := []*ast.Node{}
	add := func(n *ast.Node) {
		if !slices.Contains(found, n) {
			found = append(found, n)
		}
	}
	for _, e := range drainEffects {
		for _, r := range reads {
			if m.conflicts(e.Receiver, e.Key, r.Value, r.Key) {
				add(r.Node)
			}
		}
	}
	throws := m.flowControl().of(each).throws || len(m.Throwing(body)) > 0
	fn := enclosingFunction(b.Drain)
	for _, e := range effects {
		if throws && !m.unreachable(e.Receiver, fn, b.Drain.End()) {
			add(e.Node)
		}
		for _, r := range m.ReadsIn(body) {
			if m.conflicts(e.Receiver, e.Key, r.Value, r.Key) {
				add(e.Node)
			}
		}
		for _, d := range drainEffects {
			if m.conflicts(e.Receiver, e.Key, d.Receiver, d.Key) {
				add(e.Node)
			}
		}
	}
	slices.SortFunc(found, func(a, b *ast.Node) int { return a.Pos() - b.Pos() })
	return found
}

// exposure lists the code the model does not see that a settled entry's deferral keeps apart from data it may reach on
// the other side. The entry the drain is given is its own when own is set.
func (m *Analyzer) exposure(fn *ast.Node, b Buffer, entry Entry, own bool) []*ast.Node {
	nodes, start, end := m.afterStore(entry.Node, b.Drain)
	_, each, given := DrainParts(b.Drain)
	inBuffer := func(v Value) bool { return v.Within(b.Contents.Container) }
	fill := Side{Nodes: nodes, In: func(n *ast.Node) bool { return n.Pos() >= start && n.End() <= end }, Hidden: inBuffer}
	drain := Side{Nodes: []*ast.Node{each}, Hidden: func(v Value) bool { return inBuffer(v) || own && slices.Contains(given, v.Root) }}
	return m.Exposure(fn, b.Drain.End(), fill, drain)
}

// Side is code on one side of a change of order: Nodes, the part of them In says runs there (all when In is nil), and
// the data Hidden says code the model does not see cannot reach there anyway.
type Side struct {
	Nodes  []*ast.Node
	In     func(*ast.Node) bool
	Hidden func(Value) bool
}

// Exposure lists the code the model does not see (see Unseen) on one side while the other reads or changes data such
// code may reach (see reachable), up to to: moving one side across the other could change what that code sees.
func (m *Analyzer) Exposure(fn *ast.Node, to int, a, b Side) []*ast.Node {
	in := func(s Side, n *ast.Node) bool { return s.In == nil || s.In(n) }
	reaches := func(s Side) bool {
		reachable := func(v Value, n *ast.Node) bool {
			return (s.Hidden == nil || !s.Hidden(v)) && m.reachable(fn, v, n, to)
		}
		return slices.ContainsFunc(m.ReadsIn(s.Nodes), func(r Read) bool { return in(s, r.Node) && reachable(r.Value, r.Node) }) ||
			slices.ContainsFunc(m.EffectsIn(s.Nodes), func(e Effect) bool { return in(s, e.Node) && reachable(e.Receiver, nil) })
	}
	unseen := func(s Side) []*ast.Node {
		return slices.DeleteFunc(m.Unseen(s.Nodes, false), func(n *ast.Node) bool { return !in(s, n) })
	}
	found := []*ast.Node{}
	if reaches(a) {
		found = append(found, unseen(b)...)
	}
	if reaches(b) {
		found = append(found, unseen(a)...)
	}
	slices.SortFunc(found, func(x, y *ast.Node) int { return x.Pos() - y.Pos() })
	return slices.Compact(found)
}

// reachable reports whether code the model does not see may reach data v, read at n or acted on when n is nil: it is
// not unreachable from outside fn up to to (see unreachable), nor a function or class declaration, nor a primitive value an expression
// gives, nor a variable of fn holding a primitive read at n that no function fn declares uses.
func (m *Analyzer) reachable(fn *ast.Node, v Value, n *ast.Node, to int) bool {
	if m.unreachable(v, fn, to) {
		return false
	}
	if len(v.Path) > 0 {
		return true
	}
	switch {
	case v.Base != nil:
		return !m.primitive(v.Base)
	case v.Root != nil && v.Root.Flags&(ast.SymbolFlagsFunction|ast.SymbolFlagsClass) != 0:
		return false
	case n == nil || v.Root == nil || v.Root.ValueDeclaration == nil || enclosingFunction(v.Root.ValueDeclaration) != fn:
		return true
	}
	return !m.primitive(n) || m.captured(v.Root, fn)
}

// captured reports whether a function fn declares references symbol.
func (m *Analyzer) captured(symbol *ast.Symbol, fn *ast.Node) bool {
	captured := false
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if captured {
			return true
		}
		if c.Kind == ast.KindIdentifier && c.Parent != nil && !m.runsIn(c, fn) && m.referenced(c) == symbol {
			captured = true
		}
		c.ForEachChild(walk)
		return captured
	}
	walk(fn.Body())
	return captured
}

// declaredBy returns the variables a declaration list, declaration or parameter declares, through binding patterns.
func declaredBy(n *ast.Node) []*ast.Symbol {
	found := []*ast.Symbol{}
	var name func(*ast.Node)
	name = func(c *ast.Node) {
		switch {
		case c == nil:
		case c.Kind == ast.KindIdentifier:
			if symbol := c.Parent.Symbol(); symbol != nil {
				found = append(found, symbol)
			}
		case ast.IsBindingPattern(c):
			for _, element := range c.AsBindingPattern().Elements.Nodes {
				if element.Kind == ast.KindBindingElement {
					name(element.Name())
				}
			}
		}
	}
	switch n.Kind {
	case ast.KindVariableDeclarationList:
		for _, d := range n.AsVariableDeclarationList().Declarations.Nodes {
			name(d.Name())
		}
	case ast.KindVariableDeclaration, ast.KindParameter:
		name(n.Name())
	}
	return found
}

// conflicts reports whether an effect on changed, under key when it acts on one element, may change data read or
// changed at other, under its key: one is data within the other, unless both are one collection and the keys are
// different literals.
func (m *Analyzer) conflicts(changed Value, key *ast.Node, other Value, otherKey *ast.Node) bool {
	if !changed.Named() || !other.Named() || !other.Within(changed) && !changed.Within(other) {
		return false
	}
	if other.Same(changed) && key != nil && otherKey != nil && m.unit(key) && m.unit(otherKey) {
		return m.ctx.TypeChecker.GetTypeAtLocation(key) == m.ctx.TypeChecker.GetTypeAtLocation(otherKey)
	}
	return true
}
