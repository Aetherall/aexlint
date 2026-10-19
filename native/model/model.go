package model

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type Analyzer struct {
	ctx             rule.RuleContext
	sites           map[*ast.Node]*Site
	vocabs          map[*checker.Type]*Vocabulary
	truthiness      map[*checker.Type]bool
	htmlAll         *checker.Type
	htmlAllResolved bool
	flows           map[*ast.Node]*Flow
	stored          map[storedKey]bool
	aliases         map[*ast.Symbol]*Value
	control         *control
	writing         *control
	globals         map[string]*ast.Symbol
	summaries       map[*ast.Node]FunctionSummary
	overwritten     map[*ast.Node][]Overwritten
	unaddressed     map[*ast.Node][]UnaddressedKey
}

// NewAnalyzer reads one invocation's program with its checker.
func NewAnalyzer(ctx rule.RuleContext) *Analyzer {
	return &Analyzer{ctx: ctx, sites: map[*ast.Node]*Site{}, vocabs: map[*checker.Type]*Vocabulary{}, truthiness: map[*checker.Type]bool{}, flows: map[*ast.Node]*Flow{}, stored: map[storedKey]bool{}, aliases: map[*ast.Symbol]*Value{}, summaries: map[*ast.Node]FunctionSummary{}, overwritten: map[*ast.Node][]Overwritten{}, unaddressed: map[*ast.Node][]UnaddressedKey{}}
}

// DecisionForm is how many arms of a decision do work: at most one for a guard, several for a dispatch.
type DecisionForm string

const (
	GuardForm    DecisionForm = "guard"
	DispatchForm DecisionForm = "dispatch"
)

// FunctionKind is what a function answers: a selector or a declared type predicate answers a question about its
// parameters.
type FunctionKind string

const (
	PlainFunction     FunctionKind = "function"
	SelectorFunction  FunctionKind = "selector"
	PredicateFunction FunctionKind = "predicate"
)

// Branch is an arm of a decision: Claim holds where it runs. Tested is false for the arm reached when no test holds.
type Branch struct {
	Claim       Claim
	Complete    bool
	Tested      bool
	Body        []*ast.Node
	Outcome     Outcome
	ControlFlow []*ast.Node
}

type Decision struct {
	ID         int
	Node       *ast.Node
	Line       int
	Form       DecisionForm
	Function   int
	Field      Field
	Vocabulary *Vocabulary
	Arms       []Branch
	// Receiver is the expression or binding pattern whose type holds the tested field, Object the value it reads, and Tag
	// the field's name.
	Receiver    *ast.Node
	Object      Value
	Tag         string
	Owner       *ast.Symbol
	Inside      bool
	Root        *ast.Node
	Alternative []*ast.Node
	// Following is the return statement after a run of ifs or a switch without default whose tested arms all leave: it
	// runs for the values no arm accepts. It is not one of Arms, and nil when that statement is not a return.
	Following *Branch
}

// EntryNarrowing is a guard a function makes on a literal-valued part of a parameter before its work, possibly through a
// forwarding call, and who narrows it: its own guard, or a sibling selector whose answer already implies it. Its test
// claims what holds after the guard.
type EntryNarrowing struct {
	Test
	Property  string
	Selector  string
	Handler   string
	Container *ast.Node
}

// Function is a function with a body. A selector or predicate answers a question about its parameters: Answer holds
// when it returns true.
type Function struct {
	ID         int
	Node       *ast.Node
	Line       int
	Name       string
	Kind       FunctionKind
	Answer     []Test
	Narrowings []EntryNarrowing
}

// Gate is a selector and a handler of one object or class whose first parameters have the same type: code calling the
// handler with a value the selector accepted passes that value across the gate.
type Gate struct {
	Selector  *Function
	Handler   *Function
	Names     [2]string
	Container *ast.Node
}

type File struct {
	Functions []*Function
	Decisions []*Decision
	Gates     []Gate
}

type builder struct {
	File
	m        *Analyzer
	source   *ast.SourceFile
	ids      map[*ast.Node]int
	analyzed map[*ast.Node][]EntryNarrowing
}

func newBuilder(a *Analyzer, source *ast.SourceFile) *builder {
	md := &builder{File: File{Functions: []*Function{}, Decisions: []*Decision{}}, m: a, source: source, ids: map[*ast.Node]int{}, analyzed: map[*ast.Node][]EntryNarrowing{}}
	root := source.AsNode()
	md.collectFunctions(root)
	md.visit(root)
	return md
}

