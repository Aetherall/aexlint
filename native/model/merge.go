package model

import (
	"maps"
	"slices"
	"strings"
	_ "unsafe"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
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

// ProvenanceOf returns the kinds of value n gives. Freshness is read from the expression, since no type states it; a
// constant is an expression whose type is a single value; a const variable gives what its initializer gives, and
// another variable or a parameter what its definitions reaching n give. A call, or a getter read, gives what the
// returns of the project function or accessor it resolves to give, its parameters giving what the call's arguments
// give.
func (m *Analyzer) ProvenanceOf(n *ast.Node) Provenance {
	return m.provenance(n, newTracing())
}

// tracing is the state of one provenance read: the variables and functions being read, against cycles, and what the
// parameters of the function being followed are given by the call it was reached from.
type tracing struct {
	seen    map[*ast.Symbol]bool
	calling map[*ast.Node]bool
	bound   map[*ast.Symbol]Provenance
}

func newTracing() *tracing {
	return &tracing{seen: map[*ast.Symbol]bool{}, calling: map[*ast.Node]bool{}}
}

func (m *Analyzer) provenance(n *ast.Node, t *tracing) Provenance {
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
		return m.provenance(c.WhenTrue, t) | m.provenance(c.WhenFalse, t)
	case ast.KindBinaryExpression:
		b := n.AsBinaryExpression()
		switch operator := b.OperatorToken.Kind; {
		case ast.IsLogicalOperator(operator) || operator == ast.KindQuestionQuestionToken:
			return m.provenance(b.Left, t) | m.provenance(b.Right, t)
		case operator == ast.KindCommaToken || operator == ast.KindEqualsToken:
			return m.provenance(b.Right, t)
		}
	case ast.KindPropertyAccessExpression:
		if m.size(n) {
			return SizeValue
		}
		if !m.computed(n) {
			return ExistingValue
		}
		if getter := m.getter(n); getter != nil {
			return m.returnsOf(getter, nil, t)
		}
	case ast.KindElementAccessExpression, ast.KindThisKeyword:
		return ExistingValue
	case ast.KindAwaitExpression:
		if call := Unwrap(n.Expression()); call.Kind == ast.KindCallExpression {
			return m.called(call, true, t)
		}
	case ast.KindCallExpression:
		return m.called(n, false, t)
	case ast.KindIdentifier:
		return m.variable(n, t)
	}
	return ComputedValue
}

// variable returns what a variable or parameter read gives.
func (m *Analyzer) variable(n *ast.Node, t *tracing) Provenance {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
	if symbol == nil || t.seen[symbol] {
		return ExistingValue
	}
	d := symbol.ValueDeclaration
	if d == nil || d.Kind != ast.KindVariableDeclaration && d.Kind != ast.KindParameter {
		return ExistingValue
	}
	t.seen[symbol] = true
	defer delete(t.seen, symbol)
	if d.Kind == ast.KindVariableDeclaration && ast.IsVarConst(d) {
		return m.provenance(d.Initializer(), t)
	}
	var kinds Provenance
	for _, def := range m.reaching(n, Value{Root: symbol, Path: []string{}}) {
		if bound, ok := t.bound[symbol]; ok && def.Before {
			kinds |= bound
		} else {
			kinds |= m.defProvenance(def, t)
		}
	}
	return kinds
}

// freshResults names the methods of global collections returning a new array.
var freshResults = map[string]map[string]bool{
	"Array": {"map": true, "filter": true, "slice": true, "concat": true, "flat": true, "flatMap": true, "toSorted": true,
		"toReversed": true, "toSpliced": true, "with": true},
	"ReadonlyArray": {"map": true, "filter": true, "slice": true, "concat": true, "flat": true, "flatMap": true,
		"toSorted": true, "toReversed": true, "toSpliced": true, "with": true},
}

// called returns what a call gives, awaited or not: an entry of a collection is existing data, a collection method
// building a new array a new object, and a call to a project function what its returns give. A generator, or an async
// function not awaited, gives a new object. A call to a function being followed, or to anything else, is computed.
func (m *Analyzer) called(n *ast.Node, awaited bool, t *tracing) Provenance {
	if _, ok := m.LookupOf(n); ok {
		return ExistingValue
	}
	if callee := Unwrap(n.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
		if owners, name, ok := m.libraryMember(callee.Name()); ok {
			if _, fresh := agreed(owners, name, freshResults); fresh {
				return FreshValue
			}
		}
	}
	declaration := Implementation(m.ctx.TypeChecker.GetResolvedSignature(n))
	if declaration == nil || !ast.IsFunctionLike(declaration) || declaration.Body() == nil || !ProjectFile(ast.GetSourceFileOfNode(declaration)) {
		return ComputedValue
	}
	flags := ast.GetFunctionFlags(declaration)
	if flags&ast.FunctionFlagsGenerator != 0 || flags&ast.FunctionFlagsAsync != 0 && !awaited {
		return FreshValue
	}
	bound := map[*ast.Symbol]Provenance{}
	arguments := n.Arguments()
	for i, parameter := range declaration.Parameters() {
		if i >= len(arguments) || arguments[i].Kind == ast.KindSpreadElement || parameter.Name().Kind != ast.KindIdentifier || parameter.AsParameterDeclaration().DotDotDotToken != nil {
			break
		}
		if symbol := parameter.Symbol(); symbol != nil {
			bound[symbol] = m.provenance(arguments[i], t)
		}
	}
	return m.returnsOf(declaration, bound, t)
}

