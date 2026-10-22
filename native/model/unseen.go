package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// Unseen lists where nodes may run code the model does not see: an await, a for await loop or a yield, which let other
// code run; iterating a value whose iterator the model does not know (see knownIteration), by a for-of loop, a spread
// or an array destructuring; a read, destructuring included, or write through an accessor that is neither a project one
// with a summary that is not opaque nor declared by the default library; a call
// or construction neither resolving to a project function with a body nor declared by the default library, or one
// resolving to a project function whose summary is opaque. Constructing a project class without a constructor, or
// calling super in one deriving from it, runs the field initializers of the classes up to one with a constructor (see
// implicit). Such code may read or change any data visible to it.
// Default library functions only reach data passed to them, and the callbacks they call are read where they are
// written, those a global collection method runs during the call where they run (see synchronous). A default library
// call given a function value written elsewhere, or calling one through call, apply or
// bind, runs that function, which counts as above. With nested set, the functions nodes declare are read too.
func (m *Analyzer) Unseen(nodes []*ast.Node, nested bool) []*ast.Node {
	found := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if !nested && ast.IsFunctionLike(c) && m.runsDuring(c) && c.Body() != nil {
			walk(c.Body())
			return false
		}
		if !nested && (ast.IsFunctionLike(c) || ast.IsClassLike(c)) {
			return false
		}
		switch c.Kind {
		case ast.KindAwaitExpression, ast.KindYieldExpression:
			found = append(found, c)
		case ast.KindForOfStatement:
			if c.AsForInOrOfStatement().AwaitModifier != nil || !m.knownIteration(c.AsForInOrOfStatement().Expression) {
				found = append(found, c)
			}
		case ast.KindPropertyAccessExpression:
			if m.unseenAccessor(c) {
				found = append(found, c)
			}
		case ast.KindBindingElement:
			if m.unseenBinding(c) {
				found = append(found, c)
			}
		case ast.KindSpreadElement:
			if p := c.Parent; p != nil && p.Kind != ast.KindObjectLiteralExpression && !m.knownIteration(c.Expression()) {
				found = append(found, c)
			}
		case ast.KindVariableDeclaration:
			if c.Name().Kind == ast.KindArrayBindingPattern && c.Initializer() != nil && !m.knownIteration(c.Initializer()) {
				found = append(found, c)
			}
		case ast.KindBinaryExpression:
			if b := c.AsBinaryExpression(); b.OperatorToken.Kind == ast.KindEqualsToken && Unwrap(b.Left).Kind == ast.KindArrayLiteralExpression && !m.knownIteration(b.Right) {
				found = append(found, c)
			}
		case ast.KindCallExpression, ast.KindNewExpression, ast.KindTaggedTemplateExpression:
			if fn := m.followed(c); fn != nil {
				if m.Summary(fn).Opaque {
					found = append(found, c)
				}
			} else if constructor, ok := m.implicit(c); ok {
				if constructor != nil && m.Summary(constructor).Opaque {
					found = append(found, c)
				}
			} else if !m.library(c) || m.handsOver(c) {
				found = append(found, c)
			}
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

// library reports whether a call resolves to a function the default library declares.
func (m *Analyzer) library(call *ast.Node) bool {
	signature := m.ctx.TypeChecker.GetResolvedSignature(call)
	if signature == nil || signature.Declaration() == nil {
		return false
	}
	return utils.IsSourceFileDefaultLibrary(m.ctx.Program, ast.GetSourceFileOfNode(signature.Declaration()))
}

// private reports whether only code the model sees can reach the data v names until to: its root is a variable of fn,
// never rebound, initialized with a new object, not referenced in the functions fn declares, and, before to, only used
// as contained allows (see contained). A construction initializing it must not let the new object escape, and a call
// initializing it must return only its own local data (see FunctionSummary.Fresh).
func (m *Analyzer) private(v Value, fn *ast.Node, to int) bool {
	if v.Root == nil || v.Root.ValueDeclaration == nil {
		return false
	}
	key := privacyKey{v.Root, fn}
	until, ok := m.privacy[key]
	if !ok {
		until = m.privateUntil(v.Root, fn)
		m.privacy[key] = until
	}
	return until >= to
}

type privacyKey struct {
	root *ast.Symbol
	fn   *ast.Node
}

// privateUntil returns the position up to which only code the model sees can reach the variable root of fn, as private
// reads it: -1 when that holds nowhere, the position of its first reference not contained otherwise, or past the end
// of fn when there is none.
func (m *Analyzer) privateUntil(root *ast.Symbol, fn *ast.Node) int {
	d := root.ValueDeclaration
	if d.Kind != ast.KindVariableDeclaration || enclosingFunction(d) != fn || m.assigned(root) || m.ProvenanceOf(d.Initializer()) != FreshValue {
		return -1
	}
	initializer := Unwrap(d.Initializer())
	if fresh := m.followed(initializer); fresh != nil && (initializer.Kind == ast.KindNewExpression && slices.Contains(m.Summary(fresh).Escapes, -1) || initializer.Kind != ast.KindNewExpression && !m.Summary(fresh).Fresh) {
		return -1
	}
	until := fn.End() + 1
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if until < 0 {
			return true
		}
		if c.Kind == ast.KindIdentifier && c.Parent != nil && c.Parent != d && m.ctx.TypeChecker.GetSymbolAtLocation(c) == root {
			switch {
			case !m.runsIn(c, fn):
				until = -1
			case c.Pos() < until && !m.contained(c):
				until = c.Pos()
			}
			return until < 0
		}
		c.ForEachChild(walk)
		return until < 0
	}
	walk(fn.Body())
	return until
}

// contained reports whether a reference only reads through its data or calls on it, as private allows: the head of a
// property or element read giving a primitive, or the target of a write, or of one giving data that is itself used as
// these terms allow; the receiver of a project method whose summary lets no receiver escape and that is not a closure
// (see closure), whose variables may hold the same data, or of a default library method whose result is a primitive or
// unused; returned, which ends the function; an argument to a project function whose summary lets that argument
// neither escape nor be carried into other data (see escapes), or to a default library call that carries it nowhere,
// runs no function value given to it (see handsOver) and whose result is a primitive or unused.
func (m *Analyzer) contained(c *ast.Node) bool {
	p := c.Parent
	for p.Kind == ast.KindParenthesizedExpression || p.Kind == ast.KindNonNullExpression {
		c, p = p, p.Parent
	}
	switch {
	case (p.Kind == ast.KindPropertyAccessExpression || p.Kind == ast.KindElementAccessExpression) && p.Expression() == c:
		call := p.Parent
		if call != nil && call.Kind == ast.KindCallExpression && call.Expression() == p {
			if fn := m.followed(call); fn != nil {
				return !slices.Contains(m.Summary(fn).Escapes, -1) && !m.closure(fn)
			}
			return m.library(call) && m.dropped(call)
		}
		if call != nil && ast.IsAssignmentExpression(call, true) && Unwrap(call.AsBinaryExpression().Left) == p || m.primitive(p) {
			return true
		}
		return m.contained(p)
	case p.Kind == ast.KindReturnStatement:
		return true
	case p.Kind == ast.KindCallExpression && slices.Contains(p.Arguments(), c):
		fn := m.followed(p)
		if fn == nil {
			return m.library(p) && !m.handsOver(p) && m.dropped(p) && !slices.ContainsFunc(m.EffectsAt(p), func(e Effect) bool { return slices.Contains(e.Carried, c) })
		}
		summary := m.Summary(fn)
		position := slices.Index(p.Arguments(), c)
		return !slices.Contains(summary.Escapes, position) && !slices.ContainsFunc(summary.Effects, func(e SummaryEffect) bool { return slices.Contains(e.Carried, position) })
	}
	return false
}

// dropped reports whether a call's result keeps none of what it is given: it is a primitive, or nothing uses it.
func (m *Analyzer) dropped(call *ast.Node) bool {
	return m.primitive(call) || discarded(call)
}

// closure reports whether fn uses a variable or parameter another function declares, other than a const holding a
// primitive: its summary cannot name what that variable holds.
func (m *Analyzer) closure(fn *ast.Node) bool {
	found := false
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if found {
			return true
		}
		if c.Kind == ast.KindIdentifier && c.Parent != nil {
			if symbol := m.referenced(c); symbol != nil && symbol.ValueDeclaration != nil {
				d := symbol.ValueDeclaration
				outer := (d.Kind == ast.KindVariableDeclaration || d.Kind == ast.KindParameter || d.Kind == ast.KindBindingElement) && !encloses(fn, d) && enclosingFunction(d) != nil
				fixed := d.Kind == ast.KindVariableDeclaration && ast.IsVarConst(d) && m.primitive(c)
				found = outer && !fixed
			}
			return found
		}
		c.ForEachChild(walk)
		return found
	}
	if body := fn.Body(); body != nil {
		walk(body)
	}
	return found
}

