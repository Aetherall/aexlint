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

// argumentEffect is the effect of a default library function acting on one of its arguments: the position of that
// argument, and of the first argument whose data it may carry into it, or -1.
type argumentEffect struct {
	kind            EffectKind
	target, carried int
}

// argumentEffects names the default library functions acting on an argument, by the global interface or namespace
// declaring them.
var argumentEffects = map[string]map[string]argumentEffect{
	"ObjectConstructor": {"assign": {Change, 0, 1}, "defineProperty": {Change, 0, 2}, "defineProperties": {Change, 0, 1},
		"setPrototypeOf": {Change, 0, 1}},
	"Reflect": {"set": {Change, 0, 2}, "defineProperty": {Change, 0, 2}, "deleteProperty": {Remove, 0, -1},
		"setPrototypeOf": {Change, 0, 1}},
}

// argumentEffectOf returns the effect a call to a default library function acting on an argument makes.
func (m *Analyzer) argumentEffectOf(call, name *ast.Node) (Effect, bool) {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(name)
	if symbol == nil || len(symbol.Declarations) == 0 {
		return Effect{}, false
	}
	d := symbol.Declarations[0]
	for owner, members := range argumentEffects {
		known, ok := members[symbol.Name]
		if !ok || !m.ownedBy(d, owner) {
			continue
		}
		arguments := call.Arguments()
		if known.target >= len(arguments) || slices.ContainsFunc(arguments[:known.target+1], func(a *ast.Node) bool { return a.Kind == ast.KindSpreadElement }) {
			return Effect{}, false
		}
		effect := Effect{Node: call, Kind: known.kind, Receiver: m.ValueOf(arguments[known.target])}
		if known.carried >= 0 && known.carried < len(arguments) {
			effect.Carried = carrying(arguments[known.carried:]...)
		}
		return effect, true
	}
	return Effect{}, false
}

