package no_forgeable_validated_result

import (
	"fmt"
	"os"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// path is an access chain from a root value: a parameter, or a local that is not derived from one. A partial path
// denotes some unknown part of the data at its chain, such as what a method call or a computed key reads.
type path struct {
	root    *ast.Symbol
	chain   []string
	partial bool
}

// context is the root of paths read through `this`: state of the enclosing object, such as injected services or the
// caller's own fields. It is not input, so it never connects a check to a result, but a check reading it reads data a
// class it hands values to is not given.
var context = &ast.Symbol{Name: "this"}

func (p path) extend(key string) path {
	return path{root: p.root, chain: append(append([]string{}, p.chain...), key), partial: p.partial}
}

func related(a []string, b []string) bool {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// overlaps reports whether p and q can denote the same data: their chains are equal or one contains the other. A
// partial path overlaps a longer chain only when that chain is partial too: an unknown part of `actor` is not known to
// be `actor.id`.
func (p path) overlaps(q path) bool {
	if p.root != q.root || !related(p.chain, q.chain) {
		return false
	}
	if p.partial && !q.partial && len(q.chain) > len(p.chain) {
		return false
	}
	return !(q.partial && !p.partial && len(p.chain) > len(q.chain))
}

func partial(paths []path) []path {
	parts := make([]path, len(paths))
	for i, p := range paths {
		p.partial = true
		parts[i] = p
	}
	return parts
}

func overlapping(a []path, b []path) bool {
	for _, p := range a {
		for _, q := range b {
			if p.overlaps(q) {
				return true
			}
		}
	}
	return false
}

// rejection is a place in a function body that rejects input: a throw, or a call whose callee rejects. checked lists
// the data its conditions, or the arguments the callee checks, are built from; read lists all the data its conditions
// read, including data passed into calls whose result they test, and for a method call, an unknown part of its
// receiver, whose state the method may check.
type rejection struct {
	node    *ast.Node
	callee  *ast.Node
	checked []path
	read    []path
	// owned marks a check made by constructing an unforgeable owner, such as a value type that rejects its input. The
	// check belongs to that owner: it counts as the enclosing function checking its parameters, not as a check the
	// function makes before handing data on.
	owned bool
}

// callSite is a call or construction in a function body, the data each argument carries, and how many rejections
// precede it in source order.
type callSite struct {
	node      *ast.Node
	arguments [][]path
	prior     int
}

// summary is the analysis of one function: its rejections, the data they check in each parameter (as paths without
// a root), and the data its return statements read.
type summary struct {
	rejections []*rejection
	// readParams is like params, from the data the rejections read rather than the data they are built from.
	readParams map[int][]path
	// found lists every rejection in source order, including those whose checked data is not input.
	found   []*rejection
	calls   []callSite
	params  map[int][]path
	results []path
	// flows marks the parameters the return statements read.
	flows map[int]bool
}

type measure struct {
	ctx rule.RuleContext
	// summaries caches each analyzed function. An entry is stored as nil before the body is analyzed, so recursion
	// terminates and counts as not rejecting.
	summaries map[*ast.Node]*summary
	// stored caches whether a member stores a parameter into its class's state. An entry is stored as false before the
	// member is analyzed, so recursion terminates.
	stored map[storedKey]bool
}

func (m *measure) projectFile(file *ast.SourceFile) bool {
	return !file.IsDeclarationFile && !m.ctx.Program.IsSourceFileFromExternalLibrary(file)
}

func (m *measure) ownedSymbol(symbol *ast.Symbol) bool {
	if symbol == nil || len(symbol.Declarations) == 0 {
		return false
	}
	for _, declaration := range symbol.Declarations {
		if !m.projectFile(ast.GetSourceFileOfNode(declaration)) {
			return false
		}
	}
	return true
}

// analysis tracks one function body: which values are input (parameters and locals derived from them), and which
// data each parameter and local holds.
type analysis struct {
	m     *measure
	input map[*ast.Symbol]bool
	// aliases maps a local initialized by an access chain to that chain.
	aliases map[*ast.Symbol]path
	// locals maps every other local, and each parameter, to the data it holds.
	locals map[*ast.Symbol][]path
	// stored records data stored into parts of a local after its declaration, by assignment or by a method call.
	stored map[*ast.Symbol][]storage
	params map[*ast.Symbol]int
	// direct marks parameters declared by an identifier rather than a binding pattern.
	direct map[*ast.Symbol]bool
	found  []*rejection
	calls  []callSite
	result []path
}

type storage struct {
	chain   []string
	sources []path
}

// symbol returns the symbol an identifier refers to. The name of a shorthand property (`{ summary }`) refers to the
// variable it reads, not to the property it declares.
func (a *analysis) symbol(n *ast.Node) *ast.Symbol {
	if parent := n.Parent; parent != nil && parent.Kind == ast.KindShorthandPropertyAssignment && parent.Name() == n {
		return a.m.ctx.TypeChecker.GetShorthandAssignmentValueSymbol(parent)
	}
	return a.m.ctx.TypeChecker.GetSymbolAtLocation(n)
}

// names returns the symbols a binding name declares.
func (a *analysis) names(name *ast.Node) []*ast.Symbol {
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

func (a *analysis) taint(name *ast.Node) {
	for _, symbol := range a.names(name) {
		a.input[symbol] = true
	}
}

// declare records the data a binding name holds: base when it is initialized by an access chain, otherwise sources,
// otherwise its own root. Object pattern elements extend base by their property name; array pattern elements hold all
// of base.
func (a *analysis) declare(name *ast.Node, base *path, sources []path) {
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
			a.locals[symbol] = []path{{root: symbol}}
		}
		return
	}
	keyed := name.Kind == ast.KindObjectBindingPattern
	name.ForEachChild(func(element *ast.Node) bool {
		if element.Kind != ast.KindBindingElement || element.Name() == nil {
			return false
		}
		var key *ast.Node
		if keyed {
			key = element.PropertyName()
			if key == nil && element.Name().Kind == ast.KindIdentifier {
				key = element.Name()
			}
		}
		switch {
		case base != nil && key != nil && (key.Kind == ast.KindIdentifier || ast.IsStringOrNumericLiteralLike(key)):
			child := base.extend(key.Text())
			a.declare(element.Name(), &child, nil)
		case base != nil:
			a.declare(element.Name(), nil, []path{*base})
		default:
			a.declare(element.Name(), nil, sources)
		}
		return false
	})
}

func (a *analysis) reads(expression *ast.Node) bool {
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
func (a *analysis) exact(n *ast.Node) (path, bool) {
	n = skipOuter(n)
	if n == nil {
		return path{}, false
	}
	switch n.Kind {
	case ast.KindThisKeyword:
		return path{root: context}, true
	case ast.KindIdentifier:
		symbol := a.symbol(n)
		if symbol == nil {
			return path{}, false
		}
		if alias, ok := a.aliases[symbol]; ok {
			return alias, true
		}
		if held := a.locals[symbol]; len(held) == 1 && held[0].root == symbol && len(held[0].chain) == 0 {
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
	return path{}, false
}

func (a *analysis) primitive(n *ast.Node) bool {
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

// flow returns the data an expression's value is built from. A conditional expression's value is built from its
// branches; its condition only selects one. A method call carries part of its receiver's data. A call
// to a project function carries part of the arguments its return statements read (see carried); a call to a library
// function or a construction carries the data of its object arguments. A primitive argument to a library function, such
// as an identifier used for a lookup, carries none: the value produced from it is new data, not a transformation of the
// argument. Functions carry
// none. Inside arguments (asArgument), primitive subexpressions carry none.
func (a *analysis) flow(n *ast.Node, asArgument bool) []path {
	n = skipOuter(n)
	if n == nil || ast.IsFunctionLike(n) || ast.IsClassLike(n) || (asArgument && a.primitive(n)) {
		return nil
	}
	if p, ok := a.exact(n); ok {
		return append([]path{p}, a.storedAt(p)...)
	}
	switch n.Kind {
	case ast.KindIdentifier:
		if symbol := a.symbol(n); symbol != nil {
			return a.locals[symbol]
		}
		return nil
	case ast.KindPropertyAccessExpression:
		return partial(a.flow(n.Expression(), asArgument))
	case ast.KindConditionalExpression:
		conditional := n.AsConditionalExpression()
		return append(a.flow(conditional.WhenTrue, asArgument), a.flow(conditional.WhenFalse, asArgument)...)
	case ast.KindElementAccessExpression:
		return append(partial(a.flow(n.Expression(), asArgument)), a.flow(n.AsElementAccessExpression().ArgumentExpression, asArgument)...)
	case ast.KindCallExpression, ast.KindNewExpression:
		paths := []path{}
		callee := n.Expression()
		if n.Kind == ast.KindCallExpression && (callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression) {
			paths = append(paths, partial(a.flow(callee.Expression(), asArgument))...)
		}
		carried := a.carried(n)
		for index, argument := range n.Arguments() {
			switch {
			case carried == nil && !a.primitive(argument):
				paths = append(paths, a.flow(argument, true)...)
			case carried != nil && carried[index]:
				paths = append(paths, partial(a.flow(argument, false))...)
			}
		}
		return paths
	}
	paths := []path{}
	n.ForEachChild(func(child *ast.Node) bool {
		paths = append(paths, a.flow(child, asArgument)...)
		return false
	})
	return paths
}

// carried returns the argument positions whose data a call's result can carry, when the callee is a project function
// or method: those its return statements read, or none when it has no body to establish it, such as an interface or
// abstract method. It returns nil for library functions and constructors, whose object arguments are assumed to flow.
func (a *analysis) carried(n *ast.Node) map[int]bool {
	signature := a.m.ctx.TypeChecker.GetResolvedSignature(n)
	if signature == nil || signature.Declaration() == nil {
		return nil
	}
	declaration := signature.Declaration()
	if declaration.Kind == ast.KindConstructor || !a.m.projectFile(ast.GetSourceFileOfNode(declaration)) {
		return nil
	}
	if !ast.IsFunctionLike(declaration) || declaration.Body() == nil {
		return map[int]bool{}
	}
	if callee := a.m.summarize(declaration); callee != nil {
		return callee.flows
	}
	return map[int]bool{}
}

// store records the data stored into part of a local by an assignment or a method call on a chain rooted at it. The
// local keeps its own root, so reading another part of it does not read the stored data. A local initialized by a call
// is recorded the same way: its own root, with the call's data stored into it as a whole.
func (a *analysis) store(target *ast.Node, values []*ast.Node) {
	p, ok := a.exact(target)
	if !ok {
		return
	}
	if _, parameter := a.params[p.root]; parameter {
		return
	}
	if _, local := a.locals[p.root]; !local {
		return
	}
	sources := []path{}
	for _, value := range values {
		sources = append(sources, a.flow(value, false)...)
	}
	a.stored[p.root] = append(a.stored[p.root], storage{chain: p.chain, sources: sources})
}

// storeCall records a method call on a local as storing its arguments into it: for a project constructor or method,
// only the arguments at parameters it stores into its class's state (see stores), so a check such as
// `site.ensureMayAccess(actor)` stores nothing; for a library method such as `values.push(value)`, every argument.
func (a *analysis) storeCall(n *ast.Node) {
	values := n.Arguments()
	if _, declaration := a.m.owner(n); declaration != nil {
		values = nil
		for index, argument := range n.Arguments() {
			if a.m.stores(declaration, index) {
				values = append(values, argument)
			}
		}
	}
	a.store(n.Expression().Expression(), values)
}

// storedAt returns the data stored into the part of a local that p reads, or into parts it contains. Data stored
// into a containing part, including what a call carried into the local it initialized, is attributed to that part as
// a whole: a specific field of it is not known to hold that data.
func (a *analysis) storedAt(p path) []path {
	paths := []path{}
	for _, entry := range a.stored[p.root] {
		if related(entry.chain, p.chain) && len(p.chain) <= len(entry.chain) {
			paths = append(paths, entry.sources...)
		}
	}
	return paths
}

// narrows reports whether a condition only tests what TypeScript records in a narrowed type: negations, conjunctions,
// and disjunctions of comparisons with null or undefined, truthiness tests of values whose type is made only of object
// types, null, and undefined, `instanceof`, `in`, comparisons of `typeof`, and calls to type guards. Such a test
// establishes presence or a type, which the narrowed type of the value already carries; an object-typed index access
// without noUncheckedIndexedAccess is tested this way too.
func (a *analysis) narrows(n *ast.Node) bool {
	n = ast.SkipParentheses(n)
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		unary := n.AsPrefixUnaryExpression()
		return unary.Operator == ast.KindExclamationToken && a.narrows(unary.Operand)
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		switch binary.OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
			return a.narrows(binary.Left) && a.narrows(binary.Right)
		case ast.KindEqualsEqualsToken, ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsToken, ast.KindExclamationEqualsEqualsToken:
			return nullish(binary.Left) || nullish(binary.Right) || typeOf(binary.Left) || typeOf(binary.Right)
		case ast.KindInstanceOfKeyword, ast.KindInKeyword:
			return true
		}
		return false
	case ast.KindCallExpression:
		signature := a.m.ctx.TypeChecker.GetResolvedSignature(n)
		return signature != nil && a.m.ctx.TypeChecker.GetTypePredicateOfSignature(signature) != nil
	}
	t := a.m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil {
		return false
	}
	members := []*checker.Type{t}
	if t.Flags()&checker.TypeFlagsUnion != 0 {
		members = t.Types()
	}
	for _, member := range members {
		if member.Flags()&(checker.TypeFlagsNull|checker.TypeFlagsUndefined|checker.TypeFlagsVoid|checker.TypeFlagsObject|checker.TypeFlagsNonPrimitive) == 0 {
			return false
		}
	}
	return true
}

// tested returns the data a condition checks. Narrowing tests (see narrows) among its negations, conjunctions, and disjunctions check
// no data: in `!site || !site.teamId.equals(teamId)`, only `site.teamId` and `teamId` are checked, not `site` as a whole.
func (a *analysis) tested(n *ast.Node) []path {
	n = ast.SkipParentheses(n)
	if a.narrows(n) {
		return nil
	}
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			return a.tested(unary.Operand)
		}
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		if kind := binary.OperatorToken.Kind; kind == ast.KindAmpersandAmpersandToken || kind == ast.KindBarBarToken {
			return append(a.tested(binary.Left), a.tested(binary.Right)...)
		}
	}
	return a.flow(n, false)
}

func typeOf(n *ast.Node) bool {
	return ast.SkipParentheses(n).Kind == ast.KindTypeOfExpression
}

// testedReads returns all the data a condition reads, skipping narrowing tests like tested. Unlike tested, a call
// reads every argument, whatever its type: `isNaN(date.getTime())` reads `date`.
func (a *analysis) testedReads(n *ast.Node) []path {
	n = ast.SkipParentheses(n)
	if a.narrows(n) {
		return nil
	}
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			return a.testedReads(unary.Operand)
		}
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		if kind := binary.OperatorToken.Kind; kind == ast.KindAmpersandAmpersandToken || kind == ast.KindBarBarToken {
			return append(a.testedReads(binary.Left), a.testedReads(binary.Right)...)
		}
	}
	return a.readsAll(n)
}

