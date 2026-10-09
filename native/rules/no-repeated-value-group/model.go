package no_repeated_value_group

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

// Leaf is a test of one subject against a vocabulary, reduced to the values it accepts.
type Leaf struct {
	Vocabulary *Vocabulary
	Accepted   []string
	Subject    *ast.Node
}

// Arity is the size of the smaller side of the partition a leaf makes: 1 for a test of a single value or of all values but one.
func (l *Leaf) Arity() int {
	return min(len(l.Accepted), len(l.Vocabulary.Values)-len(l.Accepted))
}

type expr struct {
	op     string
	kids   []*expr
	leaf   *Leaf
	opaque string
}

// Site is a question a root node asks about vocabularies.
type Site struct {
	Root   *ast.Node
	Pos    int
	End    int
	Leaves []*Leaf
	// Canon identifies the question by the values of its vocabularies, the values it accepts and the text of its other operands.
	Canon string
	// Group is the accepted values of a site testing one vocabulary against several values, or "".
	Group string
	// Named is the name a site gives its question: a function whose whole body it is, or a declaration it initializes.
	Named string
	// Outcome is the normalized text of what the question leads to, or "" when it leads to nothing or to a trivial value.
	Outcome string
}

// Classifier reduces roots to sites with one invocation's checker.
type Classifier struct {
	checker *checker.Checker
	sites   map[*ast.Node]*Site
	vocabs  map[*checker.Type]*Vocabulary
}

func NewClassifier(c *checker.Checker) *Classifier {
	return &Classifier{checker: c, sites: map[*ast.Node]*Site{}, vocabs: map[*checker.Type]*Vocabulary{}}
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
func (v *Vocabulary) Display(keys []string) string {
	texts := []string{}
	for _, key := range keys {
		texts = append(texts, v.shown[key])
	}
	return strings.Join(texts, " | ")
}

// declared returns the declared type of an identifier, call, or property read; a property is read from its object's declared type, so narrowing the object does not narrow it.
func (c *Classifier) declared(n *ast.Node) *checker.Type {
	switch n = Unwrap(n); n.Kind {
	case ast.KindPropertyAccessExpression:
		if n.Name().Kind == ast.KindIdentifier {
			if object := c.declared(n.Expression()); object != nil {
				object = c.checker.GetNonNullableType(object)
				if object.Flags()&checker.TypeFlagsInstantiable != 0 {
					if constraint := checker.Checker_getBaseConstraintOfType(c.checker, object); constraint != nil {
						object = constraint
					}
				}
				if property := c.checker.GetPropertyOfType(object, n.Name().Text()); property != nil {
					return c.checker.GetTypeOfSymbol(property)
				}
			}
		}
		if symbol := c.checker.GetSymbolAtLocation(n.Name()); symbol != nil && symbol.Flags&(ast.SymbolFlagsProperty|ast.SymbolFlagsGetAccessor) != 0 {
			return c.checker.GetTypeOfSymbol(symbol)
		}
	case ast.KindIdentifier:
		if symbol := c.checker.GetSymbolAtLocation(n); symbol != nil && symbol.Flags&(ast.SymbolFlagsVariable|ast.SymbolFlagsProperty) != 0 {
			return c.checker.GetTypeOfSymbol(symbol)
		}
	case ast.KindCallExpression:
		if signature := c.checker.GetResolvedSignature(n); signature != nil {
			return c.checker.GetReturnTypeOfSignature(signature)
		}
	}
	return nil
}

func (c *Classifier) vocabularyOf(t *checker.Type) *Vocabulary {
	if t == nil {
		return nil
	}
	t = c.checker.GetNonNullableType(t)
	if t.Flags()&checker.TypeFlagsUnion == 0 && t.Flags()&checker.TypeFlagsInstantiable != 0 {
		if constraint := checker.Checker_getBaseConstraintOfType(c.checker, t); constraint != nil {
			t = c.checker.GetNonNullableType(constraint)
		}
	}
	if t.Flags()&checker.TypeFlagsUnion == 0 {
		return nil
	}
	if v, ok := c.vocabs[t]; ok {
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
			c.vocabs[t] = nil
			return nil
		}
		values = append(values, key)
		shown[key] = c.checker.TypeToString(member)
	}
	slices.Sort(values)
	v := &Vocabulary{Values: slices.Compact(values), Type: t, shown: shown}
	if len(v.Values) < 2 {
		v = nil
	}
	c.vocabs[t] = v
	return v
}

// Vocabulary returns the vocabulary of n's declared type, not its flow-narrowed type.
func (c *Classifier) Vocabulary(n *ast.Node) *Vocabulary { return c.vocabularyOf(c.declared(n)) }