// implicit reads a construction, or a super call, of a project class without a constructor: the classes up to the
// first with a constructor run their field initializers, which must run no code the model does not see, and that
// constructor, returned, runs with the arguments; nil when no class has one, or when the first base outside the
// project is declared by the default library.
func (m *Analyzer) implicit(c *ast.Node) (*ast.Node, bool) {
	var class *ast.Node
	switch {
	case c.Kind == ast.KindNewExpression:
		class = m.classOf(c.Expression())
	case c.Kind == ast.KindCallExpression && Unwrap(c.Expression()).Kind == ast.KindSuperKeyword:
		if container := ast.GetContainingClass(c); container != nil {
			class = m.baseOf(container)
		}
	}
	for seen := map[*ast.Node]bool{}; class != nil && !seen[class]; class = m.baseOf(class) {
		seen[class] = true
		if !ProjectFile(ast.GetSourceFileOfNode(class)) {
			return nil, utils.IsSourceFileDefaultLibrary(m.ctx.Program, ast.GetSourceFileOfNode(class))
		}
		for _, member := range class.Members() {
			if member.Kind == ast.KindConstructor && member.Body() != nil {
				return member, true
			}
		}
		initializers := []*ast.Node{}
		for _, member := range class.Members() {
			if member.Kind == ast.KindPropertyDeclaration && !ast.IsStatic(member) && member.Initializer() != nil {
				initializers = append(initializers, member.Initializer())
			}
		}
		if len(m.Unseen(initializers, false)) > 0 {
			return nil, false
		}
		if ast.GetClassExtendsHeritageElement(class) == nil {
			return nil, true
		}
	}
	return nil, false
}