// readsAll returns all the data an expression reads: like flow, but every call argument counts, as an unknown part of
// its data, since the callee may read only some of it.
func (a *analysis) readsAll(n *ast.Node) []path {
	n = skipOuter(n)
	if n == nil || ast.IsFunctionLike(n) || ast.IsClassLike(n) {
		return nil
	}
	if p, ok := a.exact(n); ok {
		return append([]path{p}, a.storedAt(p)...)
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
		paths := []path{}
		callee := n.Expression()
		if n.Kind == ast.KindCallExpression && (callee.Kind == ast.KindPropertyAccessExpression || callee.Kind == ast.KindElementAccessExpression) {
			paths = append(paths, partial(a.readsAll(callee.Expression()))...)
		}
		for _, argument := range n.Arguments() {
			paths = append(paths, partial(a.readsAll(argument))...)
		}
		return paths
	}
	paths := []path{}
	n.ForEachChild(func(child *ast.Node) bool {
		paths = append(paths, a.readsAll(child)...)
		return false
	})
	return paths
}

func nullish(n *ast.Node) bool {
	n = ast.SkipParentheses(n)
	return n.Kind == ast.KindNullKeyword || (n.Kind == ast.KindIdentifier && n.Text() == "undefined")
}

// call records a rejection when a call or new expression passes input to a project function that rejects its
// parameters at those positions. The rejection checks the parts of each argument that the callee checks of its
// parameter, or some part of an argument that is not an access chain. Calls to type guards and assertion functions
// (`asserts value`, `asserts value is T`) are narrowing tests, like `if (!value) throw`, not rejections.
func (a *analysis) call(n *ast.Node) {
	arguments := n.Arguments()
	passesInput := false
	for _, argument := range arguments {
		passesInput = passesInput || a.reads(argument)
	}
	if !passesInput {
		return
	}
	signature := a.m.ctx.TypeChecker.GetResolvedSignature(n)
	if signature == nil || signature.Declaration() == nil {
		return
	}
	if a.m.ctx.TypeChecker.GetTypePredicateOfSignature(signature) != nil {
		return
	}
	declaration := signature.Declaration()
	if !ast.IsFunctionLike(declaration) || declaration.Body() == nil || !a.m.projectFile(ast.GetSourceFileOfNode(declaration)) {
		return
	}
	callee := a.m.summarize(declaration)
	if callee == nil || len(callee.readParams) == 0 {
		return
	}
	read := a.mapped(arguments, callee.readParams)
	if len(read) == 0 {
		return
	}
	if receiver := n.Expression(); n.Kind == ast.KindCallExpression && receiver.Kind == ast.KindPropertyAccessExpression {
		read = append(read, partial(a.readsAll(receiver.Expression()))...)
	}
	if a.m.unforgeable(a.m.unwrap(a.m.ctx.TypeChecker.GetTypeAtLocation(n))) {
		a.found = append(a.found, &rejection{node: n, callee: declaration, read: read, owned: true})
		return
	}
	a.found = append(a.found, &rejection{node: n, callee: declaration, checked: a.mapped(arguments, callee.params), read: read})
}

