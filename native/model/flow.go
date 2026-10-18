package model

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Rejection is a place in a function body that rejects input: a throw under conditions reading input, or a call to a
// function that rejects what it receives. Its claim is not read; it is known by its data. Checked is the input its
// conditions, or the arguments the callee rejects, are built from; Read is all the data its conditions read, including
// data passed into calls whose result they test, and for a method call, an unknown part of its receiver.
type Rejection struct {
	Node    *ast.Node
	Callee  *ast.Node
	Checked []Value
	Read    []Value
	// Owned marks a rejection made by constructing a value whose type keeps the rejection: it belongs to that type, not to the
	// function handing data on.
	Owned bool
}

// Call is a call or construction in a function body, the input data and the data read through this that each argument
// carries, and the rejections reading data, not owned by a type, that precede it in source order. Thrown marks a call
// building what a throw raises. Discarded marks a call whose result nothing receives: the call, awaited or
// parenthesized, is a whole expression statement.
type Call struct {
	Thrown     bool
	Discarded  bool
	Node       *ast.Node
	Arguments  [][]Value
	Rejections []*Rejection
}

// Flow is what a function does with its input, its parameters and the locals derived from them: the rejections it makes,
// the calls it passes data to, and the data it returns.
type Flow struct {
	// Rejections lists every rejection in source order, including those whose checked data is not input.
	Rejections []*Rejection
	Calls      []Call
	// Results is the input data its return statements read, and Carried the parameters among them.
	Results []Value
	Carried map[int]bool
	// Checked and Read give, by parameter position, the data its rejections are built from and read, relative to it.
	Checked map[int][]Value
	Read    map[int][]Value
}

type storage struct {
	path    []string
	sources []Value
}

func discarded(call *ast.Node) bool {
	n := call
	for n.Parent != nil && (n.Parent.Kind == ast.KindAwaitExpression || n.Parent.Kind == ast.KindParenthesizedExpression) {
		n = n.Parent
	}
	return n.Parent != nil && n.Parent.Kind == ast.KindExpressionStatement
}

type call struct {
	node      *ast.Node
	arguments [][]Value
	prior     int
	thrown    bool
}

// flowAnalysis tracks one function body: which values are input, and which data each parameter and local holds.
type flowAnalysis struct {
	m     *Analyzer
	fn    *ast.Node
	input map[*ast.Symbol]bool
	// aliases maps a local initialized by an access chain to that chain.
	aliases map[*ast.Symbol]Value
	// locals maps every other local, and each parameter, to the data it holds.
	locals map[*ast.Symbol][]Value
	// stored records data stored into parts of a local after its declaration, by assignment or by a method call.
	stored map[*ast.Symbol][]storage
	params map[*ast.Symbol]int
	// direct marks parameters declared by an identifier rather than a binding pattern.
	direct map[*ast.Symbol]bool
	found  []*Rejection
	calls  []call
	result []Value
}

func partial(values []Value) []Value {
	parts := make([]Value, len(values))
	for i, v := range values {
		v.Partial = true
		parts[i] = v
	}
	return parts
}

func (a *flowAnalysis) symbol(n *ast.Node) *ast.Symbol {
	return a.m.SymbolOf(n)
}

// names returns the symbols a binding name declares.
func (a *flowAnalysis) names(name *ast.Node) []*ast.Symbol {
	if name == nil {
		return nil
	}
	if name.Kind == ast.KindIdentifier {
		if symbol := a.symbol(name); symbol != nil {
			return []*ast.Symbol{symbol}
		}
		return nil
	}
	symbols := []*ast.Symbol{}
	name.ForEachChild(func(child *ast.Node) bool {
		if child.Kind == ast.KindBindingElement && child.Name() != nil {
			symbols = append(symbols, a.names(child.Name())...)
		}
		return false
	})
	return symbols
}

func (a *flowAnalysis) taint(name *ast.Node) {
	for _, symbol := range a.names(name) {
		a.input[symbol] = true
	}
}

