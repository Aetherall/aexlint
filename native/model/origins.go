package model

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
)

// ValueSource is an origin: an expression or declaration (Node), or the elements of data the model cannot see the contents
// of (Elements), which every read of one of its elements shares.
type ValueSource struct {
	Node     *ast.Node
	Elements place
}

// SourceText describes a source for messages.
func SourceText(s ValueSource) string {
	if s.Node != nil {
		return Text(s.Node, 30)
	}
	v := Value{Root: s.Elements.root, This: s.Elements.this, Path: []string{}}
	if s.Elements.path != "" {
		v.Path = strings.Split(s.Elements.path, "\x00")
	}
	return "elements of " + v.String()
}

// elementsOf returns the source the elements of data n reads share, when n names that data.
func elementsOf(v Value) ([]ValueSource, bool) {
	p, ok := placeOf(v)
	if !ok {
		return nil, false
	}
	return []ValueSource{{Elements: p}}, true
}

// OriginsOf returns where the value n gives comes from, as the expressions or declarations the model cannot follow
// further: a new object, a constant, a size read, a read of data, a computed value, or a parameter or variable whose
// definitions are not seen. An element of data whose contents are not seen, read by element access or a lookup,
// bound by a for-of loop over it, or passed to the callback of an array or set method on it, comes from its elements,
// one source for every such read. Origins are followed through const initializers and the definitions of other variables
// reaching the read, conditional and logical operators, calls to project functions and their returns (a parameter
// giving the origins of the call's argument), and the entries of local containers (see Contents): a lookup gives the
// origins of the values stored, and a loop over the container binds its keys and values to theirs.
func (m *Analyzer) OriginsOf(n *ast.Node) []ValueSource {
	return m.origins(n, &originTracing{seen: map[*ast.Symbol]bool{}, calling: map[*ast.Node]bool{}})
}

type originTracing struct {
	seen    map[*ast.Symbol]bool
	calling map[*ast.Node]bool
	bound   map[*ast.Symbol][]ValueSource
}

func unionSources(a, b []ValueSource) []ValueSource {
	for _, n := range b {
		if !slices.Contains(a, n) {
			a = append(a, n)
		}
	}
	return a
}

func (m *Analyzer) origins(n *ast.Node, t *originTracing) []ValueSource {
	n = Unwrap(n)
	if n == nil {
		return nil
	}
	switch n.Kind {
	case ast.KindConditionalExpression:
		c := n.AsConditionalExpression()
		return unionSources(m.origins(c.WhenTrue, t), m.origins(c.WhenFalse, t))
	case ast.KindBinaryExpression:
		b := n.AsBinaryExpression()
		switch operator := b.OperatorToken.Kind; {
		case ast.IsLogicalOperator(operator) || operator == ast.KindQuestionQuestionToken:
			return unionSources(m.origins(b.Left, t), m.origins(b.Right, t))
		case operator == ast.KindCommaToken || operator == ast.KindEqualsToken:
			return m.origins(b.Right, t)
		}
	case ast.KindAwaitExpression:
		if call := Unwrap(n.Expression()); call.Kind == ast.KindCallExpression {
			if found, ok := m.callOrigins(call, t); ok {
				return found
			}
		}
	case ast.KindCallExpression:
		if found, ok := m.callOrigins(n, t); ok {
			return found
		}
	case ast.KindElementAccessExpression:
		lookup, _ := m.LookupOf(n)
		if contents, ok := m.Contents(lookup.Container, n); ok {
			found := []ValueSource{}
			for _, entry := range contents.Entries {
				found = unionSources(found, m.origins(entry.Value, t))
			}
			return found
		}
		if found, ok := elementsOf(lookup.Container); ok {
			return found
		}
	case ast.KindIdentifier:
		return m.variableOrigins(n, t)
	}
	return []ValueSource{{Node: n}}
}