// mapped returns the parts of each argument that a callee checks of the parameter at its position, or some part of an
// argument that is not an access path.
func (a *analysis) mapped(arguments []*ast.Node, params map[int][]path) []path {
	checked := []path{}
	for index, parts := range params {
		if index >= len(arguments) || !a.reads(arguments[index]) {
			continue
		}
		base, ok := a.exact(arguments[index])
		if !ok {
			checked = append(checked, partial(a.flow(arguments[index], false))...)
			continue
		}
		for _, part := range parts {
			p := base
			for _, key := range part.chain {
				p = p.extend(key)
			}
			p.partial = part.partial
			checked = append(checked, p)
		}
	}
	return checked
}

func (a *analysis) visit(n *ast.Node, guards []*ast.Node, inTry bool) {
	if n == nil || ast.IsFunctionLike(n) || ast.IsClassLike(n) {
		return
	}
	switch n.Kind {
	case ast.KindThrowStatement:
		if len(guards) > 0 && !inTry {
			checked := []path{}
			read := []path{}
			for _, guard := range guards {
				checked = append(checked, a.tested(guard)...)
				read = append(read, a.testedReads(guard)...)
			}
			a.found = append(a.found, &rejection{node: n, checked: checked, read: read})
		}
		return
	case ast.KindReturnStatement:
		a.visitChildren(n, guards, inTry)
		a.result = append(a.result, a.flow(n.Expression(), false)...)
		return
	case ast.KindCallExpression, ast.KindNewExpression:
		a.visitChildren(n, guards, inTry)
		site := callSite{node: n, prior: len(a.found)}
		for _, argument := range n.Arguments() {
			site.arguments = append(site.arguments, a.flow(argument, false))
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
			sources := a.flow(initializer, false)
			for _, symbol := range a.names(n.Name()) {
				a.stored[symbol] = append(a.stored[symbol], storage{sources: sources})
			}
		} else {
			a.declare(n.Name(), nil, a.flow(initializer, false))
		}
		if a.reads(initializer) {
			a.taint(n.Name())
		}
		return
	case ast.KindForOfStatement, ast.KindForInStatement:
		statement := n.AsForInOrOfStatement()
		a.visit(statement.Expression, guards, inTry)
		if list := statement.Initializer; list != nil && list.Kind == ast.KindVariableDeclarationList {
			sources := a.flow(statement.Expression, false)
			for _, declaration := range list.AsVariableDeclarationList().Declarations.Nodes {
				a.declare(declaration.Name(), nil, sources)
				if a.reads(statement.Expression) {
					a.taint(declaration.Name())
				}
			}
		}
		a.visit(statement.Statement, guards, inTry)
		return
	case ast.KindIfStatement:
		statement := n.AsIfStatement()
		a.visit(statement.Expression, guards, inTry)
		then := guards
		switch {
		case a.narrows(statement.Expression):
			then = nil
		case a.reads(statement.Expression):
			then = append(append([]*ast.Node{}, guards...), statement.Expression)
		}
		a.visit(statement.ThenStatement, then, inTry)
		otherwise := guards
		if a.reads(statement.Expression) && !a.narrows(statement.Expression) {
			otherwise = then
		}
		a.visit(statement.ElseStatement, otherwise, inTry)
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
		a.visit(statement.TryBlock, guards, true)
		a.visit(statement.CatchClause, guards, inTry)
		a.visit(statement.FinallyBlock, guards, inTry)
		return
	}
	a.visitChildren(n, guards, inTry)
}