// declare records the data a binding name holds: base when it is initialized by an access chain, otherwise sources,
// otherwise its own root. Object pattern elements extend base by their property name, and array pattern elements of an
// array or tuple by their position; other elements, such as a rest element, hold part of base.
func (a *flowAnalysis) declare(name *ast.Node, base *Value, sources []Value) {
	if name == nil {
		return
	}
	if name.Kind == ast.KindIdentifier {
		symbol := a.symbol(name)
		switch {
		case symbol == nil:
		case base != nil:
			a.aliases[symbol] = *base
		case len(sources) > 0:
			a.locals[symbol] = sources
		default:
			a.locals[symbol] = []Value{{Root: symbol}}
		}
		return
	}
	keyed := name.Kind == ast.KindObjectBindingPattern
	indexed := !keyed && a.m.positional(name)
	for position, element := range name.AsBindingPattern().Elements.Nodes {
		if element.Kind != ast.KindBindingElement || element.Name() == nil {
			continue
		}
		var key string
		switch {
		case keyed:
			name := element.PropertyName()
			if name == nil && element.Name().Kind == ast.KindIdentifier {
				name = element.Name()
			}
			if name != nil && (name.Kind == ast.KindIdentifier || ast.IsStringOrNumericLiteralLike(name)) {
				key = name.Text()
			}
		case indexed:
			key = strconv.Itoa(position)
		}
		switch {
		case base != nil && key != "" && element.AsBindingElement().DotDotDotToken == nil:
			child := base.extend(key)
			a.declare(element.Name(), &child, nil)
		case base != nil:
			a.declare(element.Name(), nil, partial([]Value{*base}))
		default:
			a.declare(element.Name(), nil, sources)
		}
	}
}

func (a *flowAnalysis) reads(expression *ast.Node) bool {
	if expression == nil {
		return false
	}
	found := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if n.Kind == ast.KindIdentifier {
			if symbol := a.symbol(n); symbol != nil && a.input[symbol] {
				found = true
			}
		}
		if !found {
			n.ForEachChild(walk)
		}
		return found
	}
	walk(expression)
	return found
}

// skipOuter looks through parentheses, assertions and await: an awaited value is the same data.
func skipOuter(n *ast.Node) *ast.Node {
	for n != nil {
		switch n.Kind {
		case ast.KindParenthesizedExpression, ast.KindAsExpression, ast.KindSatisfiesExpression, ast.KindNonNullExpression,
			ast.KindTypeAssertionExpression, ast.KindAwaitExpression:
			n = n.Expression()
		default:
			return n
		}
	}
	return nil
}

// exact returns the access chain an expression denotes, when it is an identifier, a property access, or an element
// access with a literal key over a parameter, an alias, or a local that is its own root.
func (a *flowAnalysis) exact(n *ast.Node) (Value, bool) {
	n = skipOuter(n)
	if n == nil {
		return Value{}, false
	}
	switch n.Kind {
	case ast.KindThisKeyword:
		return Value{This: a.fn}, true
	case ast.KindIdentifier:
		symbol := a.symbol(n)
		if symbol == nil {
			return Value{}, false
		}
		if alias, ok := a.aliases[symbol]; ok {
			return alias, true
		}
		if held := a.locals[symbol]; len(held) == 1 && held[0].Root == symbol && len(held[0].Path) == 0 {
			return held[0], true
		}
	case ast.KindPropertyAccessExpression:
		if base, ok := a.exact(n.Expression()); ok {
			return base.extend(n.Name().Text()), true
		}
	case ast.KindElementAccessExpression:
		key := n.AsElementAccessExpression().ArgumentExpression
		if base, ok := a.exact(n.Expression()); ok && ast.IsStringOrNumericLiteralLike(key) {
			return base.extend(key.Text()), true
		}
	}
	return Value{}, false
}

func (a *flowAnalysis) primitive(n *ast.Node) bool {
	t := a.m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil {
		return false
	}
	members := []*checker.Type{t}
	if t.Flags()&checker.TypeFlagsUnion != 0 {
		members = t.Types()
	}
	for _, member := range members {
		if member.Flags()&(checker.TypeFlagsPrimitive|checker.TypeFlagsUndefined) == 0 {
			return false
		}
	}
	return true
}

// sources returns the data an expression's value is built from. A conditional expression's value is built from its
// branches; its condition only selects one. A method call carries part of its receiver's data. A call to a project
// function carries part of the arguments its return statements read (see carried); a call to a library function or a
// construction carries the data of its object arguments. A primitive argument to a library function carries none: the
// value produced from it is new data. Functions carry none. Inside arguments (asArgument), primitive subexpressions
// carry none.
func (a *flowAnalysis) sources(n *ast.Node, asArgument bool) []Value {
	return unique(a.built(n, asArgument))
}

// unique keeps the first of equal values: data an expression is built from is a set, and copying it between locals
// must not multiply it.
func unique(values []Value) []Value {
	type key struct {
		root    *ast.Symbol
		this    *ast.Node
		base    *ast.Node
		path    string
		partial bool
	}
	seen := map[key]bool{}
	kept := values[:0:0]
	for _, v := range values {
		k := key{v.Root, v.This, v.Base, strings.Join(v.Path, "\x00"), v.Partial}
		if !seen[k] {
			seen[k] = true
			kept = append(kept, v)
		}
	}
	return kept
}

