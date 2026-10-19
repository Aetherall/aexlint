package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
)

// OriginsOf returns where the value n gives comes from, as the expressions or declarations the model cannot follow
// further: a new object, a constant, a size read, a read of data, a computed value, or a parameter or variable whose
// definitions are not seen. Origins are followed through const initializers and the definitions of other variables
// reaching the read, conditional and logical operators, calls to project functions and their returns (a parameter
// giving the origins of the call's argument), and the entries of local containers (see Contents): a lookup gives the
// origins of the values stored, and a loop over the container binds its keys and values to theirs.
func (m *Analyzer) OriginsOf(n *ast.Node) []*ast.Node {
	return m.origins(n, &originTracing{seen: map[*ast.Symbol]bool{}, calling: map[*ast.Node]bool{}})
}

type originTracing struct {
	seen    map[*ast.Symbol]bool
	calling map[*ast.Node]bool
	bound   map[*ast.Symbol][]*ast.Node
}

func unionNodes(a, b []*ast.Node) []*ast.Node {
	for _, n := range b {
		if !slices.Contains(a, n) {
			a = append(a, n)
		}
	}
	return a
}

func (m *Analyzer) origins(n *ast.Node, t *originTracing) []*ast.Node {
	n = Unwrap(n)
	if n == nil {
		return nil
	}
	switch n.Kind {
	case ast.KindConditionalExpression:
		c := n.AsConditionalExpression()
		return unionNodes(m.origins(c.WhenTrue, t), m.origins(c.WhenFalse, t))
	case ast.KindBinaryExpression:
		b := n.AsBinaryExpression()
		switch operator := b.OperatorToken.Kind; {
		case ast.IsLogicalOperator(operator) || operator == ast.KindQuestionQuestionToken:
			return unionNodes(m.origins(b.Left, t), m.origins(b.Right, t))
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
	case ast.KindIdentifier:
		return m.variableOrigins(n, t)
	}
	return []*ast.Node{n}
}

// variableOrigins returns the origins of a variable or parameter read.
func (m *Analyzer) variableOrigins(n *ast.Node, t *originTracing) []*ast.Node {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
	if symbol == nil || symbol.ValueDeclaration == nil || t.seen[symbol] {
		return []*ast.Node{n}
	}
	d := symbol.ValueDeclaration
	t.seen[symbol] = true
	defer delete(t.seen, symbol)
	if found, ok := m.iterated(d, t); ok {
		return found
	}
	if d.Kind != ast.KindVariableDeclaration && d.Kind != ast.KindParameter {
		return []*ast.Node{d}
	}
	if d.Kind == ast.KindVariableDeclaration && ast.IsVarConst(d) && d.Initializer() != nil {
		return m.origins(d.Initializer(), t)
	}
	found := []*ast.Node{}
	for _, def := range m.reaching(n, Value{Root: symbol, Path: []string{}}) {
		switch {
		case def.Before:
			if bound, ok := t.bound[symbol]; ok {
				found = unionNodes(found, bound)
			} else {
				found = unionNodes(found, []*ast.Node{d})
			}
		case def.Declaration != nil && def.Declaration.Initializer() != nil:
			found = unionNodes(found, m.origins(def.Declaration.Initializer(), t))
		case def.Write.Kind == Assignment && def.Write.Source != nil:
			found = unionNodes(found, m.origins(def.Write.Source, t))
		case def.Declaration != nil:
			found = unionNodes(found, []*ast.Node{def.Declaration})
		default:
			found = unionNodes(found, []*ast.Node{def.Write.Node})
		}
	}
	return found
}

// callOrigins returns the origins of a call: the values stored in a local container it looks an entry up in, or the
// returns of the project function it resolves to, with its parameters bound to the call's arguments.
func (m *Analyzer) callOrigins(n *ast.Node, t *originTracing) ([]*ast.Node, bool) {
	if lookup, ok := m.LookupOf(n); ok {
		if contents, ok := m.Contents(lookup.Container, n); ok {
			found := []*ast.Node{}
			for _, entry := range contents.Entries {
				found = unionNodes(found, m.origins(entry.Value, t))
			}
			return found, true
		}
		return nil, false
	}
	fn := m.followed(n)
	if fn == nil || t.calling[fn] {
		return nil, false
	}
	bound := map[*ast.Symbol][]*ast.Node{}
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
	found := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindReturnStatement {
			if c.Expression() == nil {
				found = unionNodes(found, []*ast.Node{c})
			} else {
				found = unionNodes(found, m.origins(c.Expression(), t))
			}
		}
		c.ForEachChild(walk)
		return false
	}
	body.ForEachChild(walk)
	if m.flowControl().of(body).end != nil {
		found = unionNodes(found, []*ast.Node{body})
	}
	return found, true
}

// iterated returns the origins of a variable a for-of loop over a local container binds: its keys or its values.
func (m *Analyzer) iterated(d *ast.Node, t *originTracing) ([]*ast.Node, bool) {
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
	found := []*ast.Node{}
	for _, entry := range contents.Entries {
		of := entry.Value
		if part == keyPart {
			of = entry.Key
		}
		if of == nil {
			return nil, false
		}
		found = unionNodes(found, m.origins(of, t))
	}
	return found, true
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
// rebound, initialized with an empty array literal or a construction of a global collection without arguments, and only
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

// empty reports whether n creates an empty container: an empty array literal, or a construction of a global
// collection without arguments.
func (m *Analyzer) empty(n *ast.Node) bool {
	n = Unwrap(n)
	switch {
	case n == nil:
		return false
	case n.Kind == ast.KindArrayLiteralExpression:
		return len(n.AsArrayLiteralExpression().Elements.Nodes) == 0
	case n.Kind == ast.KindNewExpression:
		return len(n.Arguments()) == 0 && slices.ContainsFunc(collectionTypes, func(name string) bool { return m.typeIs(n, name) })
	}
	return false
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
