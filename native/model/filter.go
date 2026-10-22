package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// Filter is a guard in the body of a for-of loop, or of the callback of a forEach call, that skips elements by a test
// on the element alone: Guard is an if statement without else whose branch only continues Loop, or only returns nothing
// from the callback, after statements of the body that only declare variables, never rebound, from initializers making
// no effect and calling no project function that throws, and Test is its condition, which makes no effect and calls no
// such function either. Every read the test and those initializers make is of data the loop does not change, besides
// binding the element and those variables anew for each element: no effect in the loop acts on data within it or
// holding it, and nothing in the loop rebinds it, so an element reached twice is tested alike. Exposed lists the code
// the model does not see (see Unseen) on one side, in the test and those initializers or in the rest of the loop, while
// the other reads or changes data such code may reach (see reachable).
type Filter struct {
	Loop    *ast.Node
	Guard   *ast.Node
	Test    *ast.Node
	Exposed []*ast.Node
}

// Filters lists the filters of a for-of loop or a forEach call, in order.
func (m *Analyzer) Filters(loop *ast.Node) []Filter {
	body := m.elementBody(loop)
	if body == nil || body.Kind != ast.KindBlock {
		return nil
	}
	fn := enclosingFunction(loop)
	if fn == nil {
		return nil
	}
	selection := []*ast.Node{}
	found := []Filter{}
	for _, statement := range statementList(body) {
		switch {
		case m.skips(statement, loop):
			test := statement.Expression()
			if len(m.EffectsIn([]*ast.Node{test})) > 0 || len(m.Throwing([]*ast.Node{test})) > 0 || !m.selects(loop, append(selection, test)) {
				return m.exposedFilters(found)
			}
			selection = append(selection, test)
			found = append(found, Filter{Loop: loop, Guard: statement, Test: test})
		case statement.Kind == ast.KindVariableStatement:
			declarations := statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes
			for _, d := range declarations {
				symbols := declaredBy(d)
				if len(symbols) == 0 || d.Initializer() == nil || slices.ContainsFunc(symbols, m.assigned) || len(m.EffectsIn([]*ast.Node{d.Initializer()})) > 0 || len(m.Throwing([]*ast.Node{d.Initializer()})) > 0 {
					return m.exposedFilters(found)
				}
				selection = append(selection, d.Initializer())
			}
		default:
			return m.exposedFilters(found)
		}
	}
	return m.exposedFilters(found)
}

// exposedFilters fills the exposure of a loop's filters.
func (m *Analyzer) exposedFilters(filters []Filter) []Filter {
	if len(filters) == 0 {
		return filters
	}
	loop := filters[len(filters)-1].Loop
	last := filters[len(filters)-1].Guard
	fn := enclosingFunction(loop)
	list := statementList(m.elementBody(loop))
	selection := list[:indexOf(list, last)+1]
	rest := list[indexOf(list, last)+1:]
	exposed := m.Exposure(fn, loop.End(), Side{Nodes: selection}, Side{Nodes: rest})
	for i := range filters {
		filters[i].Exposed = exposed
	}
	return filters
}

// elementBody returns the code a for-of loop, or the callback of a forEach call on a global collection, runs for each
// element.
func (m *Analyzer) elementBody(loop *ast.Node) *ast.Node {
	switch loop.Kind {
	case ast.KindForOfStatement:
		return loop.Statement()
	case ast.KindCallExpression:
		callee := Unwrap(loop.Expression())
		if callee.Kind != ast.KindPropertyAccessExpression || len(loop.Arguments()) == 0 {
			return nil
		}
		if _, name, ok := m.libraryMember(callee.Name()); !ok || name != "forEach" {
			return nil
		}
		if callback := Unwrap(loop.Arguments()[0]); ast.IsFunctionLike(callback) {
			return callback.Body()
		}
	}
	return nil
}

// skips reports whether a statement is an if statement without else whose branch only continues loop, or, for a
// forEach call, only returns nothing.
func (m *Analyzer) skips(statement, loop *ast.Node) bool {
	if statement.Kind != ast.KindIfStatement || statement.AsIfStatement().ElseStatement != nil {
		return false
	}
	then := statement.AsIfStatement().ThenStatement
	if loop.Kind == ast.KindCallExpression {
		if then.Kind == ast.KindBlock && len(statementList(then)) == 1 {
			then = statementList(then)[0]
		}
		return then.Kind == ast.KindReturnStatement && then.Expression() == nil
	}
	e := m.flowControl().of(then)
	return e.end == nil && !e.returns && !e.throws && len(e.breaks) == 0 && len(e.continues) == 1 && e.continues[loop] != nil
}

// selects reports whether the reads nodes make are of data loop does not change.
func (m *Analyzer) selects(loop *ast.Node, nodes []*ast.Node) bool {
	effects := m.EffectsIn([]*ast.Node{m.elementBody(loop)})
	writes := writesIn([]*ast.Node{loop}, true)
	for _, r := range m.ReadsIn(nodes) {
		if !r.Value.Named() {
			return false
		}
		if slices.ContainsFunc(effects, func(e Effect) bool { return e.Receiver.Within(r.Value) || r.Value.Within(e.Receiver) }) {
			return false
		}
		if slices.ContainsFunc(writes, func(w Write) bool {
			v, ok := m.WriteValue(w)
			return w.Rebinds() && ok && (r.Value.Within(v) || v.Within(r.Value))
		}) {
			return false
		}
	}
	return true
}