func Dispatches(a *Analyzer) []*Decision {
	return DispatchesIn(a, a.ctx.SourceFile)
}

func DispatchesIn(a *Analyzer, source *ast.SourceFile) []*Decision {
	return newBuilder(a, source).Decisions
}

func Build(a *Analyzer) *File {
	md := newBuilder(a, a.ctx.SourceFile)
	root := md.source.AsNode()
	for _, f := range md.Functions {
		md.selector(f)
	}
	for _, f := range md.Functions {
		f.Narrowings = md.entryNarrowings(f.Node)
	}
	md.containers(root)
	return &md.File
}

type arm struct {
	claim     Claim
	condition *ast.Node
	body      []*ast.Node
	unread    bool
}

func (md *builder) line(n *ast.Node) int {
	sf := md.source
	return scanner.GetECMALineOfPosition(sf, scanner.SkipTrivia(sf.Text(), n.Pos())) + 1
}

func (md *builder) typeName(t *checker.Type) string {
	if t == nil {
		return ""
	}
	if s := t.Alias().Symbol(); s != nil {
		return s.Name
	}
	if s := t.Symbol(); s != nil && !strings.HasPrefix(s.Name, ast.InternalSymbolNamePrefix) {
		return s.Name
	}
	return md.m.ctx.TypeChecker.TypeToString(t)
}

func (md *builder) label(n *ast.Node) string {
	n = Unwrap(n)
	tc := md.m.ctx.TypeChecker
	if n.Kind == ast.KindPropertyAccessExpression {
		for _, t := range []*checker.Type{md.m.ReceiverType(n.Expression()), tc.GetTypeAtLocation(n.Expression())} {
			if t != nil {
				return md.typeName(tc.GetNonNullableType(t)) + "." + n.Name().Text()
			}
		}
		return ""
	}
	if symbol := tc.GetSymbolAtLocation(n); symbol != nil && symbol.ValueDeclaration != nil {
		return symbol.Name
	}
	return ""
}

// exits reports whether n always leaves the statement list holding it: no way through it reaches its end.
func (m *Analyzer) exits(n *ast.Node) bool {
	return n != nil && m.flowControl().of(n).end == nil
}

func controlFlow(nodes []*ast.Node) []*ast.Node {
	found := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) && n.Kind != ast.KindArrowFunction && n.Kind != ast.KindFunctionExpression {
			return false
		}
		switch n.Kind {
		case ast.KindIfStatement, ast.KindSwitchStatement, ast.KindConditionalExpression, ast.KindTryStatement,
			ast.KindForStatement, ast.KindForOfStatement, ast.KindForInStatement, ast.KindWhileStatement, ast.KindDoStatement:
			found = append(found, n)
		}
		n.ForEachChild(walk)
		return false
	}
	for _, n := range nodes {
		if n != nil {
			walk(n)
		}
	}
	return found
}

func (md *builder) enclosing(n *ast.Node) int {
	for p := n.Parent; p != nil; p = p.Parent {
		if id, ok := md.ids[p]; ok {
			return id
		}
	}
	return -1
}

func (md *builder) add(d *Decision) *Decision {
	d.ID = len(md.Decisions)
	md.Decisions = append(md.Decisions, d)
	return d
}

