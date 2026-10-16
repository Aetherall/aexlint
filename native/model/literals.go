package model

import (
	"slices"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/jsnum"
	"github.com/microsoft/typescript-go/shim/scanner"
)

// Vocabulary is a union of at least two string or number literals, identified by its sorted value keys.
type Vocabulary struct {
	Values []string
	Type   *checker.Type
	shown  map[string]string
}

// expr is a condition read as tests of literal values joined by && and ||, with negations pushed to its operands. Any
// other operand is opaque: its node, whether it is negated, and its text.
type expr struct {
	op      string
	kids    []*expr
	test    *Test
	opaque  string
	node    *ast.Node
	negated bool
}

// Site is a question a root node asks about vocabularies: Tests are its tests of literal values, merged by subject.
type Site struct {
	Root  *ast.Node
	Pos   int
	End   int
	Tests []Test
	// Canon identifies the question by the values of its vocabularies, the values it accepts and the text of its other operands.
	Canon string
	// Group is the accepted values of a site testing one vocabulary against several values, or "".
	Group string
	// Named is the name a site gives its question: a function whose whole body it is, or a declaration it initializes.
	Named string
	// Outcome is the normalized text of what the question leads to, or "" when it leads to nothing or to a trivial value.
	Outcome string
}

func Unwrap(n *ast.Node) *ast.Node {
	for n != nil {
		switch n.Kind {
		case ast.KindParenthesizedExpression, ast.KindAsExpression, ast.KindTypeAssertionExpression, ast.KindNonNullExpression, ast.KindSatisfiesExpression:
			n = n.Expression()
		default:
			return n
		}
	}
	return n
}

func literal(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsStringLiteral|checker.TypeFlagsNumberLiteral) != 0
}

func NumberKey(value float64) string { return "n:" + strconv.FormatFloat(value, 'g', -1, 64) }

func valueKey(t *checker.Type) string {
	switch value := t.AsLiteralType().Value().(type) {
	case string:
		return "s:" + value
	case jsnum.Number:
		return NumberKey(float64(value))
	}
	return ""
}

// Display renders value keys as the vocabulary's members print, such as "sms" or Level.High.
func (v *Vocabulary) Display(keys []string) string { return strings.Join(v.Names(keys), " | ") }

// Names returns how the given value keys print, in the vocabulary's order.
func (v *Vocabulary) Names(keys []string) []string {
	names := []string{}
	for _, key := range v.Values {
		if slices.Contains(keys, key) {
			names = append(names, v.shown[key])
		}
	}
	return names
}

// Declared returns the declared type of an identifier, call, or property read; a property is read from its object's declared type, so narrowing the object does not narrow it.
func (m *Analyzer) Declared(n *ast.Node) *checker.Type {
	switch n = Unwrap(n); n.Kind {
	case ast.KindPropertyAccessExpression:
		if n.Name().Kind == ast.KindIdentifier {
			if object := m.Declared(n.Expression()); object != nil {
				object = m.ctx.TypeChecker.GetNonNullableType(object)
				if object.Flags()&checker.TypeFlagsInstantiable != 0 {
					if constraint := checker.Checker_getBaseConstraintOfType(m.ctx.TypeChecker, object); constraint != nil {
						object = constraint
					}
				}
				if property := m.ctx.TypeChecker.GetPropertyOfType(object, n.Name().Text()); property != nil {
					return m.ctx.TypeChecker.GetTypeOfSymbol(property)
				}
			}
		}
		if symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n.Name()); symbol != nil && symbol.Flags&(ast.SymbolFlagsProperty|ast.SymbolFlagsGetAccessor) != 0 {
			return m.ctx.TypeChecker.GetTypeOfSymbol(symbol)
		}
	case ast.KindIdentifier:
		if symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n); symbol != nil && symbol.Flags&(ast.SymbolFlagsVariable|ast.SymbolFlagsProperty) != 0 {
			return m.ctx.TypeChecker.GetTypeOfSymbol(symbol)
		}
	case ast.KindCallExpression:
		if signature := m.ctx.TypeChecker.GetResolvedSignature(n); signature != nil {
			return m.ctx.TypeChecker.GetReturnTypeOfSignature(signature)
		}
	}
	return nil
}