// ownedBy reports whether a member declaration belongs to the global interface or namespace named owner.
func (m *Analyzer) ownedBy(d *ast.Node, owner string) bool {
	parent := d.Parent
	if parent != nil && parent.Kind == ast.KindModuleBlock {
		parent = parent.Parent
	}
	for _, flags := range []ast.SymbolFlags{ast.SymbolFlagsType, ast.SymbolFlagsNamespace} {
		if global := m.ctx.TypeChecker.ResolveName(owner, nil, flags, false); global != nil && slices.Contains(global.Declarations, parent) {
			return true
		}
	}
	return false
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
// call to a method whose body is unknown, run as a command, changes its receiver and may carry its arguments into it.
// The default library functions acting on an argument, such as Object.assign, change that argument (see
// argumentEffects); another default library method other than the collection methods the table knows, a function held
// by a property or namespace, or a callee the checker cannot resolve has none. An assignment through a project setter
// with a body makes the effects of the setter's summary, on the object and the assigned value. A rebinding of a
// variable or this is a definition, not an effect.
func (m *Analyzer) EffectsAt(n *ast.Node) []Effect {
	if fn := m.followed(n); fn != nil {
		return m.applied(n, fn)
	}
	effects := []Effect{}
	for _, w := range WritesAt(n) {
		if target := Unwrap(w.Target); w.Kind == Assignment && target.Kind == ast.KindPropertyAccessExpression {
			if setter := m.setter(target); setter != nil {
				effects = append(effects, m.setterEffects(w, target, setter)...)
				continue
			}
		}
		if effect, ok := m.effectOf(w); ok {
			effects = append(effects, effect)
		}
	}
	return effects
}

// setterEffects returns the effects an assignment through a project setter makes: those of its summary, on the
// object for its receiver and on the assigned value for its parameter.
func (m *Analyzer) setterEffects(w Write, target, setter *ast.Node) []Effect {
	receiver := m.ValueOf(target.Expression())
	effects := []Effect{}
	for _, s := range m.Summary(setter).Effects {
		var v Value
		switch {
		case s.Target.Parameter < 0:
			v = receiver
		case s.Target.Parameter == 0 && w.Source != nil:
			v = m.ValueOf(w.Source)
		default:
			continue
		}
		v.Node, v.Path = nil, append(slices.Clone(v.Path), s.Target.Path...)
		effect := Effect{Node: w.Node, Kind: s.Kind, Receiver: v}
		if s.Stored != nil && s.Stored.Parameter == 0 && len(s.Stored.Path) == 0 {
			effect.Stored = w.Source
		}
		if slices.Contains(s.Carried, 0) && w.Source != nil {
			effect.Carried = carrying(w.Source)
		}
		effects = append(effects, effect)
	}
	return effects
}

// EffectsIn lists the effects of nodes in order, outside the functions they declare, except those running during the
// expression they are given to (see runsDuring), whose effects count where they are written. An effect such
// a callback makes on the element it is given, or a for-of loop's body on the value it binds, acts on that element of
// the collection (see Element).
func (m *Analyzer) EffectsIn(nodes []*ast.Node) []Effect {
	effects := []Effect{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) {
			if m.runsDuring(n) && n.Body() != nil {
				for _, e := range m.EffectsIn([]*ast.Node{n.Body()}) {
					if m.synchronous(n) {
						e.Receiver = m.throughElement(n, e.Receiver)
					}
					effects = append(effects, e)
				}
			}
			return false
		}
		if ast.IsClassLike(n) {
			return false
		}
		for _, e := range m.EffectsAt(n) {
			e.Receiver = m.throughLoop(e.Receiver)
			effects = append(effects, e)
		}
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
			return m.argumentEffectOf(w.Node, name)
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

// Element is the path segment naming an element of a collection, whichever it is.
const Element = "[]"

// throughLoop reads v as data of a collection when its root is a variable a for-of loop binds to the collection's
// values, an array's entries() included, never rebound: that element of the collection, then v's path.
func (m *Analyzer) throughLoop(v Value) Value {
	if v.Root == nil || v.Root.ValueDeclaration == nil || m.assigned(v.Root) {
		return v
	}
	d := v.Root.ValueDeclaration
	position := -1
	if d.Kind == ast.KindBindingElement && d.Parent.Kind == ast.KindArrayBindingPattern {
		position = slices.Index(d.Parent.AsBindingPattern().Elements.Nodes, d)
		d = d.Parent.Parent
	}
	if d.Kind != ast.KindVariableDeclaration || d.Parent == nil || d.Parent.Parent == nil || d.Parent.Parent.Kind != ast.KindForOfStatement {
		return v
	}
	loop := d.Parent.Parent.AsForInOrOfStatement()
	if loop.Initializer != d.Parent {
		return v
	}
	over, part := m.iteration(loop.Expression)
	if call := Unwrap(loop.Expression); over.Root == nil && over.This == nil && over.Base == nil && part == entryParts && call.Kind == ast.KindCallExpression {
		if callee := Unwrap(call.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
			over = m.ValueOf(callee.Expression())
		}
	}
	if !over.Named() || over.Root == nil && over.This == nil || part == keyPart || part == entryParts && position != 1 || part == valuePart && position >= 0 {
		return v
	}
	over.Node = nil
	over.Path = append(append(slices.Clone(over.Path), Element), v.Path...)
	over.Partial = over.Partial || v.Partial
	return over
}

// throughElement reads v, data a collection method's callback reaches, as data of the collection when its root is the
// element the method gives the callback: that element of the collection, then v's path. The accumulator reduce gives
// its callback is the initial value given to reduce, when one is.
func (m *Analyzer) throughElement(callback *ast.Node, v Value) Value {
	if v.Root == nil || v.Root.ValueDeclaration == nil || v.Root.ValueDeclaration.Kind != ast.KindParameter || v.Root.ValueDeclaration.Parent != callback {
		return v
	}
	call := callback.Parent
	callee := Unwrap(call.Expression())
	owners, name, _ := m.libraryMember(callee.Name())
	position, known := agreed(owners, name, elementCallbacks)
	index := slices.Index(callback.Parameters(), v.Root.ValueDeclaration)
	if known && position == 1 && index == 0 && len(call.Arguments()) > 1 {
		initial := m.ValueOf(call.Arguments()[1])
		if initial.Named() {
			initial.Node = nil
			initial.Path = append(slices.Clone(initial.Path), v.Path...)
			initial.Partial = initial.Partial || v.Partial
			return initial
		}
	}
	if !known || index != position {
		return v
	}
	collection := m.ValueOf(callee.Expression())
	collection.Node = nil
	collection.Path = append(append(slices.Clone(collection.Path), Element), v.Path...)
	collection.Partial = collection.Partial || v.Partial
	return collection
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

// FunctionSummary is what a project function with a body does with data its caller gives it: the effects it makes on
// its receiver and parameters, and the parts of them it reads, including through the calls it makes, read with
// EffectsIn and ReadsIn on its body. Effects and reads of its own locals or of module state, and of a parameter it
// rebinds, are not listed. Escapes lists the parameters, or -1 for the receiver, whose data it may give to code the
// model does not see (see escapes). Opaque is set when it may also run code the model does not see, which may read or
// change any data visible to it (see Unseen), when it uses a variable of another function other than a const
// primitive, whose data it cannot name (see closure), or when it changes data other than its locals, receiver and
// parameters. Throws is set when its body can end by throwing, directly or through a project function it calls that
// throws (see Throwing). Fresh is set when every value it returns is local data of its own (see local), which no one
// else holds once it returns: it is neither async nor a generator, and every return gives a new object or local data.
// A function reached again while it is summarized gives nothing on that call.
type FunctionSummary struct {
	Effects    []SummaryEffect
	Reads      []SummaryRead
	Escapes    []int
	Guarantees []SummaryGuarantee
	Opaque     bool
	Throws     bool
	Fresh      bool
}

// SummaryGuarantee is what holds about a container its caller gives a function once it returns: a size of at least
// Minimum, or holding the key given at Key (see Guarantees).
type SummaryGuarantee struct {
	Container Anchor
	Minimum   int
	Key       *Anchor
}

// SummaryRead is a read a function makes of data its caller gives it: Key is the key of an element read when it is
// such data as a whole, and Size whether only the size of a collection is read.
type SummaryRead struct {
	Target Anchor
	Key    *Anchor
	Size   bool
}

// Summary returns the summary of a project function with a body.
func (m *Analyzer) Summary(fn *ast.Node) FunctionSummary {
	if summary, ok := m.summaries[fn]; ok {
		return summary
	}
	m.summaries[fn] = FunctionSummary{}
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
	summary := FunctionSummary{Effects: []SummaryEffect{}, Reads: []SummaryRead{}}
	for _, e := range m.EffectsIn([]*ast.Node{fn.Body()}) {
		target, ok := anchor(e.Receiver)
		if !ok {
			continue
		}
		effect := SummaryEffect{Kind: e.Kind, Target: target, Key: whole(e.Key), Stored: whole(e.Stored)}
		for _, carried := range e.Carried {
			for _, read := range m.anchorsRead(carried, fn, anchor) {
				if !slices.Contains(effect.Carried, read) {
					effect.Carried = append(effect.Carried, read)
				}
			}
		}
		summary.Effects = append(summary.Effects, effect)
	}
	for _, r := range m.ReadsIn([]*ast.Node{fn.Body()}) {
		if target, ok := anchor(r.Value); ok {
			summary.Reads = append(summary.Reads, SummaryRead{Target: target, Key: whole(r.Key), Size: r.Size})
		}
	}
	outside := func(e Effect) bool {
		_, anchored := anchor(e.Receiver)
		local := e.Receiver.Root != nil && e.Receiver.Root.ValueDeclaration != nil && enclosingFunction(e.Receiver.Root.ValueDeclaration) == fn
		return !anchored && !local
	}
	summary.Opaque = len(m.Unseen([]*ast.Node{fn.Body()}, true)) > 0 || m.closure(fn) || slices.ContainsFunc(m.EffectsIn([]*ast.Node{fn.Body()}), outside)
	summary.Escapes = m.escapes(fn, anchor, outside)
	summary.Guarantees = m.guaranteed(fn, anchor, summary.Effects)
	summary.Throws = m.flowControl().of(fn.Body()).throws || len(m.Throwing([]*ast.Node{fn.Body()})) > 0
	summary.Fresh = m.returnsLocal(fn)
	m.summaries[fn] = summary
	return summary
}

// escapes returns the parameters, or -1 for the receiver, whose data fn may give to code the model does not see: an
// object read from them given to a call or construction that is neither followed nor declared by the default library,
// or to a followed one, or one running a constructor through classes without one (see implicit), whose summary lets
// that operand escape, or yielded; carried by an effect into data that is
// neither local nor given; or referenced by a function fn declares, which may run anywhere, unless a global collection
// method runs it during the call, where it is read as fn's own code.
func (m *Analyzer) escapes(fn *ast.Node, anchor func(Value) (Anchor, bool), outside func(Effect) bool) []int {
	found := []int{}
	add := func(n *ast.Node) {
		if n == nil || m.primitive(n) {
			return
		}
		for _, a := range m.anchorsRead(n, fn, anchor) {
			if !slices.Contains(found, a) {
				found = append(found, a)
			}
		}
	}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) && m.runsDuring(c) && c.Body() != nil {
			walk(c.Body())
			return false
		}
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			m.closedOver(c, anchor, &found)
			return false
		}
		switch c.Kind {
		case ast.KindYieldExpression:
			add(c.Expression())
		case ast.KindCallExpression, ast.KindNewExpression, ast.KindTaggedTemplateExpression:
			var receiver *ast.Node
			if c.Kind == ast.KindCallExpression {
				if callee := Unwrap(c.Expression()); callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression {
					receiver = callee.Expression()
				}
			}
			arguments := operands(c)
			callee := m.followed(c)
			switch {
			case callee != nil:
				for _, escaped := range m.Summary(callee).Escapes {
					if escaped < 0 {
						add(receiver)
						continue
					}
					for i, argument := range arguments {
						if i == escaped || argument.Kind == ast.KindSpreadElement && i <= escaped {
							add(argument)
						}
					}
				}
			default:
				if constructor, ok := m.implicit(c); ok {
					if constructor != nil {
						for _, escaped := range m.Summary(constructor).Escapes {
							for i, argument := range arguments {
								if i == escaped || argument.Kind == ast.KindSpreadElement && i <= escaped {
									add(argument)
								}
							}
						}
					}
				} else if !m.library(c) {
					add(receiver)
					for _, argument := range arguments {
						add(argument)
					}
				}
			}
		}
		c.ForEachChild(walk)
		return false
	}
	fn.Body().ForEachChild(walk)
	for _, e := range m.EffectsIn([]*ast.Node{fn.Body()}) {
		if outside(e) {
			for _, carried := range e.Carried {
				add(carried)
			}
		}
	}
	return found
}