func (a *analysis) visitChildren(n *ast.Node, guards []*ast.Node, inTry bool) {
	n.ForEachChild(func(child *ast.Node) bool {
		a.visit(child, guards, inTry)
		return false
	})
}

// inputs keeps the paths rooted at input, and with withContext, those read through `this`. Data the operation
// obtains without its input, such as the current time, does not connect a check to the result.
func (a *analysis) inputs(paths []path, withContext bool) []path {
	kept := []path{}
	for _, p := range paths {
		if a.input[p.root] || (withContext && p.root == context) {
			kept = append(kept, p)
		}
	}
	return kept
}

// relative adds the paths rooted at parameters to params, by parameter position, without their root.
func (a *analysis) relative(paths []path, params map[int][]path) {
	for _, p := range paths {
		index, ok := a.params[p.root]
		switch {
		case !ok:
		case a.direct[p.root]:
			params[index] = append(params[index], path{chain: p.chain, partial: p.partial})
		default:
			params[index] = append(params[index], path{partial: true})
		}
	}
}

// summarize analyzes fn's body, or returns nil while it is being analyzed.
func (m *measure) summarize(fn *ast.Node) *summary {
	if result, ok := m.summaries[fn]; ok {
		return result
	}
	m.summaries[fn] = nil
	a := &analysis{m: m, input: map[*ast.Symbol]bool{}, aliases: map[*ast.Symbol]path{}, locals: map[*ast.Symbol][]path{},
		stored: map[*ast.Symbol][]storage{}, params: map[*ast.Symbol]int{}, direct: map[*ast.Symbol]bool{}}
	for index, parameter := range fn.Parameters() {
		a.declare(parameter.Name(), nil, nil)
		a.taint(parameter.Name())
		for _, symbol := range a.names(parameter.Name()) {
			a.params[symbol] = index
			a.direct[symbol] = parameter.Name().Kind == ast.KindIdentifier
		}
	}
	if body := fn.Body(); body != nil && len(a.input) > 0 {
		a.visit(body, nil, false)
		if body.Kind != ast.KindBlock {
			a.result = append(a.result, a.flow(body, false)...)
		}
	}
	s := &summary{params: map[int][]path{}, readParams: map[int][]path{}, results: a.inputs(a.result, false), flows: map[int]bool{}}
	for _, p := range s.results {
		if index, ok := a.params[p.root]; ok {
			s.flows[index] = true
		}
	}
	s.found = a.found
	s.calls = a.calls
	for _, found := range a.found {
		found.checked = a.inputs(found.checked, false)
		found.read = a.inputs(found.read, true)
		if len(found.checked) > 0 {
			s.rejections = append(s.rejections, found)
		}
		a.relative(found.checked, s.params)
		a.relative(found.read, s.readParams)
	}
	m.summaries[fn] = s
	return s
}