// classOf returns the class declaration n names.
func (m *Analyzer) classOf(n *ast.Node) *ast.Node {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(Unwrap(n))
	if symbol == nil {
		return nil
	}
	if symbol.Flags&ast.SymbolFlagsAlias != 0 {
		symbol = m.ctx.TypeChecker.GetAliasedSymbol(symbol)
	}
	for _, d := range symbol.Declarations {
		if ast.IsClassLike(d) {
			return d
		}
	}
	return nil
}

// baseOf returns the class declaration a class extends.
func (m *Analyzer) baseOf(class *ast.Node) *ast.Node {
	heritage := ast.GetClassExtendsHeritageElement(class)
	if heritage == nil {
		return nil
	}
	return m.classOf(heritage.Expression())
}

// knownIteration reports whether iterating n runs only code the model knows: a default library method's result on such
// a value, a call to a project function with a body, a generator's summary reading what its body does, or a value every
// member of whose type, read through its constraint when generic, is a string or has its `[Symbol.iterator]` declared
// by a global collection type, such as an array, a subclass of Map or an interface extending Array.
func (m *Analyzer) knownIteration(n *ast.Node) bool {
	n = Unwrap(n)
	if n.Kind == ast.KindCallExpression {
		if m.followed(n) != nil {
			return true
		}
		if callee := Unwrap(n.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
			if _, _, ok := m.libraryMember(callee.Name()); ok && m.knownIteration(callee.Expression()) {
				return true
			}
		}
	}
	tc := m.ctx.TypeChecker
	t := tc.GetNonNullableType(tc.GetTypeAtLocation(n))
	if t == nil {
		return false
	}
	members := []*checker.Type{t}
	if t.Flags()&checker.TypeFlagsUnion != 0 {
		members = t.Types()
	}
	return !slices.ContainsFunc(members, func(member *checker.Type) bool { return !m.knownIterator(member) })
}

// knownIterator reports whether t is a string, or has its `[Symbol.iterator]` declared by a global collection or typed
// array type.
func (m *Analyzer) knownIterator(t *checker.Type) bool {
	tc := m.ctx.TypeChecker
	if t.Flags()&checker.TypeFlagsInstantiable != 0 {
		if t = checker.Checker_getBaseConstraintOfType(tc, t); t == nil {
			return false
		}
	}
	if t.Flags()&checker.TypeFlagsStringLike != 0 {
		return true
	}
	iterator := tc.GetPropertyOfType(t, checker.Checker_getPropertyNameForKnownSymbolName(tc, "iterator"))
	if iterator == nil || len(iterator.Declarations) == 0 {
		return false
	}
	return !slices.ContainsFunc(iterator.Declarations, func(d *ast.Node) bool {
		owner := d.Parent
		return owner == nil || !slices.ContainsFunc(slices.Concat(collectionTypes, typedArrays), func(name string) bool {
			global := m.globalType(name)
			return global != nil && slices.Contains(global.Declarations, owner)
		})
	})
}

