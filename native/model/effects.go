package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// EffectKind is what an effect does to the object it acts on.
type EffectKind string

const (
	// Store puts a value under a key: Map.set, WeakMap.set, or an element assignment with a computed key.
	Store EffectKind = "store"
	// Append adds a value: Array.push, Array.unshift, Set.add or WeakSet.add.
	Append EffectKind = "append"
	// Remove takes entries out: delete, clear, pop, shift or splice.
	Remove EffectKind = "remove"
	// Change changes it otherwise: a method whose body is unknown run as a command, a library method reordering or
	// filling an array, or an assignment or update of data inside it.
	Change EffectKind = "change"
)

// Effect is what code does to an object it acts on: Node is the write or call doing it, Receiver that object, Key
// where a store or removal acts, and Stored the value it puts there, when an expression at Node names them. Carried
// are the expressions at Node whose data the effect may put into the receiver.
type Effect struct {
	Node     *ast.Node
	Kind     EffectKind
	Receiver Value
	Key      *ast.Node
	Stored   *ast.Node
	Carried  []*ast.Node
}

// libraryEffects names the effect of default library methods changing a collection, and the positions of their key
// and stored value, or -1.
type libraryEffect struct {
	kind        EffectKind
	key, stored int
}

var libraryEffects = map[string]map[string]libraryEffect{
	"Map":     {"set": {Store, 0, 1}, "delete": {Remove, 0, -1}, "clear": {Remove, -1, -1}},
	"WeakMap": {"set": {Store, 0, 1}, "delete": {Remove, 0, -1}},
	"Set":     {"add": {Append, -1, 0}, "delete": {Remove, -1, 0}, "clear": {Remove, -1, -1}},
	"WeakSet": {"add": {Append, -1, 0}, "delete": {Remove, -1, 0}},
	"Array": {"push": {Append, -1, 0}, "unshift": {Append, -1, 0}, "pop": {Remove, -1, -1}, "shift": {Remove, -1, -1},
		"splice": {Remove, 0, -1}, "sort": {Change, -1, -1}, "reverse": {Change, -1, -1}, "fill": {Change, -1, 0}, "copyWithin": {Change, -1, -1}},
}

func argument(arguments []*ast.Node, i int) *ast.Node {
	if i < 0 || i >= len(arguments) {
		return nil
	}
	return arguments[i]
}

func carrying(nodes ...*ast.Node) []*ast.Node {
	return slices.DeleteFunc(nodes, func(n *ast.Node) bool { return n == nil })
}

// EffectsAt lists the effects n itself makes, not those of the nodes inside it. A call or construction resolving to a
// project function with a body makes the effects its summary gives, on its receiver and arguments; see Summary. A
// call to a method whose body is unknown, run as a command, changes its receiver and may carry its arguments into it;
// a default library method other than the collection methods the table knows, a function held by a property or
// namespace, or a callee the checker cannot resolve has none. A rebinding of a variable or this is a definition, not an
// effect.
func (m *Analyzer) EffectsAt(n *ast.Node) []Effect {
	if fn := m.followed(n); fn != nil {
		return m.applied(n, fn)
	}
	effects := []Effect{}
	for _, w := range WritesAt(n) {
		if effect, ok := m.effectOf(w); ok {
			effects = append(effects, effect)
		}
	}
	return effects
}

// EffectsIn lists the effects of nodes in order, outside the functions they declare.
func (m *Analyzer) EffectsIn(nodes []*ast.Node) []Effect {
	effects := []Effect{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		effects = append(effects, m.EffectsAt(n)...)
		n.ForEachChild(walk)
		return false
	}
	for _, n := range nodes {
		if n != nil {
			walk(n)
		}
	}
	return effects
}

func (m *Analyzer) effectOf(w Write) (Effect, bool) {
	target := Unwrap(w.Target)
	if w.Kind == Mutation {
		effect := Effect{Node: w.Node, Kind: Change, Receiver: m.ValueOf(target), Carried: carrying(w.Node.Arguments()...)}
		callee := Unwrap(w.Node.Expression())
		name := callee
		if callee.Kind == ast.KindPropertyAccessExpression {
			name = callee.Name()
		}
		if owners, member, ok := m.libraryMember(name); ok {
			known, found := agreed(owners, member, libraryEffects)
			if !found {
				return Effect{}, false
			}
			arguments := w.Node.Arguments()
			effect.Kind, effect.Key, effect.Stored = known.kind, argument(arguments, known.key), argument(arguments, known.stored)
			effect.Carried = carrying(effect.Stored)
			return effect, true
		}
		if utils.IsSymbolFromDefaultLibrary(m.ctx.Program, m.ctx.TypeChecker.GetSymbolAtLocation(name)) {
			return Effect{}, false
		}
		return effect, w.Discarded() && m.method(name)
	}
	if target.Kind == ast.KindElementAccessExpression {
		key := Unwrap(target.AsElementAccessExpression().ArgumentExpression)
		if !ast.IsStringLiteralLike(key) && key.Kind != ast.KindNumericLiteral {
			return Effect{Node: w.Node, Kind: Store, Receiver: m.ValueOf(target.Expression()), Key: key, Stored: w.Source, Carried: carrying(w.Source)}, true
		}
	}
	v, ok := m.WriteValue(w)
	if !ok || len(v.Path) == 0 {
		return Effect{}, false
	}
	v.Path = v.Path[:len(v.Path)-1]
	return Effect{Node: w.Node, Kind: Change, Receiver: v, Stored: w.Source, Carried: carrying(w.Source)}, true
}

// Anchor is data a function's caller gives it: its receiver (Parameter -1) or a parameter, then a property path.
type Anchor struct {
	Parameter int
	Path      []string
}