// variableOrigins returns the origins of a variable or parameter read.
func (m *Analyzer) variableOrigins(n *ast.Node, t *originTracing) []ValueSource {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
	if symbol == nil || symbol.ValueDeclaration == nil || t.seen[symbol] {
		return []ValueSource{{Node: n}}
	}
	d := symbol.ValueDeclaration
	t.seen[symbol] = true
	defer delete(t.seen, symbol)
	if found, ok := m.iterated(d, t); ok {
		return found
	}
	if found, ok := m.elementParameter(d); ok {
		return found
	}
	if d.Kind != ast.KindVariableDeclaration && d.Kind != ast.KindParameter {
		return []ValueSource{{Node: d}}
	}
	if d.Kind == ast.KindVariableDeclaration && ast.IsVarConst(d) && d.Initializer() != nil {
		return m.origins(d.Initializer(), t)
	}
	found := []ValueSource{}
	for _, def := range m.reaching(n, Value{Root: symbol, Path: []string{}}) {
		switch {
		case def.Before:
			if bound, ok := t.bound[symbol]; ok {
				found = unionSources(found, bound)
			} else {
				found = unionSources(found, []ValueSource{{Node: d}})
			}
		case def.Declaration != nil && def.Declaration.Initializer() != nil:
			found = unionSources(found, m.origins(def.Declaration.Initializer(), t))
		case def.Write.Kind == Assignment && def.Write.Source != nil:
			found = unionSources(found, m.origins(def.Write.Source, t))
		case def.Declaration != nil:
			found = unionSources(found, []ValueSource{{Node: def.Declaration}})
		default:
			found = unionSources(found, []ValueSource{{Node: def.Write.Node}})
		}
	}
	return found
}

// callOrigins returns the origins of a call: the values stored in a local container it looks an entry up in, or the
// returns of the project function it resolves to, with its parameters bound to the call's arguments.
func (m *Analyzer) callOrigins(n *ast.Node, t *originTracing) ([]ValueSource, bool) {
	if lookup, ok := m.LookupOf(n); ok {
		if contents, ok := m.Contents(lookup.Container, n); ok {
			found := []ValueSource{}
			for _, entry := range contents.Entries {
				found = unionSources(found, m.origins(entry.Value, t))
			}
			return found, true
		}
		return elementsOf(lookup.Container)
	}
	fn := m.followed(n)
	if fn == nil || t.calling[fn] {
		return nil, false
	}
	bound := map[*ast.Symbol][]ValueSource{}
	arguments := n.Arguments()
	for i, parameter := range fn.Parameters() {
		if i >= len(arguments) || arguments[i].Kind == ast.KindSpreadElement || parameter.Name().Kind != ast.KindIdentifier || parameter.AsParameterDeclaration().DotDotDotToken != nil {
			break
		}
		if symbol := parameter.Symbol(); symbol != nil {
			bound[symbol] = m.origins(arguments[i], t)
		}
	}
	t.calling[fn] = true
	saved := t.bound
	t.bound = bound
	defer func() {
		delete(t.calling, fn)
		t.bound = saved
	}()
	body := fn.Body()
	if body.Kind != ast.KindBlock {
		return m.origins(body, t), true
	}
	found := []ValueSource{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindReturnStatement {
			if c.Expression() == nil {
				found = unionSources(found, []ValueSource{{Node: c}})
			} else {
				found = unionSources(found, m.origins(c.Expression(), t))
			}
		}
		c.ForEachChild(walk)
		return false
	}
	body.ForEachChild(walk)
	if m.flowControl().of(body).end != nil {
		found = unionSources(found, []ValueSource{{Node: body}})
	}
	return found, true
}

// iterated returns the origins of a variable a for-of loop over a local container binds: its keys or its values.
func (m *Analyzer) iterated(d *ast.Node, t *originTracing) ([]ValueSource, bool) {
	position := -1
	declaration := d
	if d.Kind == ast.KindBindingElement && d.Parent.Kind == ast.KindArrayBindingPattern && d.AsBindingElement().DotDotDotToken == nil {
		position = slices.Index(d.Parent.AsBindingPattern().Elements.Nodes, d)
		declaration = d.Parent.Parent
	}
	if declaration.Kind != ast.KindVariableDeclaration || declaration.Parent == nil || declaration.Parent.Parent == nil || declaration.Parent.Parent.Kind != ast.KindForOfStatement {
		return nil, false
	}
	loop := declaration.Parent.Parent.AsForInOrOfStatement()
	if loop.Initializer != declaration.Parent {
		return nil, false
	}
	over, part := m.iteration(loop.Expression)
	contents, ok := m.Contents(over, declaration)
	if !ok {
		if part == valuePart && position < 0 {
			return elementsOf(over)
		}
		return nil, false
	}
	if part == entryParts && position >= 0 {
		part = []iterationPart{keyPart, valuePart}[min(position, 1)]
		if position > 1 {
			return nil, false
		}
	}
	if part == entryParts {
		return nil, false
	}
	found := []ValueSource{}
	for _, entry := range contents.Entries {
		of := entry.Value
		if part == keyPart {
			of = entry.Key
		}
		if of == nil {
			return nil, false
		}
		found = unionSources(found, m.origins(of, t))
	}
	return found, true
}

