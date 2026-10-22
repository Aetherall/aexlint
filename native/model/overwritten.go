package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Overwritten is a store into data that nothing reads: Store puts a value into a field of data (Field), into an
// element of a collection under a key, or appends it right after a variable is given the collection's size, which
// names its position (an append reserving a position: a placeholder). Keys are the origins of that key or position.
// Fills replace the value on every way from the store, an element through a key every origin of Keys flows to, through
// variables and the keys of a local container, with nothing able to read it before them, and one is enough: no read of
// the data as a whole or of data holding it, of the field, or of an element whose key shares an origin with Keys, and,
// unless no code outside the function can reach the data (see unreachable), no code the model does not see runs in
// between (see Unseen), nor a call to a project function that throws (see Throwing).
type Overwritten struct {
	Store Effect
	Field *Value
	Keys  []ValueSource
	Fills []Effect
}

// Reserves reports whether the store appends, reserving a position.
func (o Overwritten) Reserves() bool { return o.Store.Kind == Append }

// OverwrittenIn lists the overwritten stores of a function's body, outside the functions it declares.
func (m *Analyzer) OverwrittenIn(fn *ast.Node) []Overwritten {
	if found, ok := m.overwritten[fn]; ok {
		return found
	}
	found := m.overwrittenOf(fn)
	m.overwritten[fn] = found
	return found
}

func (m *Analyzer) overwrittenOf(fn *ast.Node) []Overwritten {
	if fn.Body() == nil {
		return nil
	}
	effects := m.EffectsIn([]*ast.Node{fn.Body()})
	var reads []Read
	found := []Overwritten{}
	for _, s := range effects {
		candidate, ok := m.storeOf(s)
		if !ok {
			continue
		}
		for _, w := range effects {
			if w.Node != s.Node && m.replaces(candidate, w) {
				candidate.Fills = append(candidate.Fills, w)
			}
		}
		if len(candidate.Fills) == 0 {
			continue
		}
		if reads == nil {
			reads = m.ReadsIn([]*ast.Node{fn.Body()})
		}
		candidate.Fills = slices.DeleteFunc(candidate.Fills, func(w Effect) bool { return m.seen(candidate, w, reads) || m.exposed(fn, candidate, w) })
		if len(candidate.Fills) == 0 {
			continue
		}
		found = append(found, candidate)
	}
	return found
}

// storeOf reads an effect as a store whose place later stores can name: an assignment of a field, a store under a
// key, or an append whose position a size read names.
func (m *Analyzer) storeOf(s Effect) (Overwritten, bool) {
	if !s.Receiver.Named() || s.Stored == nil {
		return Overwritten{}, false
	}
	switch s.Kind {
	case Append:
		if position := m.position(s); position != nil {
			return Overwritten{Store: s, Keys: []ValueSource{{Node: position}}}, true
		}
	case Store:
		if s.Key != nil {
			if keys := m.OriginsOf(s.Key); len(keys) > 0 {
				return Overwritten{Store: s, Keys: keys}, true
			}
		}
	case Change:
		if field, ok := m.fieldOf(s); ok {
			return Overwritten{Store: s, Field: &field}, true
		}
	}
	return Overwritten{}, false
}

// fieldOf returns the field a plain assignment effect sets: a declared property, not set through a setter, which runs
// code.
func (m *Analyzer) fieldOf(e Effect) (Value, bool) {
	if !ast.IsAssignmentExpression(e.Node, false) || e.Node.AsBinaryExpression().OperatorToken.Kind != ast.KindEqualsToken {
		return Value{}, false
	}
	target := Unwrap(e.Node.AsBinaryExpression().Left)
	if target.Kind != ast.KindPropertyAccessExpression && target.Kind != ast.KindElementAccessExpression || m.computed(target) {
		return Value{}, false
	}
	name := target
	if target.Kind == ast.KindPropertyAccessExpression {
		name = target.Name()
	}
	if m.ctx.TypeChecker.GetSymbolAtLocation(name) == nil {
		return Value{}, false
	}
	v := m.ValueOf(target)
	if !v.Named() || v.Partial || len(v.Path) == 0 {
		return Value{}, false
	}
	v.Node = nil
	return v, true
}