func (m *measure) objectLike(t *checker.Type) bool {
	return t.Flags()&checker.TypeFlagsObject != 0 &&
		!m.ctx.TypeChecker.IsArrayType(t) && !checker.IsTupleType(t) &&
		!m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(t)
}

// owned reports whether the project declares t: its alias, otherwise its symbol, otherwise every member.
func (m *measure) owned(t *checker.Type) bool {
	if alias := t.Alias(); alias != nil {
		return m.ownedSymbol(alias.Symbol())
	}
	if t.Flags()&(checker.TypeFlagsUnion|checker.TypeFlagsIntersection) != 0 {
		for _, member := range t.Types() {
			if !m.owned(member) {
				return false
			}
		}
		return true
	}
	return m.ownedSymbol(t.Symbol())
}

// forgeable reports whether t is an object type, or a union or intersection of them, that is not a class instance type
// and has no private, protected, #private, or unique symbol members. A class is its instances' owner: producing one
// without its constructor takes a deliberate object literal or assertion.
func (m *measure) forgeable(t *checker.Type) bool {
	if t.Flags()&(checker.TypeFlagsUnion|checker.TypeFlagsIntersection) != 0 {
		for _, member := range t.Types() {
			if !m.forgeable(member) {
				return false
			}
		}
		return true
	}
	if !m.objectLike(t) {
		return false
	}
	if symbol := t.Symbol(); symbol != nil && symbol.Flags&ast.SymbolFlagsClass != 0 {
		return false
	}
	for _, property := range m.ctx.TypeChecker.GetPropertiesOfType(t) {
		if checker.GetDeclarationModifierFlagsFromSymbol(property)&(ast.ModifierFlagsPrivate|ast.ModifierFlagsProtected) != 0 {
			return false
		}
		for _, declaration := range property.Declarations {
			name := declaration.Name()
			if name == nil {
				continue
			}
			if name.Kind == ast.KindPrivateIdentifier {
				return false
			}
			if name.Kind == ast.KindComputedPropertyName {
				key := m.ctx.TypeChecker.GetTypeAtLocation(name.Expression())
				if key != nil && key.Flags()&checker.TypeFlagsUniqueESSymbol != 0 {
					return false
				}
			}
		}
	}
	return true
}