// elementCallbacks names the global collection methods calling a callback with an element, and its position.
var elementCallbacks = map[string]map[string]int{
	"Array": {"forEach": 0, "map": 0, "filter": 0, "find": 0, "findIndex": 0, "findLast": 0, "findLastIndex": 0, "some": 0,
		"every": 0, "flatMap": 0, "reduce": 1, "reduceRight": 1},
	"ReadonlyArray": {"forEach": 0, "map": 0, "filter": 0, "find": 0, "findIndex": 0, "findLast": 0, "findLastIndex": 0,
		"some": 0, "every": 0, "flatMap": 0, "reduce": 1, "reduceRight": 1},
	"Set": {"forEach": 0}, "ReadonlySet": {"forEach": 0},
}

// elementParameter returns the elements a callback parameter is given, when the callback is the first argument of
// a global collection method calling it with an element of the receiver at that parameter's position.
func (m *Analyzer) elementParameter(d *ast.Node) ([]ValueSource, bool) {
	if d.Kind != ast.KindParameter || d.Parent == nil {
		return nil, false
	}
	callback := d.Parent
	call := callback.Parent
	if call == nil || call.Kind != ast.KindCallExpression || len(call.Arguments()) == 0 || call.Arguments()[0] != callback {
		return nil, false
	}
	callee := Unwrap(call.Expression())
	if callee.Kind != ast.KindPropertyAccessExpression {
		return nil, false
	}
	owners, name, ok := m.libraryMember(callee.Name())
	position, known := agreed(owners, name, elementCallbacks)
	if !ok || !known || slices.Index(callback.Parameters(), d) != position {
		return nil, false
	}
	return elementsOf(m.ValueOf(callee.Expression()))
}

type iterationPart int

const (
	valuePart iterationPart = iota
	keyPart
	entryParts
)

// iteration returns the container a for-of loop iterates and which part of its entries each element is: a map, or its
// entries(), gives [key, value] pairs, its keys() and values() those parts, and an array or set its values.
func (m *Analyzer) iteration(n *ast.Node) (Value, iterationPart) {
	n = Unwrap(n)
	if n.Kind == ast.KindCallExpression && len(n.Arguments()) == 0 {
		if callee := Unwrap(n.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
			if owners, name, ok := m.libraryMember(callee.Name()); ok {
				parts := map[string]iterationPart{"entries": entryParts, "keys": keyPart, "values": valuePart}
				if part, known := parts[name]; known && slices.ContainsFunc(owners, func(o string) bool { return o != "String" }) {
					if name == "entries" && !m.keyed(callee.Expression()) {
						return Value{}, entryParts
					}
					return m.ValueOf(callee.Expression()), part
				}
			}
		}
	}
	if m.keyed(n) {
		return m.ValueOf(n), entryParts
	}
	return m.ValueOf(n), valuePart
}

// keyed reports whether n's type is a global map type.
func (m *Analyzer) keyed(n *ast.Node) bool {
	return slices.ContainsFunc([]string{"Map", "ReadonlyMap"}, func(name string) bool { return m.typeIs(n, name) })
}

// typeIs reports whether the type of n is declared by the global type named name, as libraryMember resolves it.
func (m *Analyzer) typeIs(n *ast.Node, name string) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	global := m.globalType(name)
	if t == nil || global == nil {
		return false
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	symbol := t.Symbol()
	return symbol != nil && len(symbol.Declarations) > 0 && slices.Contains(global.Declarations, symbol.Declarations[0])
}