// typedArrays names the global typed array types, which iterate their elements as arrays do.
var typedArrays = []string{"Int8Array", "Uint8Array", "Uint8ClampedArray", "Int16Array", "Uint16Array", "Int32Array",
	"Uint32Array", "Float32Array", "Float64Array", "BigInt64Array", "BigUint64Array"}

// unseenAccessor reports whether a property access runs an accessor the model does not see: the getter it reads
// through, or the setter it is assigned through (see unseenAccessorOf).
func (m *Analyzer) unseenAccessor(n *ast.Node) bool {
	kind := ast.KindGetAccessor
	if p := n.Parent; p != nil && ast.IsAssignmentExpression(p, true) && Unwrap(p.AsBinaryExpression().Left) == n {
		kind = ast.KindSetAccessor
	}
	return m.unseenAccessorOf(m.ctx.TypeChecker.GetSymbolAtLocation(n.Name()), kind)
}

// unseenBinding reports whether an element of an object binding pattern reads its property through a getter the
// model does not see.
func (m *Analyzer) unseenBinding(n *ast.Node) bool {
	pattern := n.Parent
	if pattern == nil || pattern.Kind != ast.KindObjectBindingPattern {
		return false
	}
	name := n.AsBindingElement().PropertyName
	if name == nil {
		name = n.Name()
	}
	if name == nil || name.Kind != ast.KindIdentifier {
		return false
	}
	tc := m.ctx.TypeChecker
	object := tc.GetTypeAtLocation(pattern)
	if object == nil {
		return false
	}
	return m.unseenAccessorOf(tc.GetPropertyOfType(tc.GetNonNullableType(object), name.Text()), ast.KindGetAccessor)
}

// unseenAccessorOf reports whether running the accessor of kind that symbol declares runs code the model does not
// see: it is neither a project accessor with a body whose summary is not opaque nor declared by the default library.
func (m *Analyzer) unseenAccessorOf(symbol *ast.Symbol, kind ast.Kind) bool {
	if symbol == nil || symbol.Flags&ast.SymbolFlagsAccessor == 0 {
		return false
	}
	for _, d := range symbol.Declarations {
		if d.Kind != kind {
			continue
		}
		file := ast.GetSourceFileOfNode(d)
		if utils.IsSourceFileDefaultLibrary(m.ctx.Program, file) {
			return false
		}
		if d.Body() != nil && ProjectFile(file) && !m.overridden(d) {
			return m.Summary(d).Opaque
		}
		return true
	}
	return false
}

// handsOver reports whether a default library call runs a function value the model does not see: an argument, or the
// receiver of Function's call, apply or bind, that has call signatures, is not a function written in place, and does
// not name a project function with a body whose summary is not opaque and makes no effect, the library deciding what
// it acts on, nor a function the default library declares, such as Number or Boolean.
func (m *Analyzer) handsOver(call *ast.Node) bool {
	if call.Kind != ast.KindCallExpression {
		return false
	}
	values := slices.Clone(call.Arguments())
	if callee := Unwrap(call.Expression()); callee.Kind == ast.KindPropertyAccessExpression {
		if symbol := m.ctx.TypeChecker.GetSymbolAtLocation(callee.Name()); symbol != nil && len(symbol.Declarations) > 0 {
			d := symbol.Declarations[0]
			if m.ownedBy(d, "Function") || m.ownedBy(d, "CallableFunction") || m.ownedBy(d, "NewableFunction") {
				values = append(values, callee.Expression())
			}
		}
	}
	tc := m.ctx.TypeChecker
	return slices.ContainsFunc(values, func(v *ast.Node) bool {
		v = Unwrap(v)
		if v.Kind == ast.KindSpreadElement || ast.IsFunctionLike(v) {
			return false
		}
		t := tc.GetTypeAtLocation(v)
		if t == nil || len(tc.GetSignaturesOfType(tc.GetNonNullableType(t), checker.SignatureKindCall)) == 0 {
			return false
		}
		if fn := m.functionNamed(v); fn != nil {
			summary := m.Summary(fn)
			return summary.Opaque || len(summary.Effects) > 0
		}
		name := v
		if v.Kind == ast.KindPropertyAccessExpression {
			name = v.Name()
		}
		return !utils.IsSymbolFromDefaultLibrary(m.ctx.Program, m.ctx.TypeChecker.GetSymbolAtLocation(name))
	})
}