// unwrap replaces a promise with its awaited type and removes null and undefined.
func (m *measure) unwrap(t *checker.Type) *checker.Type {
	if t == nil {
		return nil
	}
	if awaited := checker.Checker_getAwaitedType(m.ctx.TypeChecker, t); awaited != nil {
		t = awaited
	}
	return m.ctx.TypeChecker.GetNonNullableType(t)
}

// objects reports whether t is an object type, or a union or intersection of them.
func (m *measure) objects(t *checker.Type) bool {
	if t.Flags()&(checker.TypeFlagsUnion|checker.TypeFlagsIntersection) != 0 {
		for _, member := range t.Types() {
			if !m.objects(member) {
				return false
			}
		}
		return true
	}
	return m.objectLike(t)
}

// unforgeable reports whether t is a project-owned object type that only code with access to its private state or
// brand can produce, so a check made while producing it is carried by the type.
func (m *measure) unforgeable(t *checker.Type) bool {
	return t != nil && m.objects(t) && m.owned(t) && !m.forgeable(t)
}

// stateful reports whether t holds data: an index signature, or a property that is not a method and whose type is not
// a function. A result made only of operations has no state for a check to establish.
func (m *measure) stateful(t *checker.Type) bool {
	if len(m.ctx.TypeChecker.GetIndexInfosOfType(t)) > 0 {
		return true
	}
	for _, property := range m.ctx.TypeChecker.GetPropertiesOfType(t) {
		if property.Flags&ast.SymbolFlagsMethod != 0 {
			continue
		}
		value := m.ctx.TypeChecker.GetNonNullableType(m.ctx.TypeChecker.GetTypeOfSymbol(property))
		if !m.ctx.TypeChecker.TypeHasCallOrConstructSignatures(value) {
			return true
		}
	}
	return false
}