func (m *Analyzer) vocabularyOf(t *checker.Type) *Vocabulary {
	if t == nil {
		return nil
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	if t.Flags()&checker.TypeFlagsUnion == 0 && t.Flags()&checker.TypeFlagsInstantiable != 0 {
		if constraint := checker.Checker_getBaseConstraintOfType(m.ctx.TypeChecker, t); constraint != nil {
			t = m.ctx.TypeChecker.GetNonNullableType(constraint)
		}
	}
	if t.Flags()&checker.TypeFlagsUnion == 0 {
		return nil
	}
	if v, ok := m.vocabs[t]; ok {
		return v
	}
	values := []string{}
	shown := map[string]string{}
	for _, member := range t.Types() {
		key := ""
		if literal(member) {
			key = valueKey(member)
		}
		if key == "" {
			m.vocabs[t] = nil
			return nil
		}
		values = append(values, key)
		shown[key] = m.ctx.TypeChecker.TypeToString(member)
	}
	slices.Sort(values)
	v := &Vocabulary{Values: slices.Compact(values), Type: t, shown: shown}
	if len(v.Values) < 2 {
		v = nil
	}
	m.vocabs[t] = v
	return v
}

// Vocabulary returns the vocabulary of n's declared type, not its flow-narrowed type.
func (m *Analyzer) Vocabulary(n *ast.Node) *Vocabulary { return m.vocabularyOf(m.Declared(n)) }

func (m *Analyzer) Unit(n *ast.Node) (string, bool) {
	switch n = Unwrap(n); n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindPrefixUnaryExpression:
		if t := m.ctx.TypeChecker.GetTypeAtLocation(n); t != nil && literal(t) {
			return valueKey(t), true
		}
	case ast.KindPropertyAccessExpression, ast.KindIdentifier:
		name := n
		if n.Kind == ast.KindPropertyAccessExpression {
			name = n.Name()
		}
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(name)
		if symbol != nil && symbol.Flags&ast.SymbolFlagsAlias != 0 {
			symbol = m.ctx.TypeChecker.GetAliasedSymbol(symbol)
		}
		if symbol != nil && symbol.Flags&(ast.SymbolFlagsEnumMember|ast.SymbolFlagsBlockScopedVariable|ast.SymbolFlagsProperty) != 0 {
			if t := m.ctx.TypeChecker.GetTypeOfSymbol(symbol); t != nil && literal(t) {
				return valueKey(t), true
			}
		}
	}
	return "", false
}

func (m *Analyzer) arrayElement(t *checker.Type) *checker.Type {
	if t == nil {
		return nil
	}
	if t.Flags()&checker.TypeFlagsInstantiable != 0 {
		if constraint := checker.Checker_getBaseConstraintOfType(m.ctx.TypeChecker, t); constraint != nil {
			t = constraint
		}
	}
	candidates := []*checker.Type{t}
	if t.Flags()&checker.TypeFlagsUnion != 0 {
		candidates = t.Types()
	}
	for _, candidate := range candidates {
		switch {
		case checker.IsTupleType(candidate):
			if arguments := m.ctx.TypeChecker.GetTypeArguments(candidate); len(arguments) > 0 {
				return arguments[0]
			}
		case checker.Checker_isArrayType(m.ctx.TypeChecker, candidate):
			return m.ctx.TypeChecker.GetElementTypeOfArrayType(candidate)
		}
	}
	return nil
}

// elements returns the values an array holds, when they are known exactly: an array literal of literal values, or a
// tuple of literal types such as an array written as const. An array type only says which values it may hold.
func (m *Analyzer) elements(receiver *ast.Node) ([]string, bool) {
	receiver = Unwrap(receiver)
	if receiver.Kind == ast.KindArrayLiteralExpression {
		values := []string{}
		for _, element := range receiver.Elements() {
			value, ok := m.Unit(element)
			if !ok {
				return nil, false
			}
			values = append(values, value)
		}
		return values, len(values) > 0
	}
	t := m.ctx.TypeChecker.GetTypeAtLocation(receiver)
	if t == nil || !checker.IsTupleType(t) {
		return nil, false
	}
	values := []string{}
	for _, member := range m.ctx.TypeChecker.GetTypeArguments(t) {
		if !literal(member) {
			return nil, false
		}
		values = append(values, valueKey(member))
	}
	return values, len(values) > 0
}

