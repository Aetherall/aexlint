package model

import (
	"maps"
	"slices"
	"strings"
	_ "unsafe"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// flowTypeOfReference is the checker's own narrowing: the type of reference where flowNode is reached.
//
//go:linkname flowTypeOfReference github.com/microsoft/typescript-go/internal/checker.(*Checker).getFlowTypeOfReferenceEx
func flowTypeOfReference(c *checker.Checker, reference *ast.Node, declaredType *checker.Type, initialType *checker.Type, flowContainer *ast.Node, flowNode *ast.FlowNode) *checker.Type

// place is data a definition gives a value to: a variable or this, then a property path.
type place struct {
	root *ast.Symbol
	this *ast.Node
	path string
}

func placeOf(v Value) (place, bool) {
	if !v.Named() || v.Partial || v.Root == nil && v.This == nil {
		return place{}, false
	}
	return place{root: v.Root, this: v.This, path: strings.Join(v.Path, "\x00")}, true
}

func (m *Analyzer) placeOfWrite(w Write) (place, bool) {
	if !w.Rebinds() {
		return place{}, false
	}
	v, ok := m.WriteValue(w)
	if !ok {
		return place{}, false
	}
	return placeOf(v)
}

// Def is a definition of a place reaching a point: a write rebinding it, the declaration initializing its variable, or,
// with Before set, the value it held before the code being read.
type Def struct {
	Write       Write
	Declaration *ast.Node
	Before      bool
}

var before = Def{Before: true}

// defs maps the places a way defines to the definitions reaching its end. A place it does not hold keeps the value it
// held before.
type defs map[place][]Def

func (d defs) at(p place) []Def {
	if found, ok := d[p]; ok {
		return found
	}
	return []Def{before}
}

func union(a, b []Def) []Def {
	joined := slices.Clone(a)
	for _, d := range b {
		if !slices.Contains(joined, d) {
			joined = append(joined, d)
		}
	}
	return joined
}

// substitute replaces the value held before in found with the definitions of it.
func substitute(found, of []Def) []Def {
	result := []Def{}
	for _, d := range found {
		if d.Before {
			result = union(result, of)
		} else {
			result = union(result, []Def{d})
		}
	}
	return result
}

func joinDefs(a, b defs) defs {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	joined := defs{}
	for _, side := range []defs{a, b} {
		for p := range side {
			if _, ok := joined[p]; !ok {
				joined[p] = union(a.at(p), b.at(p))
			}
		}
	}
	return joined
}

// composeDefs returns the definitions reaching the end of a way that follows prefix.
func composeDefs(prefix, next defs) defs {
	if len(next) == 0 {
		return prefix
	}
	if len(prefix) == 0 {
		return next
	}
	composed := maps.Clone(prefix)
	for p, found := range next {
		composed[p] = substitute(found, prefix.at(p))
	}
	return composed
}

// defined returns the definitions writes made in order leave: a rebinding of a place replaces the earlier ones, unless
// it may not run, in a branch of a conditional expression or the right side of a logical operator, or maybe is set.
func (c *control) defined(writes []Write, maybe bool) defs {
	if c.place == nil {
		return nil
	}
	var found defs
	for _, w := range writes {
		p, ok := c.place(w)
		if !ok {
			continue
		}
		if found == nil {
			found = defs{}
		}
		if maybe || conditional(w.Node) {
			found[p] = union(found.at(p), []Def{{Write: w}})
		} else {
			found[p] = []Def{{Write: w}}
		}
	}
	return found
}

// conditional reports whether n may not run when the statement holding it does.
func conditional(n *ast.Node) bool {
	for c, p := n, n.Parent; p != nil && !ast.IsStatement(p) && !ast.IsFunctionLike(p); c, p = p, p.Parent {
		switch {
		case p.Kind == ast.KindConditionalExpression && c != p.AsConditionalExpression().Condition:
			return true
		case p.Kind == ast.KindBinaryExpression && c == p.AsBinaryExpression().Right:
			operator := p.AsBinaryExpression().OperatorToken.Kind
			if ast.IsLogicalOperator(operator) || operator == ast.KindQuestionQuestionToken || ast.IsLogicalOrCoalescingAssignmentOperator(operator) {
				return true
			}
		}
	}
	return false
}

// repeated returns the ending of a loop body that may run after earlier iterations, which reached head.
func (e ending) repeated(head *exit) ending {
	if head == nil || len(head.defs) == 0 {
		return e
	}
	again := func(x *exit) *exit {
		if x == nil {
			return nil
		}
		return &exit{writes: x.writes, defs: joinDefs(x.defs, composeDefs(head.defs, x.defs))}
	}
	all := func(exits map[*ast.Node]*exit) map[*ast.Node]*exit {
		if len(exits) == 0 {
			return exits
		}
		result := map[*ast.Node]*exit{}
		for target, x := range exits {
			result[target] = again(x)
		}
		return result
	}
	return ending{end: again(e.end), breaks: all(e.breaks), continues: all(e.continues), returns: e.returns, throws: e.throws}
}

// Provenance is the kinds of value an expression can give: data that already exists, read from a variable, a
// property, an element or a container's entry; a new object, array, function or class; a constant; the size of a
// collection or string; or a value computed by a call or an operator.
type Provenance uint8

const (
	ExistingValue Provenance = 1 << iota
	FreshValue
	ConstantValue
	SizeValue
	ComputedValue
)

var provenanceNames = []struct {
	kind Provenance
	name string
}{{ExistingValue, "existing data"}, {FreshValue, "a new object"}, {ConstantValue, "a constant"}, {SizeValue, "the size of a collection"}, {ComputedValue, "a computed value"}}

func (p Provenance) String() string {
	names := []string{}
	for _, entry := range provenanceNames {
		if p&entry.kind != 0 {
			names = append(names, entry.name)
		}
	}
	if len(names) == 0 {
		return "nothing"
	}
	return strings.Join(names, " or ")
}

// Differs reports whether p and o hold no computed value and share no kind.
func (p Provenance) Differs(o Provenance) bool {
	known := func(k Provenance) bool { return k != 0 && k&ComputedValue == 0 }
	return known(p) && known(o) && p&o == 0
}

// ProvenanceOf returns the kinds of value n gives. Freshness is read from the expression, since no type states it; a
// constant is an expression whose type is a single value; a const variable gives what its initializer gives, and
// another variable what its definitions reaching n give.
func (m *Analyzer) ProvenanceOf(n *ast.Node) Provenance {
	return m.provenance(n, map[*ast.Symbol]bool{})
}

func (m *Analyzer) provenance(n *ast.Node, seen map[*ast.Symbol]bool) Provenance {
	n = Unwrap(n)
	if n == nil {
		return ConstantValue
	}
	switch n.Kind {
	case ast.KindObjectLiteralExpression, ast.KindArrayLiteralExpression, ast.KindNewExpression, ast.KindFunctionExpression,
		ast.KindArrowFunction, ast.KindClassExpression, ast.KindRegularExpressionLiteral:
		return FreshValue
	}
	if m.unit(n) {
		return ConstantValue
	}
	switch n.Kind {
	case ast.KindConditionalExpression:
		c := n.AsConditionalExpression()
		return m.provenance(c.WhenTrue, seen) | m.provenance(c.WhenFalse, seen)
	case ast.KindBinaryExpression:
		b := n.AsBinaryExpression()
		switch operator := b.OperatorToken.Kind; {
		case ast.IsLogicalOperator(operator) || operator == ast.KindQuestionQuestionToken:
			return m.provenance(b.Left, seen) | m.provenance(b.Right, seen)
		case operator == ast.KindCommaToken || operator == ast.KindEqualsToken:
			return m.provenance(b.Right, seen)
		}
	case ast.KindPropertyAccessExpression:
		if m.size(n) {
			return SizeValue
		}
		if !m.computed(n) {
			return ExistingValue
		}
	case ast.KindElementAccessExpression, ast.KindThisKeyword:
		return ExistingValue
	case ast.KindCallExpression:
		if _, ok := m.LookupOf(n); ok {
			return ExistingValue
		}
	case ast.KindIdentifier:
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
		if symbol == nil || seen[symbol] {
			return ExistingValue
		}
		d := symbol.ValueDeclaration
		if d == nil || d.Kind != ast.KindVariableDeclaration {
			return ExistingValue
		}
		seen[symbol] = true
		defer delete(seen, symbol)
		if ast.IsVarConst(d) {
			return m.provenance(d.Initializer(), seen)
		}
		var kinds Provenance
		for _, def := range m.reaching(n, Value{Root: symbol, Path: []string{}}) {
			kinds |= m.defProvenance(def, seen)
		}
		return kinds
	}
	return ComputedValue
}

// unit reports whether n's type is a single value: a literal, an enum member, null, undefined or a unique symbol.
func (m *Analyzer) unit(n *ast.Node) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	return t != nil && t.Flags()&checker.TypeFlagsUnit != 0 && t.Flags()&checker.TypeFlagsUnion == 0
}

