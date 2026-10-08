package max_interpretation_spread

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/jsnum"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const maxSafeInteger = 1<<53 - 1

const defaultMax = 4

func parseOptions(options any) int {
	if options == nil {
		return defaultMax
	}
	object, ok := options.(map[string]any)
	if !ok {
		panic(`aexlint/max-interpretation-spread accepts an optional options object: { "max"?: <positive integer> }`)
	}
	for key := range object {
		if key != "max" {
			panic(fmt.Sprintf("aexlint/max-interpretation-spread: unknown option %q; only \"max\" is accepted", key))
		}
	}
	raw, present := object["max"]
	if !present {
		return defaultMax
	}
	value, ok := raw.(float64)
	if !ok || value < 1 || value > maxSafeInteger || value != float64(int64(value)) {
		panic(`aexlint/max-interpretation-spread: "max" must be a positive safe integer`)
	}
	return int(value)
}

func unwrap(n *ast.Node) *ast.Node {
	for {
		switch n.Kind {
		case ast.KindParenthesizedExpression, ast.KindAsExpression, ast.KindTypeAssertionExpression, ast.KindNonNullExpression, ast.KindSatisfiesExpression:
			n = n.Expression()
		default:
			return n
		}
	}
}

func literal(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsStringLiteral|checker.TypeFlagsNumberLiteral) != 0
}

// Classifier recognizes subset selections on closed vocabularies with one invocation's checker.
type Classifier struct {
	checker *checker.Checker
}

func NewClassifier(c *checker.Checker) *Classifier { return &Classifier{checker: c} }

