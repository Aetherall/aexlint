package model

import (
	"maps"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// exit is one way out of a statement, with the writes made on the paths that take it and, when its control tracks
// places, the definitions of each place reaching it (see defs).
type exit struct {
	writes []Write
	defs   defs
}

// ending is how a statement can end: by reaching its end (end), by a break leaving a statement around it (breaks, by
// that statement), by a continue resuming a loop around it (continues, by that loop), by a return, or by a throw or a
// call returning never. A nil exit is a way it cannot end.
type ending struct {
	end       *exit
	breaks    map[*ast.Node]*exit
	continues map[*ast.Node]*exit
	returns   bool
	throws    bool
}

// control reads endings. Without a checker, never is nil, so no call is known to return never, and exhaustive is nil,
// so no switch without default is known to cover every value. When writes is set, exits carry the writes on their way,
// and when place is set, the definitions of the places it names.
type control struct {
	never      func(*ast.Node) bool
	exhaustive func(*ast.Node) bool
	writes     bool
	place      func(Write) (place, bool)
	endings    map[*ast.Node]ending
}

func join(a, b *exit) *exit {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	}
	return &exit{writes: append(slices.Clip(a.writes), b.writes...), defs: joinDefs(a.defs, b.defs)}
}

func joinAll(a, b map[*ast.Node]*exit) map[*ast.Node]*exit {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	joined := maps.Clone(a)
	for target, x := range b {
		joined[target] = join(joined[target], x)
	}
	return joined
}

func (e ending) merge(o ending) ending {
	return ending{end: join(e.end, o.end), breaks: joinAll(e.breaks, o.breaks), continues: joinAll(e.continues, o.continues),
		returns: e.returns || o.returns, throws: e.throws || o.throws}
}

// after returns the ending of a statement run after the way before it: its writes come first, and its definitions
// reach the places the statement does not define on every way.
func (e ending) after(before *exit) ending {
	if before == nil || len(before.writes) == 0 && len(before.defs) == 0 {
		return e
	}
	prefix := func(x *exit) *exit {
		if x == nil {
			return nil
		}
		return &exit{writes: append(slices.Clip(before.writes), x.writes...), defs: composeDefs(before.defs, x.defs)}
	}
	all := func(exits map[*ast.Node]*exit) map[*ast.Node]*exit {
		if len(exits) == 0 {
			return exits
		}
		prefixed := map[*ast.Node]*exit{}
		for target, x := range exits {
			prefixed[target] = prefix(x)
		}
		return prefixed
	}
	return ending{end: prefix(e.end), breaks: all(e.breaks), continues: all(e.continues), returns: e.returns, throws: e.throws}
}

// without removes the breaks and continues to target, returning each joined.
func (e ending) without(target *ast.Node) (ending, *exit, *exit) {
	broken, continued := e.breaks[target], e.continues[target]
	if broken == nil && continued == nil {
		return e, nil, nil
	}
	e.breaks, e.continues = maps.Clone(e.breaks), maps.Clone(e.continues)
	delete(e.breaks, target)
	delete(e.continues, target)
	return e, broken, continued
}

func (c *control) effects(nodes ...*ast.Node) []Write {
	if !c.writes {
		return nil
	}
	return writesIn(nodes, false)
}

// effected returns the way through nodes, evaluated in order, with their writes and definitions.
func (c *control) effected(nodes ...*ast.Node) *exit {
	writes := c.effects(nodes...)
	return &exit{writes: writes, defs: c.defined(writes, false)}
}

func reached(writes []Write) ending {
	return ending{end: &exit{writes: writes}}
}

// reach returns the ending of code making writes in order and reaching its end.
func (c *control) reach(writes []Write) ending {
	return ending{end: &exit{writes: writes, defs: c.defined(writes, false)}}
}

// list reads statements run in order: a statement that cannot reach its end stops the rest.
func (c *control) list(statements []*ast.Node) ending {
	result := reached(nil)
	for _, s := range statements {
		if result.end == nil {
			break
		}
		e := c.of(s).after(result.end)
		e.breaks, e.continues = joinAll(result.breaks, e.breaks), joinAll(result.continues, e.continues)
		e.returns, e.throws = e.returns || result.returns, e.throws || result.throws
		result = e
	}
	return result
}