func (m *measure) result(fn *ast.Node) *checker.Type {
	signature := m.ctx.TypeChecker.GetSignatureFromDeclaration(fn)
	if signature == nil {
		return nil
	}
	return m.unwrap(m.ctx.TypeChecker.GetReturnTypeOfSignature(signature))
}

// relative returns fileName relative to the process's working directory, the configuration directory under aexlint-typed, falling back to the program's directory.
func (m *measure) relative(fileName string) string {
	directory, err := os.Getwd()
	if err != nil {
		directory = m.ctx.Program.GetCurrentDirectory()
	}
	directory = tspath.NormalizePath(directory)
	return tspath.GetRelativePathFromDirectory(directory, fileName, tspath.ComparePathsOptions{
		UseCaseSensitiveFileNames: m.ctx.Program.UseCaseSensitiveFileNames(),
		CurrentDirectory:          directory,
	})
}

func (m *measure) line(node *ast.Node) int {
	file := ast.GetSourceFileOfNode(node)
	return scanner.GetECMALineOfPosition(file, utils.TrimNodeTextRange(file, node).Pos()) + 1
}

func displayName(fn *ast.Node) string {
	if fn.Kind == ast.KindConstructor {
		if class := fn.Parent; class != nil && class.Name() != nil {
			return "constructor of `" + class.Name().Text() + "`"
		}
		return "constructor"
	}
	if name := fn.Name(); name != nil {
		return "`" + name.Text() + "`"
	}
	return "(anonymous function)"
}

func (m *measure) evidence(found *rejection) string {
	if found.callee == nil {
		return fmt.Sprintf("throw at line %d.", m.line(found.node))
	}
	return fmt.Sprintf("call to %s (%s) at line %d.", displayName(found.callee),
		m.relative(ast.GetSourceFileOfNode(found.callee).FileName()), m.line(found.node))
}

// operation returns the name of a checked operation and the node to report. Constructors are not operations: their
// result is a class instance, which is owned.
func (m *measure) operation(n *ast.Node) (string, *ast.Node, bool) {
	switch n.Kind {
	case ast.KindFunctionDeclaration, ast.KindMethodDeclaration:
		name := n.Name()
		if name == nil || name.Kind == ast.KindComputedPropertyName {
			return "", nil, false
		}
		return name.Text(), name, true
	case ast.KindFunctionExpression, ast.KindArrowFunction:
		parent := n.Parent
		switch parent.Kind {
		case ast.KindVariableDeclaration, ast.KindPropertyAssignment, ast.KindPropertyDeclaration:
			name := parent.Name()
			if parent.Initializer() != n || name == nil || (name.Kind != ast.KindIdentifier && name.Kind != ast.KindPrivateIdentifier) {
				return "", nil, false
			}
			return name.Text(), name, true
		}
	}
	return "", nil, false
}