func (c *Classifier) unit(n *ast.Node) (string, bool) {
	switch n = Unwrap(n); n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindPrefixUnaryExpression:
		if t := c.checker.GetTypeAtLocation(n); t != nil && literal(t) {
			return valueKey(t), true
		}
	case ast.KindPropertyAccessExpression, ast.KindIdentifier:
		name := n
		if n.Kind == ast.KindPropertyAccessExpression {
			name = n.Name()
		}
		symbol := c.checker.GetSymbolAtLocation(name)
		if symbol != nil && symbol.Flags&ast.SymbolFlagsAlias != 0 {
			symbol = c.checker.GetAliasedSymbol(symbol)
		}
		if symbol != nil && symbol.Flags&(ast.SymbolFlagsEnumMember|ast.SymbolFlagsBlockScopedVariable) != 0 {
			if t := c.checker.GetTypeOfSymbol(symbol); t != nil && literal(t) {
				return valueKey(t), true
			}
		}
	}
	return "", false
}

func (c *Classifier) arrayElement(t *checker.Type) *checker.Type {
	if t == nil {
		return nil
	}
	if t.Flags()&checker.TypeFlagsInstantiable != 0 {
		if constraint := checker.Checker_getBaseConstraintOfType(c.checker, t); constraint != nil {
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
			if arguments := c.checker.GetTypeArguments(candidate); len(arguments) > 0 {
				return arguments[0]
			}
		case checker.Checker_isArrayType(c.checker, candidate):
			return c.checker.GetElementTypeOfArrayType(candidate)
		}
	}
	return nil
}

func (c *Classifier) elements(receiver *ast.Node) ([]string, bool) {
	receiver = Unwrap(receiver)
	if receiver.Kind == ast.KindArrayLiteralExpression {
		values := []string{}
		for _, element := range receiver.Elements() {
			value, ok := c.unit(element)
			if !ok {
				return nil, false
			}
			values = append(values, value)
		}
		return values, len(values) > 0
	}
	t := c.checker.GetTypeAtLocation(receiver)
	members := []*checker.Type{}
	if t != nil && checker.IsTupleType(t) {
		members = c.checker.GetTypeArguments(t)
	} else if element := c.arrayElement(t); element != nil {
		members = []*checker.Type{element}
	}
	values := []string{}
	for _, member := range members {
		parts := []*checker.Type{member}
		if member.Flags()&checker.TypeFlagsUnion != 0 {
			parts = member.Types()
		}
		for _, part := range parts {
			if !literal(part) {
				return nil, false
			}
			values = append(values, valueKey(part))
		}
	}
	return values, len(values) > 0
}

func accept(v *Vocabulary, values []string, negated bool) ([]string, bool) {
	for _, value := range values {
		if !slices.Contains(v.Values, value) {
			return nil, false
		}
	}
	accepted := []string{}
	for _, value := range v.Values {
		if slices.Contains(values, value) != negated {
			accepted = append(accepted, value)
		}
	}
	return accepted, true
}

func (c *Classifier) leaf(subject *ast.Node, values []string, negated bool) *expr {
	v := c.Vocabulary(subject)
	if v == nil {
		return nil
	}
	accepted, ok := accept(v, values, negated)
	if !ok {
		return nil
	}
	return &expr{leaf: &Leaf{Vocabulary: v, Accepted: accepted, Subject: Unwrap(subject)}}
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

func (c *Classifier) observe(n *ast.Node) *expr {
	switch n.Kind {
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		if ok, negated := equality(binary.OperatorToken.Kind); ok {
			for _, pair := range [][2]*ast.Node{{binary.Left, binary.Right}, {binary.Right, binary.Left}} {
				if Unwrap(pair[0]).Kind == ast.KindTypeOfExpression {
					continue
				}
				if value, ok := c.unit(pair[1]); ok {
					if e := c.leaf(pair[0], []string{value}, negated); e != nil {
						return e
					}
				}
			}
		}
	case ast.KindCallExpression:
		if receiver, ok := includes(n); ok {
			if values, ok := c.elements(receiver); ok {
				return c.leaf(n.Arguments()[0], values, false)
			}
		}
	}
	return nil
}

func (c *Classifier) build(n *ast.Node, text string) *expr {
	n = Unwrap(n)
	if e := c.observe(n); e != nil {
		return e
	}
	switch n.Kind {
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindExclamationToken {
			return negate(c.build(unary.Operand, text))
		}
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		switch binary.OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken:
			return &expr{op: "&", kids: []*expr{c.build(binary.Left, text), c.build(binary.Right, text)}}
		case ast.KindBarBarToken:
			return &expr{op: "|", kids: []*expr{c.build(binary.Left, text), c.build(binary.Right, text)}}
		}
	}
	return &expr{opaque: Squash(text[scanner.SkipTrivia(text, n.Pos()):n.End()])}
}

func negate(e *expr) *expr {
	switch {
	case e.leaf != nil:
		accepted, _ := accept(e.leaf.Vocabulary, e.leaf.Accepted, true)
		copied := *e.leaf
		copied.Accepted = accepted
		return &expr{leaf: &copied}
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
		return &expr{opaque: text}
	}
	return &expr{opaque: "!" + e.opaque}
}