// accepting is the claim that a value of v is one of values, or false when v does not hold them all.
func accepting(v *Vocabulary, values []string) (Claim, bool) {
	if slices.ContainsFunc(values, func(value string) bool { return !slices.Contains(v.Values, value) }) {
		return Claim{}, false
	}
	return Claim{Domain: v.Values, Accepted: within(v.Values, values), Vocabulary: v}, true
}

func (m *Analyzer) valueTest(n, subject *ast.Node, values []string, negated bool) *expr {
	v := m.Vocabulary(subject)
	if v == nil {
		return nil
	}
	claim, ok := accepting(v, values)
	if !ok {
		return nil
	}
	if negated {
		claim = claim.Not()
	}
	return &expr{test: &Test{Node: n, Value: m.ValueOf(subject), Claim: claim}}
}

func equality(kind ast.Kind) (bool, bool) {
	switch kind {
	case ast.KindEqualsEqualsEqualsToken, ast.KindEqualsEqualsToken:
		return true, false
	case ast.KindExclamationEqualsEqualsToken, ast.KindExclamationEqualsToken:
		return true, true
	}
	return false, false
}

func includes(n *ast.Node) (*ast.Node, bool) {
	if n.Kind != ast.KindCallExpression || len(n.Arguments()) != 1 {
		return nil, false
	}
	callee := Unwrap(n.AsCallExpression().Expression)
	if callee.Kind != ast.KindPropertyAccessExpression || callee.Name().Kind != ast.KindIdentifier || callee.Name().Text() != "includes" {
		return nil, false
	}
	return callee.Expression(), true
}

func (m *Analyzer) observe(n *ast.Node) *expr {
	switch n.Kind {
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		if ok, negated := equality(binary.OperatorToken.Kind); ok {
			for _, pair := range [][2]*ast.Node{{binary.Left, binary.Right}, {binary.Right, binary.Left}} {
				if Unwrap(pair[0]).Kind == ast.KindTypeOfExpression {
					continue
				}
				if value, ok := m.Unit(pair[1]); ok {
					if e := m.valueTest(n, pair[0], []string{value}, negated); e != nil {
						return e
					}
				}
			}
		}
	case ast.KindCallExpression:
		if receiver, ok := includes(n); ok {
			if values, ok := m.elements(receiver); ok {
				return m.valueTest(n, n.Arguments()[0], values, false)
			}
		}
	}
	return nil
}

func (m *Analyzer) build(n *ast.Node, text string) *expr {
	n = Unwrap(n)
	if e := m.observe(n); e != nil {
		return e
	}
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			return negate(m.build(unary.Operand, text))
		}
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		switch binary.OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken:
			return &expr{op: "&", kids: []*expr{m.build(binary.Left, text), m.build(binary.Right, text)}}
		case ast.KindBarBarToken:
			return &expr{op: "|", kids: []*expr{m.build(binary.Left, text), m.build(binary.Right, text)}}
		}
	}
	return &expr{opaque: Squash(text[scanner.SkipTrivia(text, n.Pos()):n.End()]), node: n}
}

func negate(e *expr) *expr {
	switch {
	case e.test != nil:
		copied := *e.test
		copied.Claim = copied.Claim.Not()
		return &expr{test: &copied}
	case e.op != "":
		op := "|"
		if e.op == "|" {
			op = "&"
		}
		kids := []*expr{}
		for _, k := range e.kids {
			kids = append(kids, negate(k))
		}
		return &expr{op: op, kids: kids}
	}
	if text, ok := strings.CutPrefix(e.opaque, "!"); ok {
		return &expr{opaque: text, node: e.node, negated: !e.negated}
	}
	return &expr{opaque: "!" + e.opaque, node: e.node, negated: !e.negated}
}

// SameSubject reports whether two tests of literal values test the same vocabulary on the same value.
func SameSubject(a, b Test) bool {
	return slices.Equal(a.Claim.Vocabulary.Values, b.Claim.Vocabulary.Values) && a.Value.Same(b.Value)
}

func normalize(e *expr) *expr {
	if e.op == "" {
		return e
	}
	flat := []*expr{}
	for _, k := range e.kids {
		k = normalize(k)
		if k.op == e.op {
			flat = append(flat, k.kids...)
		} else {
			flat = append(flat, k)
		}
	}
	merged := []*expr{}
	for _, k := range flat {
		var existing *expr
		if k.test != nil {
			for _, m := range merged {
				if m.test != nil && SameSubject(*m.test, *k.test) {
					existing = m
					break
				}
			}
		}
		if existing == nil {
			merged = append(merged, k)
			continue
		}
		combined := *existing.test
		if e.op == "|" {
			combined.Claim = combined.Claim.Or(k.test.Claim)
		} else {
			combined.Claim = combined.Claim.And(k.test.Claim)
		}
		existing.test = &combined
	}
	if len(merged) == 1 {
		return merged[0]
	}
	return &expr{op: e.op, kids: merged}
}