// reaching returns the definitions of v reaching the statement holding n, as Before reads them.
func (m *Analyzer) reaching(n *ast.Node, v Value) []Def {
	for statement := n; statement.Parent != nil; statement = statement.Parent {
		if ast.IsFunctionLike(statement) {
			break
		}
		if slices.Contains(statementList(statement.Parent), statement) {
			return m.Before(statement, v)
		}
	}
	return []Def{before}
}

// libraryMember returns the global collection types declaring the member name names, one per member of a union
// receiver, and the member's name, when every declaration belongs to a global type the tables below know. Global types
// are resolved in the global scope and compared by declaration, so a project type of the same name is not one.
func (m *Analyzer) libraryMember(name *ast.Node) ([]string, string, bool) {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(name)
	if symbol == nil || len(symbol.Declarations) == 0 {
		return nil, "", false
	}
	owners := []string{}
	for _, d := range symbol.Declarations {
		at := slices.IndexFunc(collectionTypes, func(global string) bool {
			g := m.globalType(global)
			return g != nil && slices.Contains(g.Declarations, d.Parent)
		})
		if at < 0 {
			return nil, "", false
		}
		if !slices.Contains(owners, collectionTypes[at]) {
			owners = append(owners, collectionTypes[at])
		}
	}
	return owners, symbol.Name, true
}

