package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// Placeholder is an element appended to a collection only to reserve its position: Append appends a constant,
// Position reads the collection's size just before, naming the element, and Fills store into the collection at a key
// that position flows to, through variables and the keys of a local container. Every way from the append reaches a
// fill of its position, and nothing between them can read the element: no read of the collection as a whole or of
// data holding it, and no element read whose key the position flows to.
type Placeholder struct {
	Append   Effect
	Position *ast.Node
	Fills    []Effect
}

// Placeholders lists the placeholders of a function's body, outside the functions it declares.
func (m *Analyzer) Placeholders(fn *ast.Node) []Placeholder {
	if found, ok := m.placeholders[fn]; ok {
		return found
	}
	found := m.placeholdersOf(fn)
	m.placeholders[fn] = found
	return found
}

// Reserved lists the placeholders appended in a branch of a decision and filled only by the code after it: the
// branch makes up an element so that code can treat its result like the other branches'.
func (m *Analyzer) Reserved(decision *ast.Node) []Placeholder {
	merge, ok := m.MergeOf(decision)
	fn := enclosingFunction(decision)
	if !ok || fn == nil {
		return nil
	}
	return slices.DeleteFunc(slices.Clone(m.Placeholders(fn)), func(h Placeholder) bool {
		inBranch := slices.ContainsFunc(merge.Arms, func(arm Arm) bool { return arm.Branch != nil && encloses(arm.Branch, h.Append.Node) })
		return !inBranch || slices.ContainsFunc(h.Fills, func(w Effect) bool { return w.Node.Pos() < decision.End() })
	})
}

func (m *Analyzer) placeholdersOf(fn *ast.Node) []Placeholder {
	if fn.Body() == nil {
		return nil
	}
	effects := m.EffectsIn([]*ast.Node{fn.Body()})
	var reads []Read
	found := []Placeholder{}
	for _, a := range effects {
		if a.Kind != Append || a.Stored == nil || m.ProvenanceOf(a.Stored) != ConstantValue || !a.Receiver.Named() {
			continue
		}
		position := m.position(a)
		if position == nil {
			continue
		}
		fills := []Effect{}
		for _, w := range effects {
			if w.Kind == Store && w.Key != nil && w.Receiver.Same(a.Receiver) && slices.Contains(m.OriginsOf(w.Key), position) && m.fills(a, w, position) {
				fills = append(fills, w)
			}
		}
		if len(fills) == 0 {
			continue
		}
		if reads == nil {
			reads = m.ReadsIn([]*ast.Node{fn.Body()})
		}
		if slices.ContainsFunc(fills, func(w Effect) bool { return m.seen(a, w, position, reads) }) {
			continue
		}
		found = append(found, Placeholder{Append: a, Position: position, Fills: fills})
	}
	return found
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

// fills reports whether every way from the append a reaches the fill w of its position: directly, when w's key holds
// it through variables, or through a local container w's key is a key of, iterated by a for-of loop: every way from a
// reaches an entry stored under a key the position flows to, before the loop, and every iteration of that loop, which
// neither breaks out of it nor leaves the function, reaches w.
func (m *Analyzer) fills(a, w Effect, position *ast.Node) bool {
	if loop, over := m.keyLoop(w.Key); loop != nil {
		contents, ok := m.Contents(over, loop)
		ending := m.flowControl().of(loop.Statement())
		if !ok || !m.always(loop.Statement(), w.Node) || ending.breaks[loop] != nil || ending.returns || ending.throws {
			return false
		}
		return slices.ContainsFunc(contents.Entries, func(e Entry) bool {
			return e.Key != nil && slices.Contains(m.OriginsOf(e.Key), position) && m.follows(a.Node, e.Node) && e.Node.End() <= loop.Pos()
		})
	}
	return m.follows(a.Node, w.Node)
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

// seen reports whether a read between the append a and the fill w may see the appended element: from the statement
// holding a in the statement list holding both, to the statement holding w, a read of the collection as a whole or of
// data holding it, or an element read whose key the position flows to.
func (m *Analyzer) seen(a, w Effect, position *ast.Node, reads []Read) bool {
	common := commonList(a.Node, w.Node)
	if common == nil {
		return true
	}
	list := statementList(common)
	from := slices.IndexFunc(list, func(s *ast.Node) bool { return encloses(s, a.Node) })
	to := slices.IndexFunc(list, func(s *ast.Node) bool { return encloses(s, w.Node) })
	start, end := list[from].Pos(), list[to].End()
	if !ast.IsIterationStatement(list[from], false) && from != to {
		start = a.Node.End()
	}
	collection := a.Receiver
	return slices.ContainsFunc(reads, func(r Read) bool {
		if r.Node.Pos() < start || r.Node.End() > end {
			return false
		}
		switch {
		case r.Size && r.Value.Same(collection):
			return false
		case r.Value.Same(collection) && r.Key != nil:
			return slices.Contains(m.OriginsOf(r.Key), position)
		case r.Value.Within(collection) && !r.Value.Same(collection):
			return false
		}
		return collection.Within(r.Value)
	})
}