// Throwing lists the calls and constructions nodes make, outside the functions they declare and the try blocks a catch
// clause handles, that resolve to a project function whose summary throws, or that are default library calls running
// a callback during the call that may throw (see synchronous and callingAtOnce); a Promise executor's throw rejects the
// promise instead. Calls the model does not see are not taken
// to throw; they count as unseen code.
func (m *Analyzer) Throwing(nodes []*ast.Node) []*ast.Node {
	found := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) {
			if (m.synchronous(c) || m.callingAtOnce(c)) && c.Body() != nil && (m.flowControl().of(c.Body()).throws || len(m.Throwing([]*ast.Node{c.Body()})) > 0) {
				found = append(found, c.Parent)
			}
			return false
		}
		if ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindTryStatement && c.AsTryStatement().CatchClause != nil {
			try := c.AsTryStatement()
			walk(try.CatchClause)
			if try.FinallyBlock != nil {
				walk(try.FinallyBlock)
			}
			return false
		}
		if fn := m.followed(c); fn != nil && m.Summary(fn).Throws {
			found = append(found, c)
		}
		c.ForEachChild(walk)
		return false
	}
	for _, n := range nodes {
		if n != nil {
			walk(n)
		}
	}
	return found
}

// returnsLocal reports whether every value fn returns is local data of its own (see FunctionSummary.Fresh).
func (m *Analyzer) returnsLocal(fn *ast.Node) bool {
	if ast.GetFunctionFlags(fn)&(ast.FunctionFlagsAsync|ast.FunctionFlagsGenerator) != 0 || fn.Body() == nil {
		return false
	}
	if fn.Body().Kind != ast.KindBlock {
		return m.madeLocally(fn.Body(), fn)
	}
	returns := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindReturnStatement {
			returns = append(returns, c)
		}
		c.ForEachChild(walk)
		return false
	}
	fn.Body().ForEachChild(walk)
	return len(returns) > 0 && m.flowControl().of(fn.Body()).end == nil && !slices.ContainsFunc(returns, func(r *ast.Node) bool {
		return r.Expression() == nil || !m.madeLocally(r.Expression(), fn)
	})
}