func (a *flowAnalysis) built(n *ast.Node, asArgument bool) []Value {
	n = skipOuter(n)
	if n == nil || ast.IsFunctionLike(n) || ast.IsClassLike(n) || (asArgument && a.primitive(n)) {
		return nil
	}
	if v, ok := a.exact(n); ok {
		return append([]Value{v}, a.storedAt(v)...)
	}
	switch n.Kind {
	case ast.KindIdentifier:
		if symbol := a.symbol(n); symbol != nil {
			return a.locals[symbol]
		}
		return nil
	case ast.KindPropertyAccessExpression:
		return partial(a.sources(n.Expression(), asArgument))
	case ast.KindConditionalExpression:
		conditional := n.AsConditionalExpression()
		return append(a.sources(conditional.WhenTrue, asArgument), a.sources(conditional.WhenFalse, asArgument)...)
	case ast.KindElementAccessExpression:
		return append(partial(a.sources(n.Expression(), asArgument)), a.sources(n.AsElementAccessExpression().ArgumentExpression, asArgument)...)
	case ast.KindCallExpression, ast.KindNewExpression:
		values := []Value{}
		callee := n.Expression()
		if n.Kind == ast.KindCallExpression && (callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression) {
			values = append(values, partial(a.sources(callee.Expression(), asArgument))...)
		}
		carried := a.m.carried(n)
		for index, argument := range n.Arguments() {
			switch {
			case carried == nil && !a.primitive(argument):
				values = append(values, a.sources(argument, true)...)
			case carried != nil && carried[index]:
				values = append(values, partial(a.sources(argument, false))...)
			}
		}
		return values
	}
	values := []Value{}
	n.ForEachChild(func(child *ast.Node) bool {
		values = append(values, a.sources(child, asArgument)...)
		return false
	})
	return values
}

// carried returns the argument positions whose data a call's result can carry across the return boundary, when the
// callee is a project function or method: those its return statements read, or none when it has no body to establish
// it. It returns nil for library functions and constructors, whose object arguments are assumed to flow.
func (m *Analyzer) carried(n *ast.Node) map[int]bool {
	declaration := Implementation(m.ctx.TypeChecker.GetResolvedSignature(n))
	if declaration == nil {
		return nil
	}
	if declaration.Kind == ast.KindConstructor || !ProjectFile(ast.GetSourceFileOfNode(declaration)) {
		return nil
	}
	if !ast.IsFunctionLike(declaration) || declaration.Body() == nil {
		return map[int]bool{}
	}
	if callee := m.Flow(declaration); callee != nil {
		return callee.Carried
	}
	return map[int]bool{}
}

// store records the data stored into part of a local by an assignment or a method call on a chain rooted at it. The
// local keeps its own root, so reading another part of it does not read the stored data.
func (a *flowAnalysis) store(target *ast.Node, values []*ast.Node) {
	v, ok := a.exact(target)
	if !ok || v.Root == nil {
		return
	}
	if _, parameter := a.params[v.Root]; parameter {
		return
	}
	if _, local := a.locals[v.Root]; !local {
		return
	}
	sources := []Value{}
	for _, value := range values {
		sources = append(sources, a.sources(value, false)...)
	}
	a.stored[v.Root] = append(a.stored[v.Root], storage{path: v.Path, sources: sources})
}

// storeCall records a method call on a local as storing its arguments into it: for a project constructor or method,
// only the arguments at parameters it stores into its class's state, so a rejection stores nothing; for a library method
// such as `values.push(value)`, every argument.
func (a *flowAnalysis) storeCall(n *ast.Node) {
	values := n.Arguments()
	if _, declaration := a.m.Member(n); declaration != nil {
		values = nil
		for index, argument := range n.Arguments() {
			if a.m.Stores(declaration, index) {
				values = append(values, argument)
			}
		}
	}
	a.store(n.Expression().Expression(), values)
}

// storedAt returns the data stored into the part of a local that v reads, or into parts it contains. Data stored into
// a containing part is attributed to that part as a whole.
func (a *flowAnalysis) storedAt(v Value) []Value {
	if v.Root == nil {
		return nil
	}
	values := []Value{}
	for _, entry := range a.stored[v.Root] {
		if related(entry.path, v.Path) && len(v.Path) <= len(entry.path) {
			values = append(values, entry.sources...)
		}
	}
	return unique(values)
}