// agreed returns what table says of member for every owner, when they all say the same.
func agreed[T comparable](owners []string, member string, table map[string]map[string]T) (T, bool) {
	var found T
	for i, owner := range owners {
		entry, ok := table[owner][member]
		if !ok || i > 0 && entry != found {
			var none T
			return none, false
		}
		found = entry
	}
	return found, len(owners) > 0
}

// collectionTypes are the global types whose members the tables below describe.
var collectionTypes = []string{"Array", "ReadonlyArray", "String", "Map", "ReadonlyMap", "WeakMap", "Set", "ReadonlySet", "WeakSet"}

// globalType resolves a type in the global scope, once per invocation.
func (m *Analyzer) globalType(name string) *ast.Symbol {
	if m.globals == nil {
		m.globals = map[string]*ast.Symbol{}
	}
	if symbol, ok := m.globals[name]; ok {
		return symbol
	}
	symbol := m.ctx.TypeChecker.ResolveName(name, nil, ast.SymbolFlagsType, false)
	m.globals[name] = symbol
	return symbol
}

// method reports whether every declaration of the member name names is a method, so calling it acts on its receiver:
// not a property holding a function, nor a function of a namespace or module.
func (m *Analyzer) method(name *ast.Node) bool {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(name)
	return symbol != nil && len(symbol.Declarations) > 0 && !slices.ContainsFunc(symbol.Declarations, func(d *ast.Node) bool {
		return d.Kind != ast.KindMethodDeclaration && d.Kind != ast.KindMethodSignature
	})
}