// operands returns the arguments of a call or construction, or the expressions of a tagged template.
func operands(c *ast.Node) []*ast.Node {
	if c.Kind != ast.KindTaggedTemplateExpression {
		return c.Arguments()
	}
	template := c.AsTaggedTemplateExpression().Template
	if template.Kind != ast.KindTemplateExpression {
		return nil
	}
	found := []*ast.Node{}
	for _, span := range template.AsTemplateExpression().TemplateSpans.Nodes {
		found = append(found, span.Expression())
	}
	return found
}

// closedOver adds the parameters, or -1 for the receiver, of the function being summarized that inner references.
func (m *Analyzer) closedOver(inner *ast.Node, anchor func(Value) (Anchor, bool), found *[]int) {
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if c.Kind == ast.KindIdentifier || c.Kind == ast.KindThisKeyword {
			if a, ok := anchor(m.ValueOf(c)); ok && !slices.Contains(*found, a.Parameter) {
				*found = append(*found, a.Parameter)
			}
			return false
		}
		c.ForEachChild(walk)
		return false
	}
	inner.ForEachChild(walk)
}

// anchorsRead returns the parameters, or -1 for the receiver, whose data n reads, outside the functions it declares.
func (m *Analyzer) anchorsRead(n, fn *ast.Node, anchor func(Value) (Anchor, bool)) []int {
	found := []int{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindIdentifier || c.Kind == ast.KindThisKeyword {
			if a, ok := anchor(m.elementData(m.ValueOf(c), fn)); ok && !slices.Contains(found, a.Parameter) {
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

// elementData reads v through the callback parameter or loop variable at its root, when that names an element of a
// collection or reduce's initial value (see throughElement and throughLoop), other than a parameter of fn itself.
func (m *Analyzer) elementData(v Value, fn *ast.Node) Value {
	if v.Root != nil && v.Root.ValueDeclaration != nil && v.Root.ValueDeclaration.Kind == ast.KindParameter {
		if callback := v.Root.ValueDeclaration.Parent; callback != fn && m.synchronous(callback) {
			return m.throughElement(callback, v)
		}
	}
	return m.throughLoop(v)
}

// followed returns the project function with a body that a call or construction resolves to, unless it is a method a
// project subclass overrides, which the call may run instead (see overridden), and the call does not name it through
// super.
func (m *Analyzer) followed(n *ast.Node) *ast.Node {
	if n.Kind != ast.KindCallExpression && n.Kind != ast.KindNewExpression {
		return nil
	}
	declaration := Implementation(m.ctx.TypeChecker.GetResolvedSignature(n))
	if declaration == nil || !ast.IsFunctionLike(declaration) || declaration.Body() == nil || !ProjectFile(ast.GetSourceFileOfNode(declaration)) {
		return nil
	}
	if n.Kind == ast.KindCallExpression {
		callee := Unwrap(n.Expression())
		throughSuper := callee.Kind == ast.KindPropertyAccessExpression && Unwrap(callee.Expression()).Kind == ast.KindSuperKeyword
		if !throughSuper && m.overridden(declaration) {
			return nil
		}
	}
	return declaration
}

// overridden reports whether a project class extending, at any depth, the class declaring a method or accessor
// declares an instance member of the same name, which a call through the base may run instead (see overrides).
// Private names cannot be overridden.
func (m *Analyzer) overridden(member *ast.Node) bool {
	if member.Kind != ast.KindMethodDeclaration && member.Kind != ast.KindGetAccessor && member.Kind != ast.KindSetAccessor {
		return false
	}
	if known, ok := m.overriddenMembers[member]; ok {
		return known
	}
	name := member.Name()
	result := member.Parent != nil && ast.IsClassLike(member.Parent) && name != nil && name.Kind == ast.KindIdentifier &&
		len(m.overrides(member.Parent, name.Text())) > 0
	m.overriddenMembers[member] = result
	return result
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
	for _, s := range m.Summary(fn).Effects {
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