// replaces reports whether w stores into the place of candidate on every way from it.
func (m *Analyzer) replaces(candidate Overwritten, w Effect) bool {
	s := candidate.Store
	if candidate.Field != nil {
		field, ok := m.fieldOf(w)
		return ok && field.Same(*candidate.Field) && m.follows(s.Node, w.Node)
	}
	if w.Kind != Store || w.Key == nil || !w.Receiver.Same(s.Receiver) {
		return false
	}
	origins := m.OriginsOf(w.Key)
	if slices.ContainsFunc(candidate.Keys, func(k ValueSource) bool { return !slices.Contains(origins, k) }) {
		return false
	}
	if loop, over := m.keyLoop(w.Key); loop != nil {
		contents, ok := m.Contents(over, loop)
		ending := m.flowControl().of(loop.Statement())
		if !ok || !m.always(loop.Statement(), w.Node) || ending.breaks[loop] != nil || ending.returns || ending.throws {
			return false
		}
		return slices.ContainsFunc(contents.Entries, func(e Entry) bool {
			if e.Key == nil || !m.follows(s.Node, e.Node) || e.Node.End() > loop.Pos() {
				return false
			}
			keys := m.OriginsOf(e.Key)
			return !slices.ContainsFunc(candidate.Keys, func(k ValueSource) bool { return !slices.Contains(keys, k) })
		})
	}
	return m.follows(s.Node, w.Node)
}

// position returns the size read of an append's collection that names the appended element: the value a variable is
// given, by the last statement before the append in its statement list doing so, with no other append to the
// collection between them.
func (m *Analyzer) position(a Effect) *ast.Node {
	statement := statementOf(a.Node)
	if statement == nil {
		return nil
	}
	list := statementList(statement.Parent)
	i := slices.Index(list, statement)
	for j := i - 1; j >= 0; j-- {
		for _, w := range slices.Backward(writesIn(list[j:j+1], false)) {
			source := Unwrap(w.Source)
			if w.Kind == Assignment && source != nil && source.Kind == ast.KindPropertyAccessExpression && m.size(source) && m.ValueOf(source.Expression()).Same(a.Receiver) {
				return source
			}
		}
		if d := declared(list[j]); d != nil {
			if source := Unwrap(d.Initializer()); source != nil && source.Kind == ast.KindPropertyAccessExpression && m.size(source) && m.ValueOf(source.Expression()).Same(a.Receiver) {
				return source
			}
		}
		if slices.ContainsFunc(m.EffectsIn(list[j:j+1]), func(e Effect) bool { return e.Kind == Append && e.Receiver.Same(a.Receiver) }) {
			return nil
		}
	}
	return nil
}

// declared returns the declaration of a variable statement declaring one variable.
func declared(statement *ast.Node) *ast.Node {
	if statement.Kind != ast.KindVariableStatement {
		return nil
	}
	declarations := statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes
	if len(declarations) != 1 {
		return nil
	}
	return declarations[0]
}

// statementOf returns the statement in a statement list holding n.
func statementOf(n *ast.Node) *ast.Node {
	for s := n; s != nil && s.Parent != nil; s = s.Parent {
		if ast.IsFunctionLike(s) {
			return nil
		}
		if slices.Contains(statementList(s.Parent), s) {
			return s
		}
	}
	return nil
}