func canon(e *expr) string {
	switch {
	case e.test != nil:
		return "{" + strings.Join(e.test.Claim.Vocabulary.Values, ",") + "}∈{" + strings.Join(e.test.Claim.Accepted, ",") + "}"
	case e.op != "":
		parts := []string{}
		for _, k := range e.kids {
			parts = append(parts, canon(k))
		}
		slices.Sort(parts)
		return "(" + strings.Join(parts, " "+e.op+" ") + ")"
	}
	return "?" + e.opaque
}

func collect(e *expr, tests *[]Test) {
	switch {
	case e.test != nil:
		*tests = append(*tests, *e.test)
	case e.op != "":
		for _, k := range e.kids {
			collect(k, tests)
		}
	}
}

// Tests returns the tests of literal values a condition makes, after merging tests of the same subject.
func (m *Analyzer) Tests(condition *ast.Node) []Test {
	tests := []Test{}
	collect(normalize(m.build(condition, ast.GetSourceFileOfNode(condition).Text())), &tests)
	return tests
}

// Implied returns what condition establishes about the subject of t, one of its tests: t's claim where condition is t,
// combined through && and || with the other operands. An operand holding only where an object the subject is read from
// is missing leaves the subject without a value; other operands establish nothing about it.
func (m *Analyzer) Implied(condition *ast.Node, t Test) Claim {
	all := Claim{Domain: t.Claim.Domain, Accepted: t.Claim.Domain, Vocabulary: t.Claim.Vocabulary}
	none := Claim{Domain: all.Domain, Accepted: []string{}, Vocabulary: all.Vocabulary}
	var implied func(*expr) Claim
	implied = func(e *expr) Claim {
		switch {
		case e.test != nil && SameSubject(*e.test, t):
			return e.test.Claim
		case e.opaque != "" && m.absent(e, t.Value):
			return none
		case e.op == "&":
			c := all
			for _, k := range e.kids {
				c = c.And(implied(k))
			}
			return c
		case e.op == "|":
			c := none
			for _, k := range e.kids {
				c = c.Or(implied(k))
			}
			return c
		}
		return all
	}
	return implied(normalize(m.build(condition, ast.GetSourceFileOfNode(condition).Text())))
}

// absent reports whether an opaque operand holds only where subject, or an object it is read from, is missing.
func (m *Analyzer) absent(e *expr, subject Value) bool {
	p, ok := PresenceTest(e.node)
	if !ok || p.Missing == e.negated {
		return false
	}
	return subject.Within(m.ValueOf(p.Subject))
}

// literalTest reads a whole condition as one test of literal values, or returns false when it has other operands.
func (m *Analyzer) literalTest(condition *ast.Node) (Test, bool) {
	e := normalize(m.build(condition, ast.GetSourceFileOfNode(condition).Text()))
	if e.test == nil {
		return Test{}, false
	}
	t := *e.test
	t.Node, t.Named = condition, m.named(condition, t.Claim.Vocabulary)
	return t, true
}

func (m *Analyzer) parameterType(call *ast.Node, argument *ast.Node) *checker.Type {
	index := slices.Index(call.Arguments(), argument)
	signature := m.ctx.TypeChecker.GetResolvedSignature(call)
	if index < 0 || signature == nil {
		return nil
	}
	if signature.Target() != nil {
		signature = signature.Target()
	}
	parameters := signature.Parameters()
	if index >= len(parameters) {
		return nil
	}
	return m.ctx.TypeChecker.GetTypeOfSymbol(parameters[index])
}