// functionNamed returns the project function with a body that an identifier or property access names.
func (m *Analyzer) functionNamed(n *ast.Node) *ast.Node {
	if n.Kind != ast.KindIdentifier && n.Kind != ast.KindPropertyAccessExpression {
		return nil
	}
	name := n
	if n.Kind == ast.KindPropertyAccessExpression {
		name = n.Name()
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(name)
	if symbol != nil && symbol.Flags&ast.SymbolFlagsAlias != 0 {
		symbol = m.ctx.TypeChecker.GetAliasedSymbol(symbol)
	}
	if symbol == nil {
		return nil
	}
	for _, d := range symbol.Declarations {
		if ast.IsFunctionLike(d) && d.Body() != nil && ProjectFile(ast.GetSourceFileOfNode(d)) {
			return d
		}
		if d.Kind == ast.KindVariableDeclaration && ast.IsVarConst(d) && d.Initializer() != nil {
			if fn := Unwrap(d.Initializer()); ast.IsFunctionLike(fn) && fn.Body() != nil {
				return fn
			}
		}
	}
	return nil
}

// local reports whether only fn's own variables can reach the data root holds, read as the greatest set of variables
// meeting these terms together, so a variable and the container it is stored in can vouch for each other: root is a
// variable of fn, rebound or not, whose every definition gives a new object or an entry of a local container (see
// Contents) holding only such data, and whose every reference is contained (see contained), rebinds it, or stores it
// into such a container. A for-of loop over such a container binds variables that must meet the same terms. A local
// container holding only such data is local itself: it is only used as a container (see Contents), and what comes out
// of it is local. Unlike private data, local data may be rebound and come back out of containers, but it is read for
// the whole function.
func (m *Analyzer) local(root *ast.Symbol, fn *ast.Node) bool {
	key := privacyKey{root, fn}
	if known, ok := m.locals[key]; ok {
		return known
	}
	m.locals[key] = true
	from := len(m.assuming)
	m.assuming = append(m.assuming, key)
	result := m.localOf(root, fn)
	if !result {
		for _, assumed := range m.assuming[from+1:] {
			delete(m.locals, assumed)
		}
		m.assuming = m.assuming[:from+1]
	}
	m.locals[key] = result
	if from == 0 {
		m.assuming = m.assuming[:0]
	}
	return result
}

func (m *Analyzer) localOf(root *ast.Symbol, fn *ast.Node) bool {
	d := root.ValueDeclaration
	if d == nil || enclosingFunction(d) != fn || fn.Body() == nil {
		return false
	}
	if d.Kind == ast.KindVariableDeclaration && m.empty(d.Initializer()) && m.localContainer(Value{Root: root, Path: []string{}}, d, fn) {
		return true
	}
	switch {
	case d.Kind == ast.KindVariableDeclaration && d.Parent != nil && d.Parent.Kind == ast.KindCatchClause:
		return false
	case d.Kind == ast.KindVariableDeclaration:
		if d.Initializer() != nil && !m.madeLocally(d.Initializer(), fn) {
			return false
		}
	case d.Kind == ast.KindBindingElement:
		loop := d
		for loop != nil && loop.Kind != ast.KindForOfStatement {
			loop = loop.Parent
		}
		if loop == nil || !encloses(loop.AsForInOrOfStatement().Initializer, d) {
			return false
		}
		over, _ := m.iteration(loop.AsForInOrOfStatement().Expression)
		if over.Root == nil || !m.localContainer(over, loop, fn) {
			return false
		}
	default:
		return false
	}
	if d.Kind == ast.KindVariableDeclaration && d.Parent != nil && d.Parent.Parent != nil && d.Parent.Parent.Kind == ast.KindForOfStatement {
		over, _ := m.iteration(d.Parent.Parent.AsForInOrOfStatement().Expression)
		if over.Root == nil || !m.localContainer(over, d, fn) {
			return false
		}
	}
	for _, w := range writesIn([]*ast.Node{fn.Body()}, true) {
		target := Unwrap(w.Target)
		if w.Rebinds() && target.Kind == ast.KindIdentifier && m.SymbolOf(target) == root && (w.Source == nil || !m.madeLocally(w.Source, fn)) {
			return false
		}
	}
	local := true
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if !local {
			return true
		}
		if c.Kind == ast.KindIdentifier && c.Parent != nil && m.referenced(c) == root {
			local = m.runsIn(c, fn) && m.keptLocally(c, fn)
			return !local
		}
		c.ForEachChild(walk)
		return !local
	}
	walk(fn.Body())
	return local
}