// keyLoop returns the for-of loop binding key to the keys of the container it iterates, and that container.
func (m *Analyzer) keyLoop(key *ast.Node) (*ast.Node, Value) {
	key = Unwrap(key)
	if key.Kind != ast.KindIdentifier {
		return nil, Value{}
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(key)
	if symbol == nil || symbol.ValueDeclaration == nil || m.assigned(symbol) {
		return nil, Value{}
	}
	d := symbol.ValueDeclaration
	position := -1
	if d.Kind == ast.KindBindingElement && d.Parent.Kind == ast.KindArrayBindingPattern {
		position = slices.Index(d.Parent.AsBindingPattern().Elements.Nodes, d)
		d = d.Parent.Parent
	}
	if d.Kind != ast.KindVariableDeclaration || d.Parent == nil || d.Parent.Parent == nil || d.Parent.Parent.Kind != ast.KindForOfStatement {
		return nil, Value{}
	}
	loop := d.Parent.Parent
	over, part := m.iteration(loop.AsForInOrOfStatement().Expression)
	if part == keyPart && position < 0 || part == entryParts && position == 0 {
		return loop, over
	}
	return nil, Value{}
}

// always reports whether every way through body from its start reaches n: on the way from n up to body, n is never in
// a branch of an if, switch or conditional expression, on the right of a logical operator, after an optional chain,
// in a loop's body, condition or update, in a catch clause or in a function, and the statements before it in each
// statement list only reach their end.
func (m *Analyzer) always(body, n *ast.Node) bool {
	for c := n; c != body; c = c.Parent {
		p := c.Parent
		if p == nil || ast.IsFunctionLike(p) || ast.IsClassLike(p) {
			return false
		}
		if list := statementList(p); slices.Contains(list, c) {
			if p.Kind == ast.KindCaseClause || p.Kind == ast.KindDefaultClause || !m.straight(list[:slices.Index(list, c)]) {
				return false
			}
			continue
		}
		switch p.Kind {
		case ast.KindIfStatement, ast.KindSwitchStatement:
			if c != p.Expression() {
				return false
			}
		case ast.KindConditionalExpression:
			if c != p.AsConditionalExpression().Condition {
				return false
			}
		case ast.KindBinaryExpression:
			if c == p.AsBinaryExpression().Right && conditional(c) {
				return false
			}
		case ast.KindForStatement:
			if c != p.AsForStatement().Initializer {
				return false
			}
		case ast.KindForInStatement, ast.KindForOfStatement:
			if c != p.AsForInOrOfStatement().Expression {
				return false
			}
		case ast.KindWhileStatement, ast.KindDoStatement:
			return false
		case ast.KindTryStatement:
			if c != p.AsTryStatement().TryBlock && c != p.AsTryStatement().FinallyBlock {
				return false
			}
		case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression, ast.KindCallExpression:
			if c != p.Expression() && ast.IsOptionalChain(p) {
				return false
			}
		}
	}
	return true
}

// follows reports whether every way from a reaches b, where b comes after a. Going out from a to the statement list
// holding both, the statements after each statement holding a only reach their end, a loop on the way neither returns
// nor throws, and a's list is not a case clause or a try block; from there, the statements before the one holding b
// only reach their end and every way through that one reaches b.
func (m *Analyzer) follows(a, b *ast.Node) bool {
	common := commonList(a, b)
	if common == nil || a.End() > b.Pos() {
		return false
	}
	for s := statementOf(a); s != nil; {
		holder := s.Parent
		list := statementList(holder)
		i := slices.Index(list, s)
		if holder == common {
			j := slices.IndexFunc(list, func(t *ast.Node) bool { return encloses(t, b) })
			if j == i {
				return !ast.IsIterationStatement(s, false) && m.always(s, b)
			}
			return j > i && m.straight(list[i+1:j]) && m.always(list[j], b)
		}
		if !m.straight(list[i+1:]) || m.leavesFrom(holder) {
			return false
		}
		next := statementOf(holder)
		if next == nil {
			return false
		}
		for p := holder; p != nil && p != next.Parent; p = p.Parent {
			if ast.IsIterationStatement(p, false) {
				if e := m.flowControl().of(p.Statement()); e.returns || e.throws {
					return false
				}
			}
		}
		s = next
	}
	return false
}

// leavesFrom reports whether the statement holding a statement list is a decision or try whose end is reached on some
// way without running that list to its end: only blocks of if, switch, try and plain blocks are followed.
func (m *Analyzer) leavesFrom(parent *ast.Node) bool {
	switch parent.Kind {
	case ast.KindBlock:
		return parent.Parent != nil && parent.Parent.Kind == ast.KindTryStatement && parent.Parent.AsTryStatement().TryBlock == parent
	case ast.KindCaseClause, ast.KindDefaultClause:
		return true
	}
	return false
}

// straight reports whether every way through statements reaches their end.
func (m *Analyzer) straight(statements []*ast.Node) bool {
	e := m.listEnding(statements)
	return e.end != nil && len(e.breaks) == 0 && len(e.continues) == 0 && !e.returns && !e.throws
}

// commonList returns the innermost statement list holding both a and b.
func commonList(a, b *ast.Node) *ast.Node {
	for s := statementOf(a); s != nil; s = statementOf(s.Parent) {
		if encloses(s.Parent, b) {
			return s.Parent
		}
	}
	return nil
}

// exposed reports whether code the model does not see may read the stored value before the fill w, or a caller may
// once a call between them throws: the data is reachable from outside fn up to w (see unreachable), and such code or
// such a call runs between them, in the region seen reads.
func (m *Analyzer) exposed(fn *ast.Node, candidate Overwritten, w Effect) bool {
	data := candidate.Store.Receiver
	if candidate.Field != nil {
		data = *candidate.Field
	}
	if m.unreachable(data, fn, w.Node.End()) {
		return false
	}
	nodes, start, end := m.region(candidate.Store, w)
	in := func(n *ast.Node) bool { return n.Pos() >= start && n.End() <= end }
	return slices.ContainsFunc(m.Unseen(nodes, false), in) || slices.ContainsFunc(m.Throwing(nodes), in)
}

// region returns the statements from the one holding the store s, in the statement list holding both, to the one
// holding the fill w, and the positions between which code runs after s and before w writes: the whole statement
// holding s when it is a loop, whose later iterations run after it.
func (m *Analyzer) region(s, w Effect) ([]*ast.Node, int, int) {
	common := commonList(s.Node, w.Node)
	if common == nil {
		return nil, 0, 0
	}
	list := statementList(common)
	from := slices.IndexFunc(list, func(t *ast.Node) bool { return encloses(t, s.Node) })
	to := slices.IndexFunc(list, func(t *ast.Node) bool { return encloses(t, w.Node) })
	start, end := list[from].Pos(), list[to].End()
	if !ast.IsIterationStatement(list[from], false) && from != to {
		start = s.Node.End()
	}
	return list[from : to+1], start, end
}

// seen reports whether a read between the store and the fill w may see the stored value: from the statement holding
// the store in the statement list holding both, to the statement holding w, a read of the field, of the collection as a
// whole, of data holding either, or of an element whose key shares an origin with the store's.
func (m *Analyzer) seen(candidate Overwritten, w Effect, reads []Read) bool {
	s := candidate.Store
	nodes, start, end := m.region(s, w)
	if nodes == nil {
		return true
	}
	return slices.ContainsFunc(reads, func(r Read) bool {
		if r.Node.Pos() < start || r.Node.End() > end {
			return false
		}
		if candidate.Field != nil {
			return r.Value.Within(*candidate.Field) || candidate.Field.Within(r.Value)
		}
		collection := s.Receiver
		switch {
		case r.Size && r.Value.Same(collection):
			return false
		case r.Value.Same(collection) && r.Key != nil:
			origins := m.OriginsOf(r.Key)
			return slices.ContainsFunc(candidate.Keys, func(k ValueSource) bool { return slices.Contains(origins, k) })
		case r.Value.Within(collection) && !r.Value.Same(collection):
			return false
		}
		return collection.Within(r.Value)
	})
}

// Lost reports whether an overwritten store is lost whole: it sets a field, an element of an array, or an entry of a
// local container (see Contents) that its function only reads by key, where the order its keys come in cannot show. A
// placeholder still grows its collection, and a store into another map, set or object may still fix where its key
// comes in iteration order.
func (m *Analyzer) Lost(o Overwritten) bool {
	if o.Field != nil {
		return true
	}
	if o.Store.Kind != Store {
		return false
	}
	if o.Store.Node.Kind == ast.KindBinaryExpression && m.array(o.Store.Node.AsBinaryExpression().Left) {
		return true
	}
	fn := enclosingFunction(o.Store.Node)
	if _, ok := m.Contents(o.Store.Receiver, o.Store.Node); !ok || fn == nil {
		return false
	}
	container := o.Store.Receiver
	return !slices.ContainsFunc(m.ReadsIn([]*ast.Node{fn.Body()}), func(r Read) bool {
		return r.Value.Within(container) && r.Key == nil && !r.Size && m.checked(r) == nil
	})
}

// array reports whether an element access reads an array or tuple.
func (m *Analyzer) array(n *ast.Node) bool {
	n = Unwrap(n)
	if n.Kind != ast.KindElementAccessExpression {
		return false
	}
	tc := m.ctx.TypeChecker
	t := tc.GetNonNullableType(tc.GetTypeAtLocation(n.Expression()))
	return t != nil && (checker.Checker_isArrayType(tc, t) || checker.IsTupleType(t))
}