func (m *Analyzer) list(n *ast.Node) *Site {
	values := []string{}
	for _, element := range n.Elements() {
		value, ok := m.Unit(element)
		if !ok {
			return nil
		}
		values = append(values, value)
	}
	outer, parent := listContext(n)
	contextual := checker.Checker_getContextualType(m.ctx.TypeChecker, outer, checker.ContextFlagsNone)
	if parent.Kind == ast.KindCallExpression || parent.Kind == ast.KindNewExpression {
		if declared := m.parameterType(parent, outer); declared != nil {
			contextual = declared
		}
	}
	v := m.vocabularyOf(m.arrayElement(contextual))
	if v == nil {
		return nil
	}
	claim, ok := accepting(v, values)
	if !ok {
		return nil
	}
	t := &Test{Node: n, Value: m.ValueOf(n), Claim: claim}
	s := &Site{Root: n, Pos: scanner.SkipTrivia(ast.GetSourceFileOfNode(n).Text(), n.Pos()), End: n.End(), Tests: []Test{*t}, Canon: canon(&expr{test: t})}
	switch parent.Kind {
	case ast.KindVariableDeclaration, ast.KindPropertyDeclaration, ast.KindPropertyAssignment:
		if parent.Name() != nil && parent.Name().Kind == ast.KindIdentifier && (parent.Kind != ast.KindPropertyAssignment || named(parent.Parent.Parent)) {
			s.Named = parent.Name().Text()
		}
	}
	return s
}

func named(n *ast.Node) bool {
	return n != nil && (n.Kind == ast.KindVariableDeclaration || n.Kind == ast.KindPropertyDeclaration) && n.Name() != nil && n.Name().Kind == ast.KindIdentifier
}

func (m *Analyzer) caseGroup(first *ast.Node) *Site {
	switchStatement := first.Parent.Parent
	v := m.Vocabulary(switchStatement.Expression())
	if v == nil {
		return nil
	}
	values := []string{}
	clauses := switchStatement.AsSwitchStatement().CaseBlock.AsCaseBlock().Clauses.Nodes
	last := first
	for _, clause := range clauses[slices.Index(clauses, first):] {
		if clause.Kind == ast.KindDefaultClause {
			return nil
		}
		value, ok := m.Unit(clause.Expression())
		if !ok {
			return nil
		}
		values = append(values, value)
		last = clause
		if len(clause.Statements()) > 0 {
			break
		}
	}
	claim, ok := accepting(v, values)
	if !ok {
		return nil
	}
	t := &Test{Node: first, Value: m.ValueOf(switchStatement.Expression()), Claim: claim}
	text := ast.GetSourceFileOfNode(first).Text()
	return &Site{Root: first, Pos: scanner.SkipTrivia(text, first.Pos()), End: last.Expression().End(), Tests: []Test{*t}, Canon: canon(&expr{test: t}), Outcome: OutcomeText(first)}
}

// Exhaustive reports whether a switch on a vocabulary has no default clause and a case for every value.
func (m *Analyzer) Exhaustive(n *ast.Node) bool {
	v := m.Vocabulary(n.Expression())
	if v == nil {
		return false
	}
	covered := map[string]bool{}
	for _, clause := range n.AsSwitchStatement().CaseBlock.AsCaseBlock().Clauses.Nodes {
		if clause.Kind == ast.KindDefaultClause {
			return false
		}
		if key, ok := m.Unit(clause.Expression()); ok {
			covered[key] = true
		}
	}
	return !slices.ContainsFunc(v.Values, func(value string) bool { return !covered[value] })
}

func (m *Analyzer) question(n *ast.Node) *Site {
	text := ast.GetSourceFileOfNode(n).Text()
	e := normalize(m.build(n, text))
	tests := []Test{}
	collect(e, &tests)
	if len(tests) == 0 {
		return nil
	}
	return &Site{Root: n, Pos: scanner.SkipTrivia(text, n.Pos()), End: n.End(), Tests: tests, Canon: canon(e), Named: m.predicateName(n), Outcome: OutcomeText(n)}
}

// Classify returns the site a root asks, or nil when it tests no vocabulary.
func (m *Analyzer) Classify(root *ast.Node) *Site {
	if s, ok := m.sites[root]; ok {
		return s
	}
	var s *Site
	switch root.Kind {
	case ast.KindArrayLiteralExpression:
		s = m.list(root)
	case ast.KindCaseClause:
		s = m.caseGroup(root)
	default:
		s = m.question(root)
	}
	if s != nil && len(s.Tests) == 1 && s.Tests[0].Claim.Arity() >= 2 {
		s.Group = strings.Join(s.Tests[0].Claim.Accepted, "\x00")
	}
	m.sites[root] = s
	return s
}