// Entry is what code puts into a container: Key is the key it stores under, nil for an append or when not known, and
// Value the value it stores, nil when not known.
type Entry struct {
	Node  *ast.Node
	Key   *ast.Node
	Value *ast.Node
}

// Contents is what a local container holds: every entry its function puts into it.
type Contents struct {
	Container Value
	Entries   []Entry
}

// Contents returns what the container v holds as read at n, when it is local: a variable of n's function, never
// rebound, initialized with an empty array or object literal or a construction of a global collection without arguments, and only
// used as the receiver of collection methods, by element access, as the object of a size read, or iterated by for-of.
// Every effect in the function storing or appending into it is an entry, wherever it is; one whose value is unknown
// stops the read.
func (m *Analyzer) Contents(v Value, n *ast.Node) (Contents, bool) {
	fn := enclosingFunction(n)
	if fn == nil || v.Root == nil || len(v.Path) > 0 || v.Partial {
		return Contents{}, false
	}
	d := v.Root.ValueDeclaration
	if d == nil || d.Kind != ast.KindVariableDeclaration || enclosingFunction(d) != fn || m.assigned(v.Root) || !m.empty(d.Initializer()) {
		return Contents{}, false
	}
	if !m.confined(v.Root, fn) {
		return Contents{}, false
	}
	contents := Contents{Container: v}
	for _, e := range m.EffectsIn([]*ast.Node{fn.Body()}) {
		if !e.Receiver.Same(v) {
			continue
		}
		switch e.Kind {
		case Store, Append:
			if e.Stored == nil {
				return Contents{}, false
			}
			contents.Entries = append(contents.Entries, Entry{Node: e.Node, Key: e.Key, Value: e.Stored})
		case Change:
			if e.Node.Kind != ast.KindCallExpression {
				return Contents{}, false
			}
		}
	}
	return contents, true
}

// empty reports whether n creates an empty container: an empty array or object literal, or a construction of a global
// collection without arguments.
func (m *Analyzer) empty(n *ast.Node) bool {
	n = Unwrap(n)
	switch {
	case n == nil:
		return false
	case n.Kind == ast.KindArrayLiteralExpression:
		return len(n.AsArrayLiteralExpression().Elements.Nodes) == 0
	case emptyObject(n):
		return true
	case n.Kind == ast.KindNewExpression:
		return len(n.Arguments()) == 0 && slices.ContainsFunc(collectionTypes, func(name string) bool { return m.typeIs(n, name) })
	}
	return false
}

// emptyObject reports whether n is an object literal without properties.
func emptyObject(n *ast.Node) bool {
	n = Unwrap(n)
	return n != nil && n.Kind == ast.KindObjectLiteralExpression && len(n.Properties()) == 0
}

// confined reports whether every read of symbol in fn uses it as a container: the receiver of a global collection
// method, the object of an element access or a size read, or the expression a for-of loop iterates.
func (m *Analyzer) confined(symbol *ast.Symbol, fn *ast.Node) bool {
	confined := true
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if !confined {
			return true
		}
		if c.Kind == ast.KindIdentifier && c.Parent != nil && c.Parent.Kind != ast.KindVariableDeclaration && m.ctx.TypeChecker.GetSymbolAtLocation(c) == symbol {
			confined = m.containerUse(c)
			return !confined
		}
		c.ForEachChild(walk)
		return !confined
	}
	walk(fn.Body())
	return confined
}

func (m *Analyzer) containerUse(c *ast.Node) bool {
	p := c.Parent
	for p.Kind == ast.KindParenthesizedExpression || p.Kind == ast.KindNonNullExpression {
		c, p = p, p.Parent
	}
	switch {
	case p.Kind == ast.KindElementAccessExpression && p.Expression() == c:
		return true
	case p.Kind == ast.KindForOfStatement && p.AsForInOrOfStatement().Expression == c:
		return true
	case p.Kind == ast.KindPropertyAccessExpression && p.Expression() == c:
		_, _, ok := m.libraryMember(p.Name())
		return ok
	}
	return false
}