func (c *control) of(n *ast.Node) ending {
	if n == nil {
		return reached(nil)
	}
	if e, ok := c.endings[n]; ok {
		return e
	}
	e := c.read(n)
	if c.endings != nil {
		c.endings[n] = e
	}
	return e
}

func (c *control) read(n *ast.Node) ending {
	switch n.Kind {
	case ast.KindBlock:
		return c.list(n.Statements())
	case ast.KindExpressionStatement:
		if c.never != nil && c.never(n.Expression()) {
			return ending{throws: true}
		}
		return c.reach(c.effects(n.Expression()))
	case ast.KindReturnStatement:
		return ending{returns: true}
	case ast.KindThrowStatement:
		return ending{throws: true}
	case ast.KindBreakStatement:
		if target := jumpTarget(n); target != nil {
			return ending{breaks: map[*ast.Node]*exit{target: {}}}
		}
		return ending{}
	case ast.KindContinueStatement:
		if target := jumpTarget(n); target != nil {
			return ending{continues: map[*ast.Node]*exit{target: {}}}
		}
		return ending{}
	case ast.KindIfStatement:
		s := n.AsIfStatement()
		return c.of(s.ThenStatement).merge(c.of(s.ElseStatement)).after(c.effected(s.Expression))
	case ast.KindSwitchStatement:
		return c.switchStatement(n)
	case ast.KindForStatement, ast.KindForInStatement, ast.KindForOfStatement, ast.KindWhileStatement, ast.KindDoStatement:
		return c.loop(n)
	case ast.KindLabeledStatement:
		e, broken, _ := c.of(n.AsLabeledStatement().Statement).without(n)
		e.end = join(e.end, broken)
		return e
	case ast.KindTryStatement:
		return c.try(n)
	}
	if ast.IsFunctionLike(n) || ast.IsClassLike(n) || ast.IsDeclaration(n) && n.Kind != ast.KindVariableStatement {
		return reached(nil)
	}
	return c.reach(c.effects(n))
}

func (c *control) switchStatement(n *ast.Node) ending {
	s := n.AsSwitchStatement()
	result := ending{}
	var falling *exit
	clauses := s.CaseBlock.AsCaseBlock().Clauses.Nodes
	for _, clause := range clauses {
		entry := join(&exit{}, falling)
		e := c.list(clause.Statements()).after(entry)
		falling = e.end
		e.end = nil
		result = result.merge(e)
	}
	result, broken, _ := result.without(n)
	result.end = join(falling, broken)
	covered := slices.ContainsFunc(clauses, func(c *ast.Node) bool { return c.Kind == ast.KindDefaultClause }) || c.exhaustive != nil && c.exhaustive(n)
	if !covered || len(clauses) == 0 {
		result.end = join(result.end, &exit{})
	}
	return result.after(c.effected(s.Expression))
}

// loop reads a loop: it ends when its condition fails before or after an iteration, or through a break leaving it. A
// loop without a condition, or whose condition is true, ends only through a break. Definitions made by one iteration
// reach the next, so each way out also carries those of earlier iterations.
func (c *control) loop(n *ast.Node) ending {
	var once, each []Write
	var condition *ast.Node
	switch n.Kind {
	case ast.KindForStatement:
		s := n.AsForStatement()
		once, each, condition = c.effects(s.Initializer), c.effects(s.Condition, s.Incrementor), s.Condition
	case ast.KindForInStatement, ast.KindForOfStatement:
		s := n.AsForInOrOfStatement()
		once, condition = c.effects(s.Expression), s.Expression
		if c.writes {
			each = WritesAt(n)
		}
	case ast.KindWhileStatement, ast.KindDoStatement:
		condition = n.Expression()
		each = c.effects(condition)
	}
	body := c.of(n.Statement()).after(&exit{writes: each, defs: c.defined(each, false)})
	body = body.repeated(join(join(&exit{}, body.end), body.continues[n]))
	body, broken, continued := body.without(n)
	infinite := condition == nil || condition.Kind == ast.KindTrueKeyword
	result := body
	result.end = broken
	if !infinite {
		result.end = join(join(result.end, body.end), continued)
		if n.Kind != ast.KindDoStatement {
			result.end = join(result.end, &exit{})
		}
	}
	return result.after(&exit{writes: once, defs: c.defined(once, false)})
}