func (m *measure) check(n *ast.Node) {
	name, node, ok := m.operation(n)
	if !ok || n.Body() == nil {
		return
	}
	t := m.result(n)
	if t == nil || !m.forgeable(t) || !m.owned(t) || !m.stateful(t) {
		return
	}
	s := m.summarize(n)
	if s == nil {
		return
	}
	var found *rejection
	for _, candidate := range s.rejections {
		if overlapping(candidate.checked, s.results) {
			found = candidate
			break
		}
	}
	if found == nil {
		return
	}
	typeName := m.ctx.TypeChecker.TypeToString(t)
	label := "`" + name + "`"
	message := rule.RuleMessage{
		Id:          "forgeableValidatedResult",
		Description: fmt.Sprintf("%s rejects some inputs, then returns `%s`, which other code can produce without that check.", label, typeName),
		Help: fmt.Sprintf("Code that receives a value of type `%s` cannot tell whether it passed %s, and each other place that builds or changes such a value must repeat the check or skip it. Rejection: %s",
			typeName, label, m.evidence(found)),
	}
	m.ctx.ReportNode(node, message)
}

func (m *measure) run() {
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		m.check(n)
		n.ForEachChild(walk)
		return false
	}
	walk(m.ctx.SourceFile.AsNode())
}

var Rule = rule.Rule{
	Name: "aexlint/no-forgeable-validated-result",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		if options != nil {
			object, ok := options.(map[string]any)
			if !ok || len(object) != 0 {
				panic("aexlint/no-forgeable-validated-result accepts no options; omit options or use an empty object")
			}
		}
		(&measure{ctx: ctx, summaries: map[*ast.Node]*summary{}, stored: map[storedKey]bool{}}).run()
		return rule.RuleListeners{}
	},
}

// Path is an access chain from a root value, as produced by the analysis.
type Path = path

// Analyzer exposes this rule's per-function analysis to other experimental rules in this repository.
type Analyzer struct{ m *measure }

func NewAnalyzer(ctx rule.RuleContext) *Analyzer {
	return &Analyzer{m: &measure{ctx: ctx, summaries: map[*ast.Node]*summary{}, stored: map[storedKey]bool{}}}
}

// Check is a rejection of input in a function body, and the input data its conditions read.
type Check struct {
	Node *ast.Node
	Read []path
}

// Call is a call or construction in a function body, the input data each argument carries, and the checks that
// precede it in source order.
type Call struct {
	Node      *ast.Node
	Arguments [][]path
	Checks    []Check
}

// Calls returns the calls and constructions in fn's body, outside nested functions and classes.
func (z *Analyzer) Calls(fn *ast.Node) []Call {
	s := z.m.summarize(fn)
	if s == nil {
		return nil
	}
	calls := []Call{}
	for _, site := range s.calls {
		call := Call{Node: site.node, Arguments: site.arguments}
		for _, found := range s.found[:site.prior] {
			if len(found.read) > 0 && !found.owned {
				call.Checks = append(call.Checks, Check{Node: found.node, Read: found.read})
			}
		}
		calls = append(calls, call)
	}
	return calls
}

// ChecksParameter reports whether fn rejects input read from its parameter at index, directly or through calls.
func (z *Analyzer) ChecksParameter(fn *ast.Node, index int) bool {
	s := z.m.summarize(fn)
	return s != nil && len(s.readParams[index]) > 0
}

// Owner returns the class and declaration of a call's callee when it is a constructor or method with a body, declared
// in a project source file.
func (z *Analyzer) Owner(call *ast.Node) (*ast.Node, *ast.Node) { return z.m.owner(call) }

// StoresParameter reports whether a constructor or method takes its parameter at index into its class's state.
func (z *Analyzer) StoresParameter(member *ast.Node, index int) bool {
	return z.m.stores(member, index)
}

// Overlap reports whether any path of a can denote the same data as any path of b.
func Overlap(a []path, b []path) bool { return overlapping(a, b) }

// definitely reports whether p and q are known to denote the same data or one to contain the other: like overlaps,
// except that two unknown parts of one value, such as the results of two getters on it, are not known to be the same.
func (p path) definitely(q path) bool {
	return !(p.partial && q.partial) && p.overlaps(q)
}

// Exact keeps the paths that denote known data, not an unknown part of it: data contained unchanged, rather than
// derived through a method call or computation.
func Exact(paths []path) []path {
	kept := []path{}
	for _, p := range paths {
		if !p.partial {
			kept = append(kept, p)
		}
	}
	return kept
}

// Definite reports whether any path of a is known to share data with any path of b.
func Definite(a []path, b []path) bool {
	for _, p := range a {
		for _, q := range b {
			if p.definitely(q) {
				return true
			}
		}
	}
	return false
}

// String describes a path as written: its root's name and chain, prefixed with "part of" when partial.
func (p path) String() string {
	text := strings.Join(append([]string{p.root.Name}, p.chain...), ".")
	if p.partial {
		return "part of " + text
	}
	return text
}