// madeLocally reports whether n gives a new object, an entry of a local container holding only local data, or local
// data, through conditional and logical operators. A construction gives a new object when its constructor lets no
// receiver escape; a call to a project function does when its summary returns only its own local data (see
// FunctionSummary.Fresh), as otherwise it may keep what it returns.
func (m *Analyzer) madeLocally(n *ast.Node, fn *ast.Node) bool {
	n = Unwrap(n)
	switch n.Kind {
	case ast.KindConditionalExpression:
		c := n.AsConditionalExpression()
		return m.madeLocally(c.WhenTrue, fn) && m.madeLocally(c.WhenFalse, fn)
	case ast.KindBinaryExpression:
		if b := n.AsBinaryExpression(); ast.IsLogicalOperator(b.OperatorToken.Kind) || b.OperatorToken.Kind == ast.KindQuestionQuestionToken {
			return m.madeLocally(b.Left, fn) && m.madeLocally(b.Right, fn)
		}
	case ast.KindIdentifier:
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n)
		return symbol != nil && m.local(symbol, fn)
	case ast.KindCallExpression, ast.KindElementAccessExpression:
		if lookup, ok := m.LookupOf(n); ok && lookup.Container.Root != nil {
			return m.localContainer(lookup.Container, n, fn)
		}
	}
	if m.ProvenanceOf(n) != FreshValue || m.primitive(n) {
		return false
	}
	if n.Kind == ast.KindNewExpression {
		if constructor := m.followed(n); constructor != nil {
			return !slices.Contains(m.Summary(constructor).Escapes, -1)
		}
		_, implicit := m.implicit(n)
		return implicit || m.library(n)
	}
	if fn := m.followed(n); fn != nil {
		return m.Summary(fn).Fresh
	}
	return true
}

// localContainer reports whether container is a local container of fn whose every entry stores local data.
func (m *Analyzer) localContainer(container Value, at, fn *ast.Node) bool {
	contents, ok := m.Contents(container, at)
	if !ok {
		return false
	}
	return !slices.ContainsFunc(contents.Entries, func(e Entry) bool { return !m.madeLocally(e.Value, fn) })
}

// keptLocally reports whether a reference to local data keeps it local: contained, rebinding it, or storing it into a
// local container holding only local data.
func (m *Analyzer) keptLocally(c, fn *ast.Node) bool {
	p := c.Parent
	if ast.IsAssignmentExpression(p, true) && Unwrap(p.AsBinaryExpression().Left) == c {
		return true
	}
	if p.Kind == ast.KindVariableDeclaration && p.Name() == c {
		return true
	}
	for _, e := range m.EffectsAt(p) {
		if (e.Kind == Store || e.Kind == Append) && e.Stored != nil && Unwrap(e.Stored) == c && e.Receiver.Root != nil {
			return m.localContainer(e.Receiver, p, fn)
		}
	}
	return m.contained(c)
}

// unreachable reports whether no code outside fn can reach data v before to: it is private (see private) or local
// (see local) to fn.
func (m *Analyzer) unreachable(v Value, fn *ast.Node, to int) bool {
	return m.private(v, fn, to) || v.Root != nil && v.This == nil && m.local(v.Root, fn)
}

// Local reports whether only fn's own variables can reach the data the variable symbol holds (see local).
func (m *Analyzer) Local(symbol *ast.Symbol, fn *ast.Node) bool {
	return symbol != nil && fn != nil && m.local(symbol, fn)
}

// runsIn reports whether code c runs as fn's own: written in fn, or in functions running during the expression fn gives
// them to (see runsDuring).
func (m *Analyzer) runsIn(c, fn *ast.Node) bool {
	for p := enclosingFunction(c); p != nil; p = enclosingFunction(p) {
		if p == fn {
			return true
		}
		if !m.runsDuring(p) {
			return false
		}
	}
	return false
}