func (md *builder) record(root, subjectNode *ast.Node, vocabulary *Vocabulary, arms []arm, alternative []*ast.Node, following *ast.Node) {
	if len(arms) == 0 {
		return
	}
	field := md.m.FieldOf(subjectNode)
	value, tested := md.m.ValueOf(subjectNode), ""
	if n := len(value.Path); n > 0 {
		value, tested = Value{Node: value.Node, Root: value.Root, This: value.This, Base: value.Base, Path: value.Path[:n-1]}, value.Path[n-1]
	}
	d := &Decision{Node: Unwrap(subjectNode), Line: md.line(subjectNode), Function: md.enclosing(root), Field: field, Vocabulary: vocabulary}
	left := md.left(root, Test{Value: md.m.ValueOf(subjectNode), Claim: Claim{Domain: vocabulary.Values, Vocabulary: vocabulary}})
	for i := range arms {
		arms[i].claim = arms[i].claim.And(left)
	}
	pure := true
	covered := Claim{Domain: vocabulary.Values, Accepted: []string{}, Vocabulary: vocabulary}
	for _, a := range arms {
		if a.unread {
			pure = false
			continue
		}
		if a.condition != nil {
			_, whole := md.m.literalTest(a.condition)
			pure = pure && whole
		}
		complete := pure && len(a.claim.And(covered).Accepted) == 0
		covered = covered.Or(a.claim)
		d.Arms = append(d.Arms, Branch{Claim: a.claim, Complete: complete, Tested: true, Body: a.body, Outcome: md.m.OutcomeOf(a.body, value, tested), ControlFlow: controlFlow(a.body)})
	}
	if alternative != nil && pure {
		held, _ := md.m.Holds(subjectNode)
		missing := held&(AbsentUndefined|AbsentNull) != 0
		d.Arms = append(d.Arms, Branch{Claim: covered.Not().And(left), Complete: !missing, Body: alternative, Outcome: md.m.OutcomeOf(alternative, value, tested), ControlFlow: controlFlow(alternative)})
	}
	if following != nil && alternative == nil && pure && md.armsLeave(root, d.Arms) {
		held, _ := md.m.Holds(subjectNode)
		missing := held&(AbsentUndefined|AbsentNull) != 0
		body := []*ast.Node{following}
		d.Following = &Branch{Claim: covered.Not().And(left), Complete: !missing, Body: body, Outcome: md.m.OutcomeOf(body, value, tested), ControlFlow: controlFlow(body)}
	}
	work := 0
	for _, a := range d.Arms {
		if !a.Outcome.Fallback() {
			work++
		}
	}
	d.Form = GuardForm
	if work >= 2 {
		d.Form = DispatchForm
	}
	d.Root = root
	d.Alternative = alternative
	if receiver := md.m.objectNode(subjectNode); receiver != nil && tested != "" {
		d.Receiver, d.Object, d.Tag = receiver, value, tested
		if d.Owner = md.m.Owner(receiver); d.Owner != nil {
			d.Inside = md.m.InsideOwner(Unwrap(subjectNode), d.Owner)
		}
	}
	md.add(d)
}

// armsLeave reports whether every arm leaves the statement list holding root without reaching the statement after it:
// a break leaving a switch reaches it.
func (md *builder) armsLeave(root *ast.Node, arms []Branch) bool {
	return !slices.ContainsFunc(arms, func(b Branch) bool {
		e := md.m.listEnding(b.Body)
		return len(b.Body) == 0 || e.end != nil || e.breaks[root] != nil
	})
}

// next returns the statement after last in the statement list holding it, when it is a return statement.
func next(last *ast.Node) *ast.Node {
	list := statementList(last.Parent)
	i := slices.Index(list, last)
	if i < 0 || i+1 >= len(list) || list[i+1].Kind != ast.KindReturnStatement {
		return nil
	}
	return list[i+1]
}

// objectNode returns the expression or binding pattern whose type holds the field subject reads: the object of a
// property or literal-key element read, the object a variable is destructured from when it is a plain read, or else the
// pattern it is bound in, as for a parameter or a cast, or the object a const copies the field from. It returns nil when
// subject reads no field.
func (m *Analyzer) objectNode(subject *ast.Node) *ast.Node {
	switch u := Unwrap(subject); u.Kind {
	case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
		return Unwrap(u.Expression())
	case ast.KindIdentifier:
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(u)
		if symbol == nil {
			return nil
		}
		if _, ok := m.alias(symbol); !ok {
			return nil
		}
		switch d := symbol.ValueDeclaration; d.Kind {
		case ast.KindBindingElement:
			if declaration := d.Parent.Parent; declaration.Kind == ast.KindVariableDeclaration && declaration.Initializer() != nil {
				if object := ast.SkipParentheses(declaration.Initializer()); object.Kind == ast.KindIdentifier || object.Kind == ast.KindThisKeyword || object.Kind == ast.KindPropertyAccessExpression {
					return object
				}
			}
			return d.Parent
		case ast.KindVariableDeclaration:
			return m.objectNode(d.Initializer())
		}
	}
	return nil
}