// sizes names the member giving the size of each global collection, and of strings.
var sizes = map[string]map[string]bool{"Array": {"length": true}, "ReadonlyArray": {"length": true}, "String": {"length": true},
	"Map": {"size": true}, "ReadonlyMap": {"size": true}, "Set": {"size": true}, "ReadonlySet": {"size": true}}

// size reports whether a property read gives the size of a global collection or of a string.
func (m *Analyzer) size(n *ast.Node) bool {
	owners, name, ok := m.libraryMember(n.Name())
	_, known := agreed(owners, name, sizes)
	return ok && known
}

// Lookup is a read of an entry of a container: Container is the container, and Key selects the entry, when one
// expression does.
type Lookup struct {
	Container Value
	Key       *ast.Node
}

// lookups names the default library methods reading an entry, and whether their first argument is its key.
var lookups = map[string]map[string]bool{
	"Map": {"get": true}, "ReadonlyMap": {"get": true}, "WeakMap": {"get": true},
	"Array": {"at": true, "find": false, "findLast": false}, "ReadonlyArray": {"at": true, "find": false, "findLast": false},
}

// LookupOf reads n as a read of a container's entry: an element read, or a default library method returning an entry
// of a map or array.
func (m *Analyzer) LookupOf(n *ast.Node) (Lookup, bool) {
	n = Unwrap(n)
	switch n.Kind {
	case ast.KindElementAccessExpression:
		return Lookup{Container: m.ValueOf(n.Expression()), Key: n.AsElementAccessExpression().ArgumentExpression}, true
	case ast.KindCallExpression:
		callee := Unwrap(n.Expression())
		if callee.Kind != ast.KindPropertyAccessExpression {
			return Lookup{}, false
		}
		owners, name, ok := m.libraryMember(callee.Name())
		keyed, found := agreed(owners, name, lookups)
		if !ok || !found {
			return Lookup{}, false
		}
		lookup := Lookup{Container: m.ValueOf(callee.Expression())}
		if arguments := n.Arguments(); keyed && len(arguments) > 0 {
			lookup.Key = arguments[0]
		}
		return lookup, true
	}
	return Lookup{}, false
}

// EffectKind is what an effect does to the object it acts on.
type EffectKind string

const (
	// Store puts a value under a key: Map.set, WeakMap.set, or an element assignment with a computed key.
	Store EffectKind = "store"
	// Append adds a value: Array.push, Array.unshift, Set.add or WeakSet.add.
	Append EffectKind = "append"
	// Remove takes entries out: delete, clear, pop, shift or splice.
	Remove EffectKind = "remove"
	// Change changes it otherwise: a project or unknown method run as a command, a library method reordering or
	// filling an array, or an assignment or update of data inside it.
	Change EffectKind = "change"
)