// Narrowing reports whether a condition only tests what TypeScript keeps in a narrowed type: negations, conjunctions,
// and disjunctions of comparisons with null or undefined, truthiness tests of values that cannot be falsy when present,
// `instanceof`, `in`, comparisons of `typeof`, and calls to type guards. Such a test establishes presence or a type,
// which the narrowed type of the value already carries. A const boolean local stands for the condition it was initialized with,
// as it does for TypeScript; unlike TypeScript, the model does not require the tested data to stay unassigned. Among
// the parts of a conjunction or disjunction, a truthiness test of a value that another part requires to be an object is
// narrowing too: no object is falsy, so the narrowed object type carries it.
func (m *Analyzer) Narrowing(n *ast.Node) bool {
	n = ast.SkipParentheses(n)
	switch n.Kind {
	case ast.KindIdentifier:
		if condition := m.condition(n); condition != nil {
			return m.Narrowing(condition)
		}
	case ast.KindPrefixUnaryExpression:
		unary := n.AsPrefixUnaryExpression()
		return unary.Operator == ast.KindExclamationToken && m.Narrowing(unary.Operand)
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		switch operator := binary.OperatorToken.Kind; operator {
		case ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
			operands := parts(n, operator)
			for i := range operands {
				if !m.narrowingAmong(operands, i, operator) {
					return false
				}
			}
			return true
		case ast.KindEqualsEqualsToken, ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsToken, ast.KindExclamationEqualsEqualsToken:
			return nullish(binary.Left) || nullish(binary.Right) || typeOf(binary.Left) || typeOf(binary.Right)
		case ast.KindInstanceOfKeyword, ast.KindInKeyword:
			return true
		}
		return false
	case ast.KindCallExpression:
		signature := m.ctx.TypeChecker.GetResolvedSignature(n)
		return signature != nil && m.ctx.TypeChecker.GetTypePredicateOfSignature(signature) != nil
	}
	held, ok := m.Holds(n)
	return ok && held&AbsentFalsy == 0
}

// condition returns the initializer of a const boolean local declared before n in its file, so that following
// conditions never cycles.
func (m *Analyzer) condition(n *ast.Node) *ast.Node {
	if t := m.ctx.TypeChecker.GetTypeAtLocation(n); t == nil || t.Flags()&checker.TypeFlagsBooleanLike == 0 {
		return nil
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
	if symbol == nil || symbol.ValueDeclaration == nil {
		return nil
	}
	d := symbol.ValueDeclaration
	if d.Kind != ast.KindVariableDeclaration || d.Name().Kind != ast.KindIdentifier || !ast.IsVarConst(d) {
		return nil
	}
	if ast.GetSourceFileOfNode(d) != ast.GetSourceFileOfNode(n) || d.End() > n.Pos() {
		return nil
	}
	return d.Initializer()
}

// narrowingAmong reports whether the operand at i of a conjunction or disjunction is narrowing: on its own, or as a
// truthiness test of a value that another operand requires to be an object. In a conjunction both tests hold where it
// holds; in a disjunction both are negated, ruling the value out where it holds.
func (m *Analyzer) narrowingAmong(operands []*ast.Node, i int, operator ast.Kind) bool {
	if m.Narrowing(operands[i]) {
		return true
	}
	value, truthy, ok := truthiness(operands[i])
	positive := operator == ast.KindAmpersandAmpersandToken
	return ok && truthy == positive && slices.ContainsFunc(operands, func(o *ast.Node) bool {
		subject, requires, ok := m.objectTest(o)
		return ok && requires == positive && m.SameValue(subject, value)
	})
}

// truthiness reads a truthiness test of an access chain, possibly negated: truthy is whether it holds where the value
// is truthy.
func truthiness(n *ast.Node) (*ast.Node, bool, bool) {
	n = ast.SkipParentheses(n)
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			value, truthy, ok := truthiness(unary.Operand)
			return value, !truthy, ok
		}
	case ast.KindIdentifier, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
		return n, true, true
	}
	return nil, false, false
}

// objectTest reads a test that holds only for objects, or only for other values: a comparison of `typeof` with
// "object" or "function", `instanceof`, `in`, or a call to a type guard whose type admits no primitive. requires is
// whether it holds only for objects.
func (m *Analyzer) objectTest(n *ast.Node) (*ast.Node, bool, bool) {
	n = ast.SkipParentheses(n)
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			subject, requires, ok := m.objectTest(unary.Operand)
			return subject, !requires, ok
		}
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		switch operator := binary.OperatorToken.Kind; operator {
		case ast.KindEqualsEqualsToken, ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsToken, ast.KindExclamationEqualsEqualsToken:
			equal := operator == ast.KindEqualsEqualsToken || operator == ast.KindEqualsEqualsEqualsToken
			for _, pair := range [][2]*ast.Node{{binary.Left, binary.Right}, {binary.Right, binary.Left}} {
				test, text := ast.SkipParentheses(pair[0]), ast.SkipParentheses(pair[1])
				if test.Kind == ast.KindTypeOfExpression && ast.IsStringLiteralLike(text) && (text.Text() == "object" || text.Text() == "function") {
					return test.Expression(), equal, true
				}
			}
		case ast.KindInstanceOfKeyword:
			return binary.Left, true, true
		case ast.KindInKeyword:
			return binary.Right, true, true
		}
	case ast.KindCallExpression:
		signature := m.ctx.TypeChecker.GetResolvedSignature(n)
		if signature == nil {
			break
		}
		predicate := m.ctx.TypeChecker.GetTypePredicateOfSignature(signature)
		if predicate == nil || checker.TypePredicate_kind(predicate) != checker.TypePredicateKindIdentifier {
			break
		}
		index := int(checker.TypePredicate_parameterIndex(predicate))
		if arguments := n.Arguments(); index < len(arguments) && m.admitsNoFalsy(checker.TypePredicate_t(predicate)) {
			return arguments[index], true, true
		}
	}
	return nil, false, false
}