// left returns the values of subject a decision at root can still see: earlier statements of the list holding root
// that leave when their whole condition, a test of subject, holds have already taken their values, unless a statement
// after them assigns the subject or a value it is read from.
func (md *builder) left(root *ast.Node, subject Test) Claim {
	left := Claim{Domain: subject.Claim.Domain, Accepted: subject.Claim.Domain, Vocabulary: subject.Claim.Vocabulary}
	statement := root
	for statement.Parent != nil && statementList(statement.Parent) == nil && !ast.IsFunctionLike(statement.Parent) {
		statement = statement.Parent
	}
	if statement.Parent == nil || statementList(statement.Parent) == nil {
		return left
	}
	list := statementList(statement.Parent)
	for i, earlier := range list {
		if earlier == statement {
			break
		}
		if earlier.Kind != ast.KindIfStatement || earlier.AsIfStatement().ElseStatement != nil || !md.m.exits(earlier.AsIfStatement().ThenStatement) {
			continue
		}
		if t, ok := md.m.literalTest(earlier.AsIfStatement().Expression); ok && SameSubject(t, subject) && !md.m.assigns(list[i+1:slices.Index(list, statement)], subject.Value) {
			left = left.And(t.Claim.Not())
		}
	}
	return left
}

// assigns reports whether nodes rebind v, or a value v is read from, by assignment, update, destructuring or as the
// variable of a for-in or for-of loop, including inside the functions they declare.
func (m *Analyzer) assigns(nodes []*ast.Node, v Value) bool {
	return slices.ContainsFunc(writesIn(nodes, true), func(w Write) bool { return w.Rebinds() && v.Within(m.ValueOf(w.Target)) })
}

// matching returns the first test of literal values in condition, or the first on the same subject as first, whose
// condition restricts that subject, with what the condition implies about it.
func (md *builder) matching(condition *ast.Node, first *Test) (Test, Claim, bool) {
	for _, t := range md.m.Tests(condition) {
		if first != nil && !SameSubject(t, *first) {
			continue
		}
		if implied := md.m.Implied(condition, t); len(implied.Accepted) < len(implied.Domain) {
			return t, implied, true
		}
	}
	return Test{}, Claim{}, false
}

func (md *builder) chain(n *ast.Node, first Test) ([]arm, *ast.Node) {
	arms := []arm{}
	for {
		statement := n.AsIfStatement()
		_, implied, ok := md.matching(statement.Expression, &first)
		if !ok {
			return arms, n
		}
		arms = append(arms, arm{claim: implied, condition: statement.Expression, body: []*ast.Node{statement.ThenStatement}})
		if statement.ElseStatement == nil || statement.ElseStatement.Kind != ast.KindIfStatement {
			return arms, statement.ElseStatement
		}
		n = statement.ElseStatement
	}
}

func (md *builder) statements(list []*ast.Node) {
	for i := 0; i < len(list); i++ {
		if list[i].Kind != ast.KindIfStatement {
			continue
		}
		first, _, ok := md.matching(list[i].AsIfStatement().Expression, nil)
		if !ok {
			continue
		}
		arms, rest := md.chain(list[i], first)
		end := i
		if rest == nil {
			for end+1 < len(list) && list[end+1].Kind == ast.KindIfStatement {
				more, tail := md.chain(list[end+1], first)
				if len(more) == 0 || tail != nil {
					break
				}
				arms = append(arms, more...)
				end++
			}
		}
		var alternative []*ast.Node
		var following *ast.Node
		if rest != nil {
			alternative = []*ast.Node{rest}
		} else {
			following = next(list[end])
		}
		md.record(list[i], first.Value.Node, first.Claim.Vocabulary, arms, alternative, following)
		i = end
	}
}

// clauseBody returns the statements that run for the first of clauses: its own and those of the clauses it falls
// through to, until statements that leave.
func (md *builder) clauseBody(clauses []*ast.Node) []*ast.Node {
	var body []*ast.Node
	for _, c := range clauses {
		statements := c.Statements()
		body = append(body, statements...)
		if len(statements) > 0 && md.m.listEnding(statements).end == nil {
			break
		}
	}
	return body
}