// Effect is what a write does to the object it acts on: Receiver is that object, Key where a store or removal acts, and
// Stored the value it puts there, when the write names them.
type Effect struct {
	Node     *ast.Node
	Kind     EffectKind
	Receiver Value
	Key      *ast.Node
	Stored   *ast.Node
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

// EffectOf reads what a write does to the object it acts on. A rebinding of a variable or this is a definition, not an
// effect. A call has one only when its callee is a method: a default library method other than the collection methods
// the table knows, a project method whose result is used, a function held by a property or namespace, or a callee the
// checker cannot resolve has none.
func (m *Analyzer) EffectOf(w Write) (Effect, bool) {
	argument := func(arguments []*ast.Node, i int) *ast.Node {
		if i < 0 || i >= len(arguments) {
			return nil
		}
		return arguments[i]
	}
	target := Unwrap(w.Target)
	if w.Kind == Mutation {
		effect := Effect{Node: w.Node, Kind: Change, Receiver: m.ValueOf(target)}
		callee := Unwrap(w.Node.Expression())
		name := callee
		if callee.Kind == ast.KindPropertyAccessExpression {
			name = callee.Name()
		}
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(name)
		if owners, member, ok := m.libraryMember(name); ok {
			known, found := agreed(owners, member, libraryEffects)
			if !found {
				return Effect{}, false
			}
			arguments := w.Node.Arguments()
			effect.Kind, effect.Key, effect.Stored = known.kind, argument(arguments, known.key), argument(arguments, known.stored)
			return effect, true
		}
		if utils.IsSymbolFromDefaultLibrary(m.ctx.Program, symbol) {
			return Effect{}, false
		}
		return effect, w.Discarded() && m.method(name)
	}
	if target.Kind == ast.KindElementAccessExpression {
		key := Unwrap(target.AsElementAccessExpression().ArgumentExpression)
		if !ast.IsStringLiteralLike(key) && key.Kind != ast.KindNumericLiteral {
			return Effect{Node: w.Node, Kind: Store, Receiver: m.ValueOf(target.Expression()), Key: key, Stored: w.Source}, true
		}
	}
	v, ok := m.WriteValue(w)
	if !ok || len(v.Path) == 0 {
		return Effect{}, false
	}
	v.Path = v.Path[:len(v.Path)-1]
	return Effect{Node: w.Node, Kind: Change, Receiver: v, Stored: w.Source}, true
}

// EffectsIn lists the effects of statements in order, outside the functions they declare.
func (m *Analyzer) EffectsIn(statements []*ast.Node) []Effect {
	effects := []Effect{}
	for _, w := range writesIn(statements, false) {
		if effect, ok := m.EffectOf(w); ok {
			effects = append(effects, effect)
		}
	}
	return effects
}

// Arm is one way into the point after a decision where its ways rejoin: Branch is the then or else statement, the case
// clause the way enters, or the branch of a conditional expression, and nil for the way no branch takes. Its entry is
// the checker's flow node where the arm starts, so the checker narrows values there as it does for code in the arm. An
// arm of an if statement or conditional expression also holds the condition it is entered by, found truth, and the
// flow node before the decision.
type Arm struct {
	Branch *ast.Node
	entry  *ast.FlowNode
	test   *ast.Node
	truth  bool
	start  *ast.FlowNode
	defs   defs
}

// flowAt returns the checker's flow node where a statement starts, or nil when it cannot be reached.
func flowAt(statement *ast.Node) *ast.FlowNode {
	if statement == nil || statement.FlowNodeData() == nil {
		return nil
	}
	return statement.FlowNodeData().FlowNode
}

// branching returns the flow node where condition, reached from start, has been found truth.
func branching(start *ast.FlowNode, condition *ast.Node, truth bool) *ast.FlowNode {
	if start == nil {
		return nil
	}
	flags := ast.FlowFlagsFalseCondition
	if truth {
		flags = ast.FlowFlagsTrueCondition
	}
	return &ast.FlowNode{Flags: flags, Node: condition, Antecedent: start}
}

// Merge is where the ways through a decision that reach its end rejoin: an if or switch statement, or a conditional
// expression run as a statement. Ways leaving by return, throw, break or continue do not reach it, except a break
// leaving the switch itself.
type Merge struct {
	Decision *ast.Node
	Arms     []Arm
}

// MergeOf reads the merge of a decision.
func (m *Analyzer) MergeOf(n *ast.Node) (*Merge, bool) {
	c := m.writeControl()
	merge := &Merge{Decision: n}
	switch {
	case n.Kind == ast.KindIfStatement:
		s := n.AsIfStatement()
		condition := c.effected(s.Expression)
		for _, arm := range []struct {
			branch *ast.Node
			truth  bool
		}{{s.ThenStatement, true}, {s.ElseStatement, false}} {
			if end := c.of(arm.branch).after(condition).end; end != nil {
				merge.Arms = append(merge.Arms, Arm{Branch: arm.branch, entry: branching(flowAt(n), s.Expression, arm.truth),
					test: s.Expression, truth: arm.truth, start: flowAt(n), defs: end.defs})
			}
		}
	case n.Kind == ast.KindSwitchStatement:
		s := n.AsSwitchStatement()
		entry := c.effected(s.Expression)
		clauses := s.CaseBlock.AsCaseBlock().Clauses.Nodes
		for i, clause := range clauses {
			statements := []*ast.Node{}
			for _, later := range clauses[i:] {
				statements = append(statements, later.Statements()...)
			}
			e := c.list(statements).after(entry)
			if end := join(e.end, e.breaks[n]); end != nil {
				var start *ast.FlowNode
				if len(statements) > 0 {
					start = flowAt(statements[0])
				}
				merge.Arms = append(merge.Arms, Arm{Branch: clause, entry: start, defs: end.defs})
			}
		}
		covered := slices.ContainsFunc(clauses, func(c *ast.Node) bool { return c.Kind == ast.KindDefaultClause }) || m.Exhaustive(n)
		if !covered || len(clauses) == 0 {
			merge.Arms = append(merge.Arms, Arm{defs: entry.defs})
		}
	case n.Kind == ast.KindExpressionStatement && Unwrap(n.Expression()).Kind == ast.KindConditionalExpression:
		conditional := Unwrap(n.Expression()).AsConditionalExpression()
		condition := c.effected(conditional.Condition)
		for _, arm := range []struct {
			branch *ast.Node
			truth  bool
		}{{conditional.WhenTrue, true}, {conditional.WhenFalse, false}} {
			found := composeDefs(condition.defs, c.defined(c.effects(arm.branch), false))
			merge.Arms = append(merge.Arms, Arm{Branch: arm.branch, entry: branching(flowAt(n), conditional.Condition, arm.truth),
				test: conditional.Condition, truth: arm.truth, start: flowAt(n), defs: found})
		}
	default:
		return nil, false
	}
	return merge, true
}

// MissingOn reports whether the checker finds the place reference names undefined or null where arm starts. For an
// arm entered by a condition, the checker narrows the type the place holds before the decision, with undefined and
// null added, by that condition alone: an arm entered only when a value its type says is always present is missing,
// such as a record's entry, finds it missing rather than unreachable. A generic type is read as its constraint, which
// narrowing can tell present or missing. A case clause reads the type the checker narrows
// there.
func (m *Analyzer) MissingOn(arm Arm, reference *ast.Node) bool {
	if arm.entry == nil || reference == nil {
		return false
	}
	tc := m.ctx.TypeChecker
	reference = Unwrap(reference)
	name := reference
	if name.Kind == ast.KindPropertyAccessExpression {
		name = name.Name()
	}
	symbol := tc.GetSymbolAtLocation(name)
	if symbol == nil {
		return false
	}
	declared := tc.GetTypeOfSymbol(symbol)
	narrowed := func(held *checker.Type, flow *ast.FlowNode) *checker.Type {
		return flowTypeOfReference(tc, reference, held, held, nil, flow)
	}
	t := narrowed(declared, arm.entry)
	if arm.test != nil {
		held := narrowed(declared, arm.start)
		if constraint := checker.Checker_getBaseConstraintOfType(tc, held); held.Flags()&checker.TypeFlagsInstantiable != 0 && constraint != nil {
			held = constraint
		}
		held = tc.GetUnionType([]*checker.Type{held, tc.GetUndefinedType(), tc.GetNullType()})
		t = narrowed(held, branching(&ast.FlowNode{Flags: ast.FlowFlagsStart}, arm.test, arm.truth))
	}
	return t != nil && t.Flags()&checker.TypeFlagsNever == 0 && !m.present(t)
}

// Phi is a value where a decision's ways rejoin: the definitions reaching it along each arm of the merge, and whether
// it is carried: data of this, or a variable declared outside the innermost loop around the decision in its function,
// whose value outlives this pass. Reference is an expression naming it that some arm writes, for the checker to narrow.
type Phi struct {
	Value     Value
	Inputs    [][]Def
	Carried   bool
	Reference *ast.Node
}

// PhiOf reads v where merge's ways rejoin.
func (m *Analyzer) PhiOf(merge *Merge, v Value) (Phi, bool) {
	p, ok := placeOf(v)
	if !ok {
		return Phi{}, false
	}
	phi := Phi{Value: v, Carried: carried(merge.Decision, v)}
	for _, arm := range merge.Arms {
		input := arm.defs.at(p)
		phi.Inputs = append(phi.Inputs, input)
		for _, d := range input {
			if phi.Reference == nil && d.Write.Target != nil {
				phi.Reference = d.Write.Target
			}
		}
	}
	return phi, true
}

// Joins reports whether some arm defines the value.
func (p Phi) Joins() bool {
	return slices.ContainsFunc(p.Inputs, func(input []Def) bool { return !slices.Equal(input, []Def{before}) })
}

func carried(decision *ast.Node, v Value) bool {
	if v.This != nil {
		return true
	}
	if v.Root == nil || v.Root.ValueDeclaration == nil {
		return false
	}
	for p := decision.Parent; p != nil && !ast.IsFunctionLike(p); p = p.Parent {
		if ast.IsIterationStatement(p, false) {
			return !encloses(p.Statement(), v.Root.ValueDeclaration)
		}
	}
	return false
}

// Before returns the definitions of v reaching decision from the statements before it in each statement list around
// it, back to the statement declaring its variable. It keeps the value held before, unresolved, past a loop around the
// decision or the start of its function.
func (m *Analyzer) Before(decision *ast.Node, v Value) []Def {
	p, ok := placeOf(v)
	if !ok {
		return []Def{before}
	}
	found := []Def{before}
	for current := decision; current.Parent != nil; current = current.Parent {
		if ast.IsFunctionLike(current.Parent) || ast.IsIterationStatement(current.Parent, false) {
			break
		}
		list := statementList(current.Parent)
		i := slices.Index(list, current)
		if i < 0 {
			continue
		}
		declared := -1
		if d := declarationOf(v); d != nil {
			declared = slices.IndexFunc(list[:i], func(s *ast.Node) bool { return d.Parent != nil && d.Parent.Parent == s })
		}
		if end := m.writeControl().list(list[declared+1 : i]).end; end != nil {
			found = substitute(found, end.defs.at(p))
		}
		if declared >= 0 {
			return substitute(found, []Def{{Declaration: declarationOf(v)}})
		}
	}
	return found
}

// declarationOf returns the variable declaration v is the whole of.
func declarationOf(v Value) *ast.Node {
	if v.Root == nil || len(v.Path) > 0 || v.Root.ValueDeclaration == nil || v.Root.ValueDeclaration.Kind != ast.KindVariableDeclaration {
		return nil
	}
	return v.Root.ValueDeclaration
}

// DefProvenance returns the kinds of value a definition gives. The value held before is existing data, a destructured
// part or loop variable is read from other data, and an update computes its value.
func (m *Analyzer) DefProvenance(d Def) Provenance {
	return m.defProvenance(d, map[*ast.Symbol]bool{})
}

func (m *Analyzer) defProvenance(d Def, seen map[*ast.Symbol]bool) Provenance {
	switch {
	case d.Before:
		return ExistingValue
	case d.Declaration != nil:
		return m.provenance(d.Declaration.Initializer(), seen)
	case d.Write.Kind == Update:
		return ComputedValue
	case d.Write.Source != nil:
		return m.provenance(d.Write.Source, seen)
	}
	return ExistingValue
}

// InputProvenance returns the kinds of value phi gets from the arm at index of merge, reading the value held before
// the decision through Before.
func (m *Analyzer) InputProvenance(merge *Merge, phi Phi, arm int) Provenance {
	var kinds Provenance
	for _, d := range phi.Inputs[arm] {
		if !d.Before {
			kinds |= m.DefProvenance(d)
			continue
		}
		for _, held := range m.Before(merge.Decision, phi.Value) {
			kinds |= m.DefProvenance(held)
		}
	}
	return kinds
}

// Divergence is an effect after a decision whose destination changes kind with the way into the decision's merge:
// Through is a value the arms define, not carried, that the effect acts on or selects its key with, and Kinds are the
// kinds of value it holds on each arm that does not find it missing. Two of them share no kind.
type Divergence struct {
	Effect  Effect
	Through Value
	Kinds   []Provenance
}

// Divergences lists the effects of the statements after a decision, in its statement list, whose destination diverges
// with the way into its merge. An effect after the value is rebound does not see the merge.
func (m *Analyzer) Divergences(n *ast.Node) []Divergence {
	merge, ok := m.MergeOf(n)
	if !ok || len(merge.Arms) < 2 || n.Parent == nil {
		return nil
	}
	list := statementList(n.Parent)
	i := slices.Index(list, n)
	if i < 0 {
		return nil
	}
	after := list[i+1:]
	found := []Divergence{}
	for _, effect := range m.EffectsIn(after) {
		destinations := []Value{effect.Receiver}
		if effect.Key != nil {
			destinations = append(destinations, m.ValueOf(effect.Key))
		}
		for _, destination := range destinations {
			phi, ok := m.steering(merge, destination)
			if !ok || m.rebound(after, phi.Value, effect.Node) {
				continue
			}
			kinds := []Provenance{}
			for index, arm := range merge.Arms {
				if !m.MissingOn(arm, phi.Reference) {
					kinds = append(kinds, m.InputProvenance(merge, phi, index))
				}
			}
			if slices.ContainsFunc(kinds, func(k Provenance) bool { return slices.ContainsFunc(kinds, k.Differs) }) {
				found = append(found, Divergence{Effect: effect, Through: phi.Value, Kinds: kinds})
				break
			}
		}
	}
	return found
}

// steering returns the value destination is read from that merge's arms define and that is not carried: destination
// itself or data it is read from.
func (m *Analyzer) steering(merge *Merge, destination Value) (Phi, bool) {
	for length := len(destination.Path); length >= 0; length-- {
		v := destination
		v.Node, v.Path = nil, slices.Clone(destination.Path[:length])
		if phi, ok := m.PhiOf(merge, v); ok && phi.Joins() {
			return phi, !phi.Carried
		}
	}
	return Phi{}, false
}

// rebound reports whether statements rebind v before node.
func (m *Analyzer) rebound(statements []*ast.Node, v Value, node *ast.Node) bool {
	p, _ := placeOf(v)
	return slices.ContainsFunc(writesIn(statements, false), func(w Write) bool {
		written, ok := m.placeOfWrite(w)
		return ok && written == p && w.Node.End() <= node.Pos()
	})
}

// Defined lists the values merge's arms define that outlive the decision: data of this, or of a variable declared
// outside it, in the order of their first definition.
func (m *Analyzer) Defined(merge *Merge) []Value {
	type defined struct {
		value    Value
		position int
	}
	found := map[place]defined{}
	for _, arm := range merge.Arms {
		for p, ds := range arm.defs {
			for _, d := range ds {
				if d.Write.Node == nil {
					continue
				}
				v, ok := m.WriteValue(d.Write)
				if !ok || v.Root != nil && v.Root.ValueDeclaration != nil && encloses(merge.Decision, v.Root.ValueDeclaration) {
					continue
				}
				if earlier, ok := found[p]; !ok || d.Write.Node.Pos() < earlier.position {
					v.Node = nil
					found[p] = defined{v, d.Write.Node.Pos()}
				}
			}
		}
	}
	sorted := slices.SortedFunc(maps.Values(found), func(a, b defined) int { return a.position - b.position })
	values := []Value{}
	for _, d := range sorted {
		values = append(values, d.value)
	}
	return values
}