// try reads a try statement: a throw in its block goes to its catch clause, and a finally block runs on every way out.
func (c *control) try(n *ast.Node) ending {
	s := n.AsTryStatement()
	result := c.of(s.TryBlock)
	if s.CatchClause != nil {
		thrown := c.effects(s.TryBlock)
		caught := c.of(s.CatchClause.AsCatchClause().Block).after(&exit{writes: thrown, defs: c.defined(thrown, true)})
		result.throws = false
		result = result.merge(caught)
	}
	if s.FinallyBlock == nil {
		return result
	}
	finally := c.of(s.FinallyBlock)
	if finally.end == nil {
		return finally
	}
	suffix := func(x *exit) *exit {
		if x == nil {
			return nil
		}
		return &exit{writes: append(slices.Clip(x.writes), finally.end.writes...), defs: composeDefs(x.defs, finally.end.defs)}
	}
	all := func(exits map[*ast.Node]*exit) map[*ast.Node]*exit {
		suffixed := map[*ast.Node]*exit{}
		for target, x := range exits {
			suffixed[target] = suffix(x)
		}
		return suffixed
	}
	result.end, result.breaks, result.continues = suffix(result.end), all(result.breaks), all(result.continues)
	return result.merge(ending{breaks: finally.breaks, continues: finally.continues, returns: finally.returns, throws: finally.throws})
}

// jumpTarget returns the statement a break leaves or the loop a continue resumes: the statement its label names, or
// else the nearest loop, or switch for a break, around it.
func jumpTarget(n *ast.Node) *ast.Node {
	label := n.Label()
	for p := n.Parent; p != nil && !ast.IsFunctionLike(p) && !ast.IsClassLike(p); p = p.Parent {
		switch {
		case label != nil:
			if p.Kind == ast.KindLabeledStatement && p.AsLabeledStatement().Label.Text() == label.Text() {
				if n.Kind == ast.KindContinueStatement {
					return p.AsLabeledStatement().Statement
				}
				return p
			}
		case ast.IsIterationStatement(p, false):
			return p
		case p.Kind == ast.KindSwitchStatement && n.Kind == ast.KindBreakStatement:
			return p
		}
	}
	return nil
}

// flowControl reads endings with the checker, without writes.
func (m *Analyzer) flowControl() *control {
	if m.control == nil {
		m.control = &control{never: m.neverCall, exhaustive: m.Exhaustive, endings: map[*ast.Node]ending{}}
	}
	return m.control
}

// writeControl reads endings with the checker, and the writes and definitions on each way out.
func (m *Analyzer) writeControl() *control {
	if m.writing == nil {
		m.writing = &control{never: m.neverCall, exhaustive: m.Exhaustive, writes: true, place: m.placeOfWrite, endings: map[*ast.Node]ending{}}
	}
	return m.writing
}

// neverCall reports whether an expression, awaited or not, is a call returning never.
func (m *Analyzer) neverCall(e *ast.Node) bool {
	e = skipOuter(e)
	if e == nil || e.Kind != ast.KindCallExpression {
		return false
	}
	t := m.ctx.TypeChecker.GetTypeAtLocation(e)
	return t != nil && t.Flags()&checker.TypeFlagsNever != 0
}

// listEnding reads statements run in order.
func (m *Analyzer) listEnding(statements []*ast.Node) ending {
	return m.flowControl().list(statements)
}

// returnsEarly reports whether a statement always leaves its function, by a return on some way through it.
func (m *Analyzer) returnsEarly(n *ast.Node) bool {
	if n == nil {
		return false
	}
	e := m.flowControl().of(n)
	return e.end == nil && len(e.breaks) == 0 && len(e.continues) == 0 && e.returns
}

// EndingText describes the ways a statement can end, for review.
func (m *Analyzer) EndingText(n *ast.Node) string {
	e := m.flowControl().of(n)
	ways := []string{}
	if e.end != nil {
		ways = append(ways, "reaches its end")
	}
	byPosition := func(a, b *ast.Node) int { return a.Pos() - b.Pos() }
	for _, target := range slices.SortedFunc(maps.Keys(e.breaks), byPosition) {
		ways = append(ways, "breaks out of "+Text(target, 30))
	}
	for _, target := range slices.SortedFunc(maps.Keys(e.continues), byPosition) {
		ways = append(ways, "continues "+Text(target, 30))
	}
	if e.returns {
		ways = append(ways, "returns")
	}
	if e.throws {
		ways = append(ways, "throws")
	}
	if len(ways) == 0 {
		return "never ends"
	}
	return strings.Join(ways, "; ")
}