func (md *builder) switchStatement(n *ast.Node) {
	subjectNode := n.Expression()
	v := md.m.Vocabulary(subjectNode)
	if v == nil {
		return
	}
	arms := []arm{}
	var alternative []*ast.Node
	clauses := n.AsSwitchStatement().CaseBlock.AsCaseBlock().Clauses.Nodes
	for i, clause := range clauses {
		body := md.clauseBody(clauses[i:])
		if clause.Kind == ast.KindDefaultClause {
			if body == nil {
				body = []*ast.Node{}
			}
			alternative = body
			continue
		}
		if !Root(clause) {
			continue
		}
		if s := md.m.Classify(clause); s != nil {
			arms = append(arms, arm{claim: s.Tests[0].Claim, body: body})
		} else {
			arms = append(arms, arm{unread: true})
		}
	}
	var following *ast.Node
	if alternative == nil {
		following = next(n)
	}
	md.record(n, subjectNode, v, arms, alternative, following)
}

func (md *builder) ternary(n *ast.Node) {
	if p := n.Parent; p.Kind == ast.KindConditionalExpression && p.AsConditionalExpression().WhenFalse == n {
		return
	}
	first, _, ok := md.matching(n.AsConditionalExpression().Condition, nil)
	if !ok {
		return
	}
	arms := []arm{}
	current := n
	for current.Kind == ast.KindConditionalExpression {
		c := current.AsConditionalExpression()
		_, implied, ok := md.matching(c.Condition, &first)
		if !ok {
			break
		}
		arms = append(arms, arm{claim: implied, condition: c.Condition, body: []*ast.Node{c.WhenTrue}})
		current = c.WhenFalse
	}
	md.record(n, first.Value.Node, first.Claim.Vocabulary, arms, []*ast.Node{current}, nil)
}

func parts(n *ast.Node, operator ast.Kind) []*ast.Node {
	n = Unwrap(n)
	if n.Kind == ast.KindBinaryExpression && n.AsBinaryExpression().OperatorToken.Kind == operator {
		b := n.AsBinaryExpression()
		return append(parts(b.Left, operator), parts(b.Right, operator)...)
	}
	return []*ast.Node{n}
}

// declaresPredicate reports whether fn's written return type is a type predicate, `x is T`: the author states that it
// narrows.
func declaresPredicate(fn *ast.Node) bool {
	return fn.Type() != nil && fn.Type().Kind == ast.KindTypePredicate
}

// result returns fn's whole result: its expression body or its only statement's returned value, after any directives.
func result(fn *ast.Node) *ast.Node {
	body := fn.Body()
	if body == nil {
		return nil
	}
	if body.Kind != ast.KindBlock {
		return body
	}
	statements := work(body)
	if len(statements) == 1 && statements[0].Kind == ast.KindReturnStatement {
		return statements[0].Expression()
	}
	return nil
}

// work returns block's statements after its directive prologue, such as "use strict" or "worklet".
func work(block *ast.Node) []*ast.Node {
	statements := block.Statements()
	for len(statements) > 0 && ast.IsPrologueDirective(statements[0]) {
		statements = statements[1:]
	}
	return statements
}

func (md *builder) selector(f *Function) {
	fn := f.Node
	e := result(fn)
	if e == nil || len(fn.Parameters()) == 0 {
		return
	}
	if declaresPredicate(fn) {
		f.Kind = PredicateFunction
	} else {
		t := md.m.ctx.TypeChecker.GetTypeAtLocation(e)
		if t == nil || t.Flags()&checker.TypeFlagsBooleanLike == 0 {
			return
		}
		f.Kind = SelectorFunction
	}
	f.Answer, _ = md.answer(fn, e, 0)
	if len(f.Answer) == 0 {
		f.Kind = PlainFunction
	}
}