// SummaryEffect is an effect a function makes on data its caller gives it: Target is what it acts on, Key and Stored
// the key and stored value when they are such data as a whole, and Carried the data the effect may put into its target.
type SummaryEffect struct {
	Kind    EffectKind
	Target  Anchor
	Key     *Anchor
	Stored  *Anchor
	Carried []int
}

// Summary lists the effects a project function with a body makes on its receiver and parameters, including through
// the calls it makes, read with EffectsIn on its body. Effects on its own locals, on module state, or on a parameter it
// rebinds are not listed. A function reached again while it is summarized gives nothing on that call.
func (m *Analyzer) Summary(fn *ast.Node) []SummaryEffect {
	if summary, ok := m.summaries[fn]; ok {
		return summary
	}
	m.summaries[fn] = nil
	parameters := map[*ast.Symbol]int{}
	for i, parameter := range fn.Parameters() {
		if symbol := parameter.Symbol(); symbol != nil && parameter.Name().Kind == ast.KindIdentifier && !m.assigned(symbol) {
			parameters[symbol] = i
		}
	}
	anchor := func(v Value) (Anchor, bool) {
		if !v.Named() || v.Partial {
			return Anchor{}, false
		}
		if v.This != nil && v.This == fn {
			return Anchor{Parameter: -1, Path: v.Path}, true
		}
		if i, ok := parameters[v.Root]; ok && v.Root != nil {
			return Anchor{Parameter: i, Path: v.Path}, true
		}
		return Anchor{}, false
	}
	whole := func(n *ast.Node) *Anchor {
		if n == nil {
			return nil
		}
		if a, ok := anchor(m.ValueOf(n)); ok {
			return &a
		}
		return nil
	}
	summary := []SummaryEffect{}
	for _, e := range m.EffectsIn([]*ast.Node{fn.Body()}) {
		target, ok := anchor(e.Receiver)
		if !ok {
			continue
		}
		effect := SummaryEffect{Kind: e.Kind, Target: target, Key: whole(e.Key), Stored: whole(e.Stored)}
		for _, carried := range e.Carried {
			for _, read := range m.anchorsRead(carried, anchor) {
				if !slices.Contains(effect.Carried, read) {
					effect.Carried = append(effect.Carried, read)
				}
			}
		}
		summary = append(summary, effect)
	}
	m.summaries[fn] = summary
	return summary
}

// anchorsRead returns the parameters, or -1 for the receiver, whose data n reads, outside the functions it declares.
func (m *Analyzer) anchorsRead(n *ast.Node, anchor func(Value) (Anchor, bool)) []int {
	found := []int{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindIdentifier || c.Kind == ast.KindThisKeyword {
			if a, ok := anchor(m.ValueOf(c)); ok && !slices.Contains(found, a.Parameter) {
				found = append(found, a.Parameter)
			}
			return false
		}
		c.ForEachChild(walk)
		return false
	}
	walk(n)
	return found
}

// followed returns the project function with a body that a call or construction resolves to.
func (m *Analyzer) followed(n *ast.Node) *ast.Node {
	if n.Kind != ast.KindCallExpression && n.Kind != ast.KindNewExpression {
		return nil
	}
	declaration := Implementation(m.ctx.TypeChecker.GetResolvedSignature(n))
	if declaration == nil || !ast.IsFunctionLike(declaration) || declaration.Body() == nil || !ProjectFile(ast.GetSourceFileOfNode(declaration)) {
		return nil
	}
	return declaration
}

// applied returns the effects of a call to fn: its summary, with the receiver being what the call is made on (this for
// super) and parameters the call's arguments, up to the first spread. A construction's receiver is the new object, so
// effects on it are not listed.
func (m *Analyzer) applied(call, fn *ast.Node) []Effect {
	var receiver *Value
	var object *ast.Node
	if call.Kind == ast.KindCallExpression {
		switch callee := Unwrap(call.Expression()); {
		case callee.Kind == ast.KindSuperKeyword:
			receiver = &Value{This: ast.GetThisContainer(call, false, false), Path: []string{}}
		case callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression:
			object = Unwrap(callee.Expression())
			v := m.ValueOf(object)
			if object.Kind == ast.KindSuperKeyword {
				v, object = Value{This: ast.GetThisContainer(call, false, false), Path: []string{}}, nil
			}
			receiver = &v
		}
	}
	arguments := call.Arguments()
	if spread := slices.IndexFunc(arguments, func(a *ast.Node) bool { return a.Kind == ast.KindSpreadElement }); spread >= 0 {
		arguments = arguments[:spread]
	}
	given := func(parameter int) (Value, *ast.Node, bool) {
		if parameter < 0 {
			if receiver == nil {
				return Value{}, nil, false
			}
			return *receiver, object, true
		}
		if parameter >= len(arguments) {
			return Value{}, nil, false
		}
		return m.ValueOf(arguments[parameter]), arguments[parameter], true
	}
	whole := func(a *Anchor) *ast.Node {
		if a == nil || len(a.Path) > 0 {
			return nil
		}
		_, node, _ := given(a.Parameter)
		return node
	}
	effects := []Effect{}
	for _, s := range m.Summary(fn) {
		v, _, ok := given(s.Target.Parameter)
		if !ok {
			continue
		}
		v.Node, v.Path = nil, append(slices.Clone(v.Path), s.Target.Path...)
		effect := Effect{Node: call, Kind: s.Kind, Receiver: v, Key: whole(s.Key), Stored: whole(s.Stored)}
		for _, parameter := range s.Carried {
			if _, node, ok := given(parameter); ok && node != nil {
				effect.Carried = append(effect.Carried, node)
			}
		}
		effects = append(effects, effect)
	}
	return effects
}