// getter returns the get accessor with a body, in a project file, that a property read resolves to.
func (m *Analyzer) getter(n *ast.Node) *ast.Node {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n.Name())
	if symbol == nil {
		return nil
	}
	for _, d := range symbol.Declarations {
		if d.Kind == ast.KindGetAccessor && d.Body() != nil && ProjectFile(ast.GetSourceFileOfNode(d)) {
			return d
		}
	}
	return nil
}

// returnsOf returns what the returns of a function give, its parameters giving what bound says: the expression body of
// an arrow function, or every return outside the functions it declares. A return without a value, or reaching the end
// of the body, gives undefined, a constant. A function already being followed is computed.
func (m *Analyzer) returnsOf(fn *ast.Node, bound map[*ast.Symbol]Provenance, t *tracing) Provenance {
	if t.calling[fn] {
		return ComputedValue
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
		return m.provenance(body, t)
	}
	var kinds Provenance
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if ast.IsFunctionLike(c) || ast.IsClassLike(c) {
			return false
		}
		if c.Kind == ast.KindReturnStatement {
			kinds |= m.provenance(c.Expression(), t)
		}
		c.ForEachChild(walk)
		return false
	}
	body.ForEachChild(walk)
	if m.flowControl().of(body).end != nil {
		kinds |= ConstantValue
	}
	return kinds
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
	return m.defProvenance(d, newTracing())
}

func (m *Analyzer) defProvenance(d Def, t *tracing) Provenance {
	switch {
	case d.Before:
		return ExistingValue
	case d.Declaration != nil:
		return m.provenance(d.Declaration.Initializer(), t)
	case d.Write.Kind == Update:
		return ComputedValue
	case d.Write.Source != nil:
		return m.provenance(d.Write.Source, t)
	}
	return ExistingValue
}

// InputProvenance returns the kinds of value phi gets from the arm at index of merge, reading the value held before
// the decision through Before. Kinds are read where the arm reaches the merge (see published): a new object the arm, or
// the code before the decision, makes reachable from existing data is existing data there, and so is the size of a
// collection read before an element is appended to it, which then names that element.
func (m *Analyzer) InputProvenance(merge *Merge, phi Phi, arm int) Provenance {
	var kinds Provenance
	scope, end := merge.Arms[arm].Branch, merge.Decision.End()
	if scope != nil {
		end = scope.End()
	}
	for _, d := range phi.Inputs[arm] {
		if !d.Before {
			kinds |= m.publishedAs(d, phi.Value, scope, end)
			continue
		}
		for _, held := range m.Before(merge.Decision, phi.Value) {
			kinds |= m.publishedAs(held, phi.Value, enclosingFunction(merge.Decision), merge.Decision.Pos())
		}
	}
	return kinds
}

// publishedAs returns the kinds of value d gives to v, read after the effects in scope between d and end.
func (m *Analyzer) publishedAs(d Def, v Value, scope *ast.Node, end int) Provenance {
	kinds := m.DefProvenance(d)
	node := d.Declaration
	source := (*ast.Node)(nil)
	if node != nil {
		source = node.Initializer()
	} else if d.Write.Node != nil {
		node, source = d.Write.Node, d.Write.Source
	}
	if node == nil || scope == nil {
		return kinds
	}
	if kinds&FreshValue != 0 && m.published(v, scope, node.End(), end) {
		kinds = kinds&^FreshValue | ExistingValue
	}
	if kinds&SizeValue != 0 && source != nil {
		if sized := Unwrap(source); sized.Kind == ast.KindPropertyAccessExpression && m.appended(m.ValueOf(sized.Expression()), scope, node.End(), end) {
			kinds = kinds&^SizeValue | ExistingValue
		}
	}
	return kinds
}

// published reports whether an effect in scope between from and to makes v reachable from data that outlives scope:
// it may carry v into a receiver that is this or rooted in a variable declared outside scope.
func (m *Analyzer) published(v Value, scope *ast.Node, from, to int) bool {
	return slices.ContainsFunc(m.effectsBetween(scope, from, to), func(e Effect) bool {
		if !m.outlives(e.Receiver, scope) || e.Receiver.Within(v) {
			return false
		}
		return slices.ContainsFunc(e.Carried, func(n *ast.Node) bool { return m.ValueOf(n).Same(v) })
	})
}

// appended reports whether an effect in scope between from and to appends to collection.
func (m *Analyzer) appended(collection Value, scope *ast.Node, from, to int) bool {
	return slices.ContainsFunc(m.effectsBetween(scope, from, to), func(e Effect) bool {
		return e.Kind == Append && e.Receiver.Same(collection)
	})
}

func (m *Analyzer) effectsBetween(scope *ast.Node, from, to int) []Effect {
	return slices.DeleteFunc(m.EffectsIn([]*ast.Node{scope}), func(e Effect) bool { return e.Node.Pos() < from || e.Node.End() > to })
}

// outlives reports whether v is data of this, or of a variable declared outside scope.
func (m *Analyzer) outlives(v Value, scope *ast.Node) bool {
	if v.This != nil {
		return true
	}
	return v.Root != nil && v.Root.ValueDeclaration != nil && !encloses(scope, v.Root.ValueDeclaration)
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