// answer returns the tests of literal values on fn's parameters that hold where e is true: joined with &&, merged
// across || on one value, negated with !, and read through calls to project functions, up to three calls deep. It
// reports whether the tests are exact, holding only where e is true; only an exact answer is negated.
func (md *builder) answer(fn, e *ast.Node, depth int) ([]Test, bool) {
	e = Unwrap(e)
	if t, ok := md.m.literalTest(e); ok {
		if _, ok := md.m.parameterOf(fn, t.Value.Node); !ok {
			return nil, false
		}
		return []Test{t}, true
	}
	if e.Kind == ast.KindBinaryExpression {
		switch e.AsBinaryExpression().OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken:
			out, exact := []Test{}, true
			for _, part := range parts(e, ast.KindAmpersandAmpersandToken) {
				found, partExact := md.answer(fn, part, depth)
				out, exact = append(out, found...), exact && partExact
			}
			return out, exact
		case ast.KindBarBarToken:
			var merged *Test
			exact := true
			for _, part := range parts(e, ast.KindBarBarToken) {
				found, partExact := md.answer(fn, part, depth)
				if len(found) != 1 || merged != nil && !found[0].Value.Same(merged.Value) {
					return nil, false
				}
				exact = exact && partExact
				if merged == nil {
					merged = &Test{Node: e, Value: found[0].Value, Claim: found[0].Claim}
				} else {
					merged.Claim = merged.Claim.Or(found[0].Claim)
				}
			}
			if merged == nil {
				return nil, false
			}
			return []Test{*merged}, exact
		}
	}
	if e.Kind == ast.KindPrefixUnaryExpression && e.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken {
		found, exact := md.answer(fn, e.AsPrefixUnaryExpression().Operand, depth)
		if len(found) != 1 || !exact {
			return nil, false
		}
		return []Test{{Node: e, Value: found[0].Value, Claim: found[0].Claim.Not()}}, true
	}
	if e.Kind != ast.KindCallExpression || depth >= 3 {
		return nil, false
	}
	out, exact, unmapped := []Test{}, true, -1
	for _, argument := range e.Arguments() {
		if _, ok := md.m.parameterOf(fn, argument); Unwrap(argument).Kind != ast.KindIdentifier || !ok {
			continue
		}
		b, ok := md.m.BoundaryOf(e, argument)
		if !ok {
			continue
		}
		body := result(b.Callee)
		if body == nil {
			continue
		}
		found, calleeExact := md.answer(b.Callee, body, depth+1)
		exact = exact && calleeExact
		if unmapped < 0 {
			unmapped = len(found)
		}
		for _, t := range found {
			if v, ok := md.m.Outward(b, t.Value); ok {
				out = append(out, Test{Node: e, Value: v, Claim: t.Claim})
				unmapped--
			}
		}
	}
	return out, exact && unmapped == 0
}

func (md *builder) entryNarrowings(fn *ast.Node) []EntryNarrowing {
	if d, ok := md.analyzed[fn]; ok {
		return d
	}
	md.analyzed[fn] = nil
	domains := []EntryNarrowing{}
	for _, e := range md.m.Entries(fn) {
		if e.Test.Claim.Presence() {
			continue
		}
		domains = append(domains, EntryNarrowing{Test: Test{Node: e.Test.Node, Value: e.Test.Value, Claim: e.Claim}, Property: md.label(e.Test.Value.Node)})
	}
	if len(domains) == 0 {
		domains = md.passthrough(fn)
	}
	md.analyzed[fn] = domains
	return domains
}

func (md *builder) passthrough(fn *ast.Node) []EntryNarrowing {
	body := fn.Body()
	if body == nil {
		return nil
	}
	call, _ := entryExpression(body)
	if call = Unwrap(call); call == nil || call.Kind != ast.KindCallExpression {
		return nil
	}
	out := []EntryNarrowing{}
	for _, argument := range call.Arguments() {
		if _, ok := md.m.parameterOf(fn, argument); !ok {
			continue
		}
		b, ok := md.m.BoundaryOf(call, argument)
		if !ok {
			continue
		}
		for _, d := range md.entryNarrowings(b.Callee) {
			if v, ok := md.m.Outward(b, d.Value); ok {
				d.Value = v
				d.Node = call
				out = append(out, d)
			}
		}
	}
	return out
}

func (md *builder) member(n *ast.Node) (string, *ast.Node) {
	switch n.Kind {
	case ast.KindPropertyAssignment, ast.KindPropertyDeclaration:
		if v := n.Initializer(); v != nil && ast.IsFunctionLike(Unwrap(v)) {
			return Text(n.Name(), 40), Unwrap(v)
		}
	case ast.KindMethodDeclaration:
		return Text(n.Name(), 40), n
	case ast.KindShorthandPropertyAssignment:
		if s := checker.Checker_GetShorthandAssignmentValueSymbol(md.m.ctx.TypeChecker, n); s != nil && s.ValueDeclaration != nil {
			d := s.ValueDeclaration
			if ast.IsFunctionLike(d) {
				return Text(n.Name(), 40), d
			}
			if d.Kind == ast.KindVariableDeclaration {
				if v := d.Initializer(); v != nil && ast.IsFunctionLike(Unwrap(v)) {
					return Text(n.Name(), 40), Unwrap(v)
				}
			}
		}
	}
	return "", nil
}