// declared returns the declared (not flow-narrowed) type of a property read, identifier, or call.
func (c *Classifier) declared(n *ast.Node) *checker.Type {
	switch n = unwrap(n); n.Kind {
	case ast.KindPropertyAccessExpression:
		if symbol := c.checker.GetSymbolAtLocation(n.Name()); symbol != nil && symbol.Flags&(ast.SymbolFlagsProperty|ast.SymbolFlagsGetAccessor|ast.SymbolFlagsEnumMember) != 0 {
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

// vocabulary returns a key for t's literal set without null and undefined, and the union, or "" when t is not a union of at least two literals.
func (c *Classifier) vocabulary(t *checker.Type) (string, *checker.Type) {
	if t == nil {
		return "", nil
	}
	t = c.checker.GetNonNullableType(t)
	if t.Flags()&checker.TypeFlagsUnion == 0 {
		if t.Flags()&checker.TypeFlagsInstantiable != 0 {
			if constraint := checker.Checker_getBaseConstraintOfType(c.checker, t); constraint != nil && constraint != t {
				return c.vocabulary(constraint)
			}
		}
		return "", nil
	}
	values := []string{}
	for _, member := range t.Types() {
		if !literal(member) {
			return "", nil
		}
		values = append(values, valueKey(member))
	}
	slices.Sort(values)
	return strings.Join(values, "\x00"), t
}

// valueKey identifies a literal by its value, so that fresh and regular literal types of the same value share a key.
func valueKey(t *checker.Type) string {
	prefix := ""
	if symbol := t.Symbol(); symbol != nil && symbol.Flags&ast.SymbolFlagsEnumMember != 0 && symbol.Parent != nil {
		prefix = "e:" + symbol.Parent.Name + "." + symbol.Name + "="
	}
	switch value := t.AsLiteralType().Value().(type) {
	case string:
		return prefix + "s:" + value
	case jsnum.Number:
		return prefix + NumberKey(float64(value))
	}
	return prefix + fmt.Sprint(t.AsLiteralType().Value())
}

func (c *Classifier) unit(n *ast.Node) *checker.Type {
	switch n = unwrap(n); n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindPrefixUnaryExpression:
		t := c.checker.GetTypeAtLocation(n)
		if t != nil && literal(t) {
			return t
		}
		return nil
	case ast.KindTypeOfExpression, ast.KindNullKeyword, ast.KindTrueKeyword, ast.KindFalseKeyword:
		return nil
	}
	if t := c.declared(n); t != nil && literal(t) {
		return t
	}
	return nil
}

// Site is a subset selection: Node holds a value of the vocabulary identified by Key.
type Site struct {
	Key   string
	Union *checker.Type
	Node  *ast.Node
}

// Classify returns the subset selection a branch makes on a closed vocabulary: an equality against a single literal, or a switch that is not exhaustive or has a default.
func (c *Classifier) Classify(n *ast.Node) (Site, bool) {
	if n.Kind == ast.KindSwitchStatement {
		key, union := c.vocabulary(c.declared(n.Expression()))
		if key == "" {
			return Site{}, false
		}
		covered, hasDefault := map[string]bool{}, false
		for _, clause := range n.AsSwitchStatement().CaseBlock.AsCaseBlock().Clauses.Nodes {
			if clause.Kind == ast.KindDefaultClause {
				hasDefault = true
			} else if t := c.unit(clause.Expression()); t != nil {
				covered[valueKey(t)] = true
			}
		}
		exhaustive := !hasDefault
		for _, member := range union.Types() {
			exhaustive = exhaustive && covered[valueKey(member)]
		}
		return Site{key, union, n.Expression()}, !exhaustive
	}
	binary := n.AsBinaryExpression()
	for _, pair := range [][2]*ast.Node{{binary.Left, binary.Right}, {binary.Right, binary.Left}} {
		if unwrap(pair[0]).Kind == ast.KindTypeOfExpression || c.unit(pair[1]) == nil {
			continue
		}
		if key, union := c.vocabulary(c.declared(pair[0])); key != "" {
			return Site{key, union, pair[0]}, true
		}
	}
	return Site{}, false
}

// Keys returns the index keys under which a branch comparing against a member of union is listed.
func Keys(union *checker.Type) []string {
	result := []string{}
	for _, member := range union.Types() {
		switch value := member.AsLiteralType().Value().(type) {
		case string:
			result = append(result, "s:"+value)
		case jsnum.Number:
			result = append(result, NumberKey(float64(value)))
		}
	}
	return result
}

// origin names where a site's vocabulary comes from: a type alias or enum (rank 0), a property (rank 1), or a variable, parameter, or function (rank 2).
type origin struct {
	label       string
	declaration *ast.Node
	rank        int
}

// origins returns where a site's vocabulary comes from. A property read through a union of object types has one origin per declaration.
func (c *Classifier) origins(n *ast.Node) []origin {
	n = unwrap(n)
	if t := c.declared(n); t != nil {
		for _, candidate := range []*checker.Type{t, c.checker.GetNonNullableType(t)} {
			if alias := candidate.Alias(); alias != nil {
				return []origin{{alias.Symbol().Name, firstDeclaration(alias.Symbol()), 0}}
			}
			if symbol := candidate.Symbol(); symbol != nil && symbol.Flags&ast.SymbolFlagsEnum != 0 {
				return []origin{{symbol.Name, firstDeclaration(symbol), 0}}
			}
		}
	}
	switch n.Kind {
	case ast.KindPropertyAccessExpression:
		if symbol := c.checker.GetSymbolAtLocation(n.Name()); symbol != nil {
			if symbol.Parent != nil && !strings.HasPrefix(symbol.Parent.Name, "__") {
				return []origin{{symbol.Parent.Name + "." + symbol.Name, symbol.ValueDeclaration, 1}}
			}
			result := []origin{}
			for _, declaration := range symbol.Declarations {
				label := symbol.Name
				if container := container(declaration); container != "" {
					label = container + "." + label
				}
				result = append(result, origin{label, declaration, 1})
			}
			if len(result) > 0 {
				return result
			}
			return []origin{{symbol.Name, nil, 1}}
		}
	case ast.KindIdentifier:
		if symbol := c.checker.GetSymbolAtLocation(n); symbol != nil {
			return []origin{{symbol.Name, symbol.ValueDeclaration, 2}}
		}
	case ast.KindCallExpression:
		if signature := c.checker.GetResolvedSignature(n); signature != nil && signature.Declaration() != nil && signature.Declaration().Name() != nil {
			return []origin{{signature.Declaration().Name().Text() + "()", signature.Declaration(), 2}}
		}
	}
	return []origin{{"value", nil, 3}}
}

// container returns the name of the type alias, interface, or class enclosing declaration, when no function, variable, or other named declaration intervenes.
func container(declaration *ast.Node) string {
	for n := declaration.Parent; n != nil && n.Kind != ast.KindSourceFile; n = n.Parent {
		switch n.Kind {
		case ast.KindTypeAliasDeclaration, ast.KindInterfaceDeclaration, ast.KindClassDeclaration, ast.KindClassExpression:
			if n.Name() != nil {
				return n.Name().Text()
			}
			return ""
		}
		if n.Name() != nil {
			return ""
		}
	}
	return ""
}

func firstDeclaration(symbol *ast.Symbol) *ast.Node {
	if len(symbol.Declarations) == 0 {
		return nil
	}
	return symbol.Declarations[0]
}

type group struct {
	first Site
	local int
	files map[*ast.SourceFile]bool
	sites []Site
}

type measure struct {
	ctx        rule.RuleContext
	classifier *Classifier
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

// listed joins items, showing at most shown of them.
func listed(items []string, shown int) string {
	if len(items) <= shown {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(items[:shown], ", "), len(items)-shown)
}

// values lists the vocabulary's sorted literals.
func (m *measure) values(union *checker.Type) []string {
	texts := []string{}
	for _, member := range union.Types() {
		texts = append(texts, m.ctx.TypeChecker.TypeToString(member))
	}
	slices.Sort(texts)
	return texts
}

// nearest orders the group's other files by how many leading directories they share with the linted file, then by path.
func (m *measure) nearest(files map[*ast.SourceFile]bool) []string {
	own := strings.Split(tspath.GetDirectoryPath(m.ctx.SourceFile.FileName()), "/")
	shared := func(file *ast.SourceFile) int {
		directories := strings.Split(tspath.GetDirectoryPath(file.FileName()), "/")
		count := 0
		for count < len(own) && count < len(directories) && own[count] == directories[count] {
			count++
		}
		return count
	}
	others := []*ast.SourceFile{}
	for file := range files {
		if file != m.ctx.SourceFile {
			others = append(others, file)
		}
	}
	slices.SortFunc(others, func(a, b *ast.SourceFile) int {
		if byShared := shared(b) - shared(a); byShared != 0 {
			return byShared
		}
		return strings.Compare(a.FileName(), b.FileName())
	})
	result := []string{}
	for _, file := range others {
		result = append(result, m.relative(file.FileName()))
	}
	return result
}

// origins returns the group's distinct origins, by how many sites read them, then type aliases and enums before properties before other values, and the first origin's label.
func (m *measure) origins(sites []Site) ([]string, string) {
	type entry struct {
		origin
		text  string
		sites int
	}
	byText := map[string]*entry{}
	for _, s := range sites {
		for _, o := range m.classifier.origins(s.Node) {
			text := o.label
			if o.declaration != nil {
				text += " (" + m.relative(ast.GetSourceFileOfNode(o.declaration).FileName()) + ")"
			}
			if byText[text] == nil {
				byText[text] = &entry{origin: o, text: text}
			}
			byText[text].sites++
		}
	}
	entries := []*entry{}
	for _, e := range byText {
		entries = append(entries, e)
	}
	slices.SortFunc(entries, func(a, b *entry) int {
		if a.sites != b.sites {
			return b.sites - a.sites
		}
		if a.rank != b.rank {
			return a.rank - b.rank
		}
		return strings.Compare(a.text, b.text)
	})
	result := []string{}
	for _, e := range entries {
		result = append(result, e.text)
	}
	return result, entries[0].label
}

func (m *measure) report(g *group, limit int) {
	origins, _ := m.origins(g.sites)
	local := []Site{}
	for _, s := range g.sites {
		if ast.GetSourceFileOfNode(s.Node) == m.ctx.SourceFile {
			local = append(local, s)
		}
	}
	_, name := m.origins(local)
	checks := "1 such check"
	if g.local > 1 {
		checks = fmt.Sprintf("%d such checks", g.local)
	}
	values := m.values(g.first.Union)
	shown := strings.Join(values, " | ")
	help := fmt.Sprintf("Checked in %d files. Other files, nearest first: %s. Declared as: %s.", len(g.files), listed(m.nearest(g.files), 3), listed(origins, 4))
	if len(values) > 5 {
		shown = fmt.Sprintf("%s | … (%d values)", strings.Join(values[:4], " | "), len(values))
		help += " Values: " + strings.Join(values, " | ") + "."
	}
	limited := fmt.Sprintf("%d files", limit)
	if limit == 1 {
		limited = "1 file"
	}
	m.ctx.ReportNode(g.first.Node, rule.RuleMessage{
		Id: "spreadInterpretation",
		Description: fmt.Sprintf("`%s` (%s) is checked for particular values in more than %s. The compiler points to none of these checks when a value is added. This file has %s.",
			name, shown, limited, checks),
		Help: help,
	})
}

func (m *measure) run(limit int) {
	index := IndexFor(m.ctx.Program)
	groups := map[string]*group{}
	order := []string{}
	for _, n := range index.File(m.ctx.SourceFile) {
		if s, ok := m.classifier.Classify(n); ok {
			if groups[s.Key] == nil {
				groups[s.Key] = &group{first: s, files: map[*ast.SourceFile]bool{}}
				order = append(order, s.Key)
			}
			groups[s.Key].local++
		}
	}
	if len(order) == 0 {
		return
	}
	seen := map[*ast.Node]bool{}
	visit := func(n *ast.Node) {
		if seen[n] {
			return
		}
		seen[n] = true
		if s, ok := m.classifier.Classify(n); ok {
			if g := groups[s.Key]; g != nil {
				g.files[ast.GetSourceFileOfNode(n)] = true
				g.sites = append(g.sites, s)
			}
		}
	}
	for _, key := range order {
		for _, value := range Keys(groups[key].first.Union) {
			for _, n := range index.Keyed(value) {
				visit(n)
			}
		}
	}
	for _, n := range index.Other() {
		visit(n)
	}
	for _, key := range order {
		if len(groups[key].files) > limit {
			m.report(groups[key], limit)
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/max-interpretation-spread",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		limit := parseOptions(options)
		(&measure{ctx: ctx, classifier: NewClassifier(ctx.TypeChecker)}).run(limit)
		return rule.RuleListeners{}
	},
}