// SameSubject reports whether two leaves test the same vocabulary on the same expression text.
func SameSubject(a, b *Leaf) bool {
	return slices.Equal(a.Vocabulary.Values, b.Vocabulary.Values) && Squash(source(a.Subject)) == Squash(source(b.Subject))
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
		if k.leaf != nil {
			for _, m := range merged {
				if m.leaf != nil && SameSubject(m.leaf, k.leaf) {
					existing = m
					break
				}
			}
		}
		if existing == nil {
			merged = append(merged, k)
			continue
		}
		combined := *existing.leaf
		combined.Accepted = []string{}
		for _, value := range k.leaf.Vocabulary.Values {
			in, other := slices.Contains(existing.leaf.Accepted, value), slices.Contains(k.leaf.Accepted, value)
			if (e.op == "|" && (in || other)) || (e.op == "&" && in && other) {
				combined.Accepted = append(combined.Accepted, value)
			}
		}
		existing.leaf = &combined
	}
	if len(merged) == 1 {
		return merged[0]
	}
	return &expr{op: e.op, kids: merged}
}

func canon(e *expr) string {
	switch {
	case e.leaf != nil:
		return "{" + strings.Join(e.leaf.Vocabulary.Values, ",") + "}∈{" + strings.Join(e.leaf.Accepted, ",") + "}"
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

func collect(e *expr, leaves *[]*Leaf) {
	switch {
	case e.leaf != nil:
		*leaves = append(*leaves, e.leaf)
	case e.op != "":
		for _, k := range e.kids {
			collect(k, leaves)
		}
	}
}

// Leaves returns the vocabulary tests of a condition after merging tests of the same subject.
func (c *Classifier) Leaves(condition *ast.Node) []*Leaf {
	text := ast.GetSourceFileOfNode(condition).Text()
	leaves := []*Leaf{}
	collect(normalize(c.build(condition, text)), &leaves)
	return leaves
}

func (c *Classifier) parameterType(call *ast.Node, argument *ast.Node) *checker.Type {
	index := slices.Index(call.Arguments(), argument)
	signature := c.checker.GetResolvedSignature(call)
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
	return c.checker.GetTypeOfSymbol(parameters[index])
}

func (c *Classifier) list(n *ast.Node) *Site {
	values := []string{}
	for _, element := range n.Elements() {
		value, ok := c.unit(element)
		if !ok {
			return nil
		}
		values = append(values, value)
	}
	outer, parent := listContext(n)
	contextual := checker.Checker_getContextualType(c.checker, outer, checker.ContextFlagsNone)
	if parent.Kind == ast.KindCallExpression || parent.Kind == ast.KindNewExpression {
		if declared := c.parameterType(parent, outer); declared != nil {
			contextual = declared
		}
	}
	v := c.vocabularyOf(c.arrayElement(contextual))
	if v == nil {
		return nil
	}
	accepted, ok := accept(v, values, false)
	if !ok {
		return nil
	}
	l := &Leaf{Vocabulary: v, Accepted: accepted, Subject: n}
	s := &Site{Root: n, Pos: scanner.SkipTrivia(ast.GetSourceFileOfNode(n).Text(), n.Pos()), End: n.End(), Leaves: []*Leaf{l}, Canon: canon(&expr{leaf: l})}
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

func (c *Classifier) caseGroup(first *ast.Node) *Site {
	switchStatement := first.Parent.Parent
	v := c.Vocabulary(switchStatement.Expression())
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
		value, ok := c.unit(clause.Expression())
		if !ok {
			return nil
		}
		values = append(values, value)
		last = clause
		if len(clause.Statements()) > 0 {
			break
		}
	}
	accepted, ok := accept(v, values, false)
	if !ok {
		return nil
	}
	l := &Leaf{Vocabulary: v, Accepted: accepted, Subject: Unwrap(switchStatement.Expression())}
	text := ast.GetSourceFileOfNode(first).Text()
	return &Site{Root: first, Pos: scanner.SkipTrivia(text, first.Pos()), End: last.Expression().End(), Leaves: []*Leaf{l}, Canon: canon(&expr{leaf: l}), Outcome: Outcome(first)}
}

func (c *Classifier) question(n *ast.Node) *Site {
	text := ast.GetSourceFileOfNode(n).Text()
	e := normalize(c.build(n, text))
	leaves := []*Leaf{}
	collect(e, &leaves)
	if len(leaves) == 0 {
		return nil
	}
	return &Site{Root: n, Pos: scanner.SkipTrivia(text, n.Pos()), End: n.End(), Leaves: leaves, Canon: canon(e), Named: predicateName(n), Outcome: Outcome(n)}
}

// Classify returns the site a root asks, or nil when it tests no vocabulary.
func (c *Classifier) Classify(root *ast.Node) *Site {
	if s, ok := c.sites[root]; ok {
		return s
	}
	var s *Site
	switch root.Kind {
	case ast.KindArrayLiteralExpression:
		s = c.list(root)
	case ast.KindCaseClause:
		s = c.caseGroup(root)
	default:
		s = c.question(root)
	}
	if s != nil && len(s.Leaves) == 1 && s.Leaves[0].Arity() >= 2 {
		s.Group = strings.Join(s.Leaves[0].Accepted, "\x00")
	}
	c.sites[root] = s
	return s
}