func (md *builder) firstParameterType(fn *ast.Node) *checker.Type {
	parameters := fn.Parameters()
	if len(parameters) == 0 {
		return nil
	}
	return md.m.ctx.TypeChecker.GetTypeAtLocation(parameters[0].Name())
}

func (md *builder) siblings(container *ast.Node, members []*ast.Node) {
	type entry struct {
		name string
		fn   *ast.Node
	}
	entries := []entry{}
	for _, n := range members {
		if name, fn := md.member(n); fn != nil {
			entries = append(entries, entry{name, fn})
		}
	}
	byNode := map[*ast.Node]*Function{}
	for _, f := range md.Functions {
		byNode[f.Node] = f
	}
	for _, s := range entries {
		sf := byNode[s.fn]
		if sf == nil || (sf.Kind != SelectorFunction && sf.Kind != PredicateFunction) {
			continue
		}
		st := md.firstParameterType(s.fn)
		for _, h := range entries {
			hf := byNode[h.fn]
			if h.fn == s.fn || hf == nil || st == nil || md.firstParameterType(h.fn) != st {
				continue
			}
			g := Gate{Selector: sf, Handler: hf, Names: [2]string{s.name, h.name}, Container: container}
			md.Gates = append(md.Gates, g)
			for i := range hf.Narrowings {
				if d := &hf.Narrowings[i]; md.implies(g, *d) {
					d.Selector, d.Handler, d.Container = s.name, h.name, container
				}
			}
		}
	}
}

// implies reports whether the selector's answer across g already ensures what d lets through: a test of the same part
// of the first parameter accepting only values d accepts.
func (md *builder) implies(g Gate, d EntryNarrowing) bool {
	selector, handler := g.Selector.Node.Parameters()[0].Symbol(), g.Handler.Node.Parameters()[0].Symbol()
	if d.Value.Root != handler || d.Value.This != nil || d.Value.Base != nil {
		return false
	}
	return slices.ContainsFunc(g.Selector.Answer, func(t Test) bool {
		return t.Value.Root == selector && t.Value.This == nil && t.Value.Base == nil && slices.Equal(t.Value.Path, d.Value.Path) && t.Claim.Implies(d.Claim)
	})
}
func (md *builder) collectFunctions(n *ast.Node) bool {
	if ast.IsFunctionLike(n) && n.Body() != nil {
		name := ""
		if nm := n.Name(); nm != nil {
			name = Text(nm, 40)
		} else if p := n.Parent; p != nil && (p.Kind == ast.KindPropertyAssignment || p.Kind == ast.KindVariableDeclaration || p.Kind == ast.KindPropertyDeclaration) {
			name = Text(p.Name(), 40)
		}
		f := &Function{ID: len(md.Functions), Line: md.line(n), Name: name, Kind: PlainFunction, Node: n}
		md.ids[n] = f.ID
		md.Functions = append(md.Functions, f)
	}
	n.ForEachChild(md.collectFunctions)
	return false
}

func (md *builder) visit(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindSourceFile, ast.KindBlock, ast.KindModuleBlock, ast.KindCaseClause, ast.KindDefaultClause:
		md.statements(n.Statements())
	case ast.KindIfStatement:
		if statementList(n.Parent) == nil && !elseOf(n) {
			md.statements([]*ast.Node{n})
		}
	case ast.KindSwitchStatement:
		md.switchStatement(n)
	case ast.KindConditionalExpression:
		md.ternary(n)
	}
	n.ForEachChild(md.visit)
	return false
}

// elseOf reports whether n is the else branch of an if statement, read with that statement's chain.
func elseOf(n *ast.Node) bool {
	return n.Parent.Kind == ast.KindIfStatement && n.Parent.AsIfStatement().ElseStatement == n
}

func (md *builder) containers(n *ast.Node) bool {
	switch {
	case n.Kind == ast.KindObjectLiteralExpression:
		md.siblings(n, n.Properties())
	case ast.IsClassLike(n):
		md.siblings(n, n.Members())
	}
	n.ForEachChild(md.containers)
	return false
}