func typeOf(n *ast.Node) bool {
	return ast.SkipParentheses(n).Kind == ast.KindTypeOfExpression
}

func nullish(n *ast.Node) bool {
	_, ok := missingLiteral(ast.SkipParentheses(n))
	return ok
}

// tested returns the data a condition checks. Narrowing tests among its negations, conjunctions, and disjunctions
// check no data.
func (a *flowAnalysis) tested(n *ast.Node) []Value {
	n = ast.SkipParentheses(n)
	if a.m.Narrowing(n) {
		return nil
	}
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			return a.tested(unary.Operand)
		}
	case ast.KindBinaryExpression:
		if kind := n.AsBinaryExpression().OperatorToken.Kind; kind == ast.KindAmpersandAmpersandToken || kind == ast.KindBarBarToken {
			found := []Value{}
			operands := parts(n, kind)
			for i, operand := range operands {
				if !a.m.narrowingAmong(operands, i, kind) {
					found = append(found, a.tested(operand)...)
				}
			}
			return found
		}
	}
	return append(a.sources(n, false), a.libraryArguments(n)...)
}

// libraryArguments returns the data of the primitive arguments to library calls in a condition. Such an argument
// carries no data into the call's result, but a condition testing the result tests it: `pattern.test(input.id)`
// checks input.id.
func (a *flowAnalysis) libraryArguments(n *ast.Node) []Value {
	found := []Value{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindCallExpression && a.m.carried(n) == nil {
			for _, argument := range n.Arguments() {
				if a.primitive(argument) {
					found = append(found, a.sources(argument, false)...)
				}
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(n)
	return found
}

// testedReads returns all the data a condition reads, skipping narrowing tests like tested. Unlike tested, a call
// reads every argument, whatever its type.
func (a *flowAnalysis) testedReads(n *ast.Node) []Value {
	n = ast.SkipParentheses(n)
	if a.m.Narrowing(n) {
		return nil
	}
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			return a.testedReads(unary.Operand)
		}
	case ast.KindBinaryExpression:
		if kind := n.AsBinaryExpression().OperatorToken.Kind; kind == ast.KindAmpersandAmpersandToken || kind == ast.KindBarBarToken {
			found := []Value{}
			operands := parts(n, kind)
			for i, operand := range operands {
				if !a.m.narrowingAmong(operands, i, kind) {
					found = append(found, a.testedReads(operand)...)
				}
			}
			return found
		}
	}
	return a.readsAll(n)
}

// readsAll returns all the data an expression reads: like sources, but every call argument counts, as an unknown part
// of its data, since the callee may read only some of it.
func (a *flowAnalysis) readsAll(n *ast.Node) []Value {
	return unique(a.allReads(n))
}

func (a *flowAnalysis) allReads(n *ast.Node) []Value {
	n = skipOuter(n)
	if n == nil || ast.IsFunctionLike(n) || ast.IsClassLike(n) {
		return nil
	}
	if v, ok := a.exact(n); ok {
		return append([]Value{v}, a.storedAt(v)...)
	}
	switch n.Kind {
	case ast.KindIdentifier:
		if symbol := a.symbol(n); symbol != nil {
			return a.locals[symbol]
		}
		return nil
	case ast.KindPropertyAccessExpression:
		return partial(a.readsAll(n.Expression()))
	case ast.KindCallExpression, ast.KindNewExpression:
		values := []Value{}
		callee := n.Expression()
		if n.Kind == ast.KindCallExpression && (callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression) {
			values = append(values, partial(a.readsAll(callee.Expression()))...)
		}
		for _, argument := range n.Arguments() {
			values = append(values, partial(a.readsAll(argument))...)
		}
		return values
	}
	values := []Value{}
	n.ForEachChild(func(child *ast.Node) bool {
		values = append(values, a.readsAll(child)...)
		return false
	})
	return values
}

// call records a rejection when a call or new expression passes input to a project function that rejects on its parameters
// at those positions. Calls to type guards and assertion functions are narrowing tests, not rejections.
func (a *flowAnalysis) call(n *ast.Node) {
	arguments := n.Arguments()
	if !slices.ContainsFunc(arguments, a.reads) {
		return
	}
	signature := a.m.ctx.TypeChecker.GetResolvedSignature(n)
	declaration := Implementation(signature)
	if declaration == nil || a.m.ctx.TypeChecker.GetTypePredicateOfSignature(signature) != nil {
		return
	}
	if !ast.IsFunctionLike(declaration) || declaration.Body() == nil || !ProjectFile(ast.GetSourceFileOfNode(declaration)) {
		return
	}
	callee := a.m.Flow(declaration)
	if callee == nil || len(callee.Read) == 0 {
		return
	}
	read := a.mapped(arguments, callee.Read)
	if len(read) == 0 {
		return
	}
	if receiver := n.Expression(); n.Kind == ast.KindCallExpression && receiver.Kind == ast.KindPropertyAccessExpression {
		read = append(read, partial(a.readsAll(receiver.Expression()))...)
	}
	if a.m.Unforgeable(a.m.Unwrap(a.m.ctx.TypeChecker.GetTypeAtLocation(n))) {
		a.found = append(a.found, &Rejection{Node: n, Callee: declaration, Read: read, Owned: true})
		return
	}
	a.found = append(a.found, &Rejection{Node: n, Callee: declaration, Checked: a.mapped(arguments, callee.Checked), Read: read})
}

// mapped returns the parts of each argument that a callee checks of the parameter at its position, or some part of an
// argument that is not an access chain.
func (a *flowAnalysis) mapped(arguments []*ast.Node, params map[int][]Value) []Value {
	checked := []Value{}
	for _, index := range slices.Sorted(maps.Keys(params)) {
		if index >= len(arguments) || !a.reads(arguments[index]) {
			continue
		}
		base, ok := a.exact(arguments[index])
		if !ok {
			checked = append(checked, partial(a.sources(arguments[index], false))...)
			continue
		}
		for _, part := range params[index] {
			v := base
			for _, key := range part.Path {
				v = v.extend(key)
			}
			v.Partial = part.Partial
			checked = append(checked, v)
		}
	}
	return checked
}

func (a *flowAnalysis) visit(n *ast.Node, guards []*ast.Node, inTry bool) {
	if n == nil || ast.IsFunctionLike(n) || ast.IsClassLike(n) {
		return
	}
	switch n.Kind {
	case ast.KindThrowStatement:
		before := len(a.calls)
		a.visitChildren(n, guards, inTry)
		for i := before; i < len(a.calls); i++ {
			a.calls[i].thrown = true
		}
		if len(guards) > 0 && !inTry {
			checked, read := []Value{}, []Value{}
			for _, guard := range guards {
				checked = append(checked, a.tested(guard)...)
				read = append(read, a.testedReads(guard)...)
			}
			a.found = append(a.found, &Rejection{Node: n, Checked: checked, Read: read})
		}
		return
	case ast.KindReturnStatement:
		a.visitChildren(n, guards, inTry)
		a.result = append(a.result, a.sources(n.Expression(), false)...)
		return
	case ast.KindCallExpression, ast.KindNewExpression:
		a.visitChildren(n, guards, inTry)
		site := call{node: n, prior: len(a.found)}
		for _, argument := range n.Arguments() {
			site.arguments = append(site.arguments, a.sources(argument, false))
		}
		a.calls = append(a.calls, site)
		if !inTry {
			a.call(n)
		}
		if callee := n.Expression(); n.Kind == ast.KindCallExpression && callee.Kind == ast.KindPropertyAccessExpression {
			a.storeCall(n)
		}
		return
	case ast.KindBinaryExpression:
		a.visitChildren(n, guards, inTry)
		if ast.IsAssignmentExpression(n, false) {
			binary := n.AsBinaryExpression()
			a.store(binary.Left, []*ast.Node{binary.Right})
		}
		return
	case ast.KindVariableDeclaration:
		a.visitChildren(n, guards, inTry)
		initializer := n.Initializer()
		if base, ok := a.exact(initializer); ok {
			a.declare(n.Name(), &base, nil)
		} else if produced := skipOuter(initializer); produced != nil && (produced.Kind == ast.KindCallExpression || produced.Kind == ast.KindNewExpression) {
			a.declare(n.Name(), nil, nil)
			sources := a.sources(initializer, false)
			for _, symbol := range a.names(n.Name()) {
				a.stored[symbol] = append(a.stored[symbol], storage{sources: sources})
			}
		} else {
			a.declare(n.Name(), nil, a.sources(initializer, false))
		}
		if a.reads(initializer) {
			a.taint(n.Name())
		}
		return
	case ast.KindForOfStatement, ast.KindForInStatement:
		statement := n.AsForInOrOfStatement()
		a.visit(statement.Expression, guards, inTry)
		if list := statement.Initializer; list != nil && list.Kind == ast.KindVariableDeclarationList {
			sources := a.sources(statement.Expression, false)
			for _, declaration := range list.AsVariableDeclarationList().Declarations.Nodes {
				a.declare(declaration.Name(), nil, sources)
				if a.reads(statement.Expression) {
					a.taint(declaration.Name())
				}
			}
		}
		a.visit(statement.Statement, guards, inTry)
		return
	case ast.KindBlock, ast.KindCaseClause, ast.KindDefaultClause:
		if n.Kind == ast.KindCaseClause {
			a.visit(n.Expression(), guards, inTry)
		}
		for _, statement := range statementList(n) {
			a.visit(statement, guards, inTry)
			guards = a.after(statement, guards)
		}
		return
	case ast.KindIfStatement:
		statement := n.AsIfStatement()
		a.visit(statement.Expression, guards, inTry)
		inner := guards
		if a.reads(statement.Expression) && !a.m.Narrowing(statement.Expression) {
			inner = append(append([]*ast.Node{}, guards...), statement.Expression)
		}
		a.visit(statement.ThenStatement, inner, inTry)
		a.visit(statement.ElseStatement, inner, inTry)
		return
	case ast.KindSwitchStatement:
		statement := n.AsSwitchStatement()
		a.visit(statement.Expression, guards, inTry)
		inner := guards
		if a.reads(statement.Expression) {
			inner = append(append([]*ast.Node{}, guards...), statement.Expression)
		}
		a.visit(statement.CaseBlock, inner, inTry)
		return
	case ast.KindTryStatement:
		statement := n.AsTryStatement()
		a.visit(statement.TryBlock, guards, inTry || statement.CatchClause != nil)
		a.visit(statement.CatchClause, guards, inTry)
		a.visit(statement.FinallyBlock, guards, inTry)
		return
	}
	a.visitChildren(n, guards, inTry)
}

// after returns the guards of the statements following statement in its list: an if statement reading input whose one
// branch returns early, accepting that input, guards them with its condition.
func (a *flowAnalysis) after(statement *ast.Node, guards []*ast.Node) []*ast.Node {
	if statement.Kind != ast.KindIfStatement {
		return guards
	}
	s := statement.AsIfStatement()
	if !a.reads(s.Expression) || a.m.Narrowing(s.Expression) || returns(s.ThenStatement) == returns(s.ElseStatement) {
		return guards
	}
	return append(append([]*ast.Node{}, guards...), s.Expression)
}

// returns reports whether a branch ends with a return statement.
func returns(n *ast.Node) bool {
	if n != nil && n.Kind == ast.KindBlock {
		statements := n.Statements()
		return len(statements) > 0 && returns(statements[len(statements)-1])
	}
	return n != nil && n.Kind == ast.KindReturnStatement
}

func (a *flowAnalysis) visitChildren(n *ast.Node, guards []*ast.Node, inTry bool) {
	n.ForEachChild(func(child *ast.Node) bool {
		a.visit(child, guards, inTry)
		return false
	})
}

// inputs keeps the values rooted at input, and with withContext, those read through this. Data obtained without the
// input, such as the current time, does not connect a rejection to the result.
func (a *flowAnalysis) inputs(values []Value, withContext bool) []Value {
	kept := []Value{}
	for _, v := range values {
		if v.Root != nil && a.input[v.Root] || withContext && v.This != nil {
			kept = append(kept, v)
		}
	}
	return kept
}

// relative adds the values rooted at parameters to params, by parameter position, without their root.
func (a *flowAnalysis) relative(values []Value, params map[int][]Value) {
	for _, v := range values {
		index, ok := a.params[v.Root]
		switch {
		case v.Root == nil || !ok:
		case a.direct[v.Root]:
			params[index] = append(params[index], Value{Path: v.Path, Partial: v.Partial})
		default:
			params[index] = append(params[index], Value{Partial: true})
		}
	}
}

// Flow analyzes fn's body, or returns nil while it is being analyzed, so recursion counts as not checking.
func (m *Analyzer) Flow(fn *ast.Node) *Flow {
	if result, ok := m.flows[fn]; ok {
		return result
	}
	m.flows[fn] = nil
	a := &flowAnalysis{m: m, fn: fn, input: map[*ast.Symbol]bool{}, aliases: map[*ast.Symbol]Value{}, locals: map[*ast.Symbol][]Value{},
		stored: map[*ast.Symbol][]storage{}, params: map[*ast.Symbol]int{}, direct: map[*ast.Symbol]bool{}}
	for index, parameter := range fn.Parameters() {
		if symbol := parameter.Symbol(); ast.IsBindingPattern(parameter.Name()) && symbol != nil {
			root := Value{Root: symbol, Path: []string{}}
			a.declare(parameter.Name(), &root, nil)
			a.input[symbol], a.params[symbol], a.direct[symbol] = true, index, true
		} else {
			a.declare(parameter.Name(), nil, nil)
		}
		a.taint(parameter.Name())
		for _, symbol := range a.names(parameter.Name()) {
			a.params[symbol] = index
			a.direct[symbol] = parameter.Name().Kind == ast.KindIdentifier
		}
	}
	if body := fn.Body(); body != nil && len(a.input) > 0 {
		a.visit(body, nil, false)
		if body.Kind != ast.KindBlock {
			a.result = append(a.result, a.sources(body, false)...)
		}
	}
	f := &Flow{Checked: map[int][]Value{}, Read: map[int][]Value{}, Results: a.inputs(a.result, false), Carried: map[int]bool{}}
	for _, v := range f.Results {
		if index, ok := a.params[v.Root]; ok && v.Root != nil {
			f.Carried[index] = true
		}
	}
	f.Rejections = a.found
	for _, found := range a.found {
		found.Checked = a.inputs(found.Checked, false)
		found.Read = a.inputs(found.Read, true)
		a.relative(found.Checked, f.Checked)
		a.relative(found.Read, f.Read)
	}
	for _, site := range a.calls {
		c := Call{Node: site.node, Thrown: site.thrown, Discarded: discarded(site.node)}
		for _, argument := range site.arguments {
			c.Arguments = append(c.Arguments, a.inputs(argument, true))
		}
		for _, found := range a.found[:site.prior] {
			if len(found.Read) > 0 && !found.Owned {
				c.Rejections = append(c.Rejections, found)
			}
		}
		f.Calls = append(f.Calls, c)
	}
	m.flows[fn] = f
	return f
}

func statementList(n *ast.Node) []*ast.Node {
	switch n.Kind {
	case ast.KindBlock:
		return n.AsBlock().Statements.Nodes
	case ast.KindCaseClause, ast.KindDefaultClause:
		return n.AsCaseOrDefaultClause().Statements.Nodes
	case ast.KindSourceFile:
		return n.AsSourceFile().Statements.Nodes
	case ast.KindModuleBlock:
		return n.AsModuleBlock().Statements.Nodes
	}
	return nil
}

func encloses(outer *ast.Node, inner *ast.Node) bool {
	return outer.Pos() <= inner.Pos() && inner.End() <= outer.End()
}

// leaves reports whether a statement list ends by leaving it, through a statement that does not contain the rejection: the
// rejection's own throw ending its guard does not count.
func (m *Analyzer) leaves(list []*ast.Node, rejection *ast.Node) bool {
	return len(list) > 0 && m.exits(list[len(list)-1]) && !encloses(list[len(list)-1], rejection)
}

// Precedes reports whether a rejection can run before a call on the way to it: the statement holding the rejection comes
// before the call in a statement list enclosing both, and no statement list between the rejection and that statement ends
// by leaving it, as a branch that returns does. A rejection nested in conditions still precedes a call after them.
func (m *Analyzer) Precedes(check *ast.Node, call *ast.Node) bool {
	for list := call.Parent; list != nil; list = list.Parent {
		if statementList(list) == nil || !encloses(list, check) {
			continue
		}
		statement := check
		for statement.Parent != nil && statement.Parent != list {
			statement = statement.Parent
		}
		if statement.End() > call.Pos() {
			return false
		}
		for n := check.Parent; n != nil && n != list; n = n.Parent {
			if m.leaves(statementList(n), check) {
				return false
			}
		}
		return true
	}
	return false
}

// Exclusive reports whether a and b cannot both run: the if conditions guarding them below the statement list enclosing
// both make claims about the same value that no value satisfies together. Tests joined by && hold in a then branch, and
// the opposites of tests joined by || in an else branch.
func (m *Analyzer) Exclusive(a, b *ast.Node) bool {
	list := a.Parent
	for list != nil && (statementList(list) == nil || !encloses(list, b)) {
		list = list.Parent
	}
	if list == nil {
		return false
	}
	for _, x := range m.holding(a, list) {
		for _, y := range m.holding(b, list) {
			if x.Value.Same(y.Value) && x.Claim.comparable(y.Claim) && len(within(x.Claim.Accepted, y.Claim.Accepted)) == 0 {
				return true
			}
		}
	}
	return false
}

// holding returns the tests that hold where n runs, from the if statements between n and list.
func (m *Analyzer) holding(n, list *ast.Node) []Test {
	tests := []Test{}
	for c := n; c.Parent != nil && c.Parent != list; c = c.Parent {
		p := c.Parent
		if p.Kind != ast.KindIfStatement {
			continue
		}
		s := p.AsIfStatement()
		switch c {
		case s.ThenStatement:
			for _, part := range parts(s.Expression, ast.KindAmpersandAmpersandToken) {
				if t, ok := m.TestOf(part); ok && t.Value.Named() {
					tests = append(tests, t)
				}
			}
		case s.ElseStatement:
			for _, part := range parts(s.Expression, ast.KindBarBarToken) {
				if t, ok := m.TestOf(part); ok && t.Value.Named() {
					t.Claim = t.Claim.Not()
					tests = append(tests, t)
				}
			}
		}
	}
	return tests
}
