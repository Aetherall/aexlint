package max_interpretation_spread

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const defaultMax = 4

func parseOptions(options any) int {
	return model.ParseMax("max-interpretation-spread", options, defaultMax)
}

// Site is a check of particular values: Node holds a value of Vocabulary. Check is the question it belongs to, or the
// switch for the cases of one.
type Site struct {
	Key        string
	Vocabulary *model.Vocabulary
	Node       *ast.Node
	Check      *ast.Node
}

// sites returns the checks of particular values a root makes. A typed list defines a group of values rather than
// checking one, and an exhaustive switch without default fails to compile when a value is added.
func sites(c *model.Analyzer, root *ast.Node) []Site {
	check := root
	switch root.Kind {
	case ast.KindArrayLiteralExpression:
		return nil
	case ast.KindCaseClause:
		if check = root.Parent.Parent; c.Exhaustive(check) {
			return nil
		}
	}
	s := c.Classify(root)
	if s == nil {
		return nil
	}
	found := []Site{}
	for _, t := range s.Tests {
		found = append(found, Site{strings.Join(t.Claim.Vocabulary.Values, "\x00"), t.Claim.Vocabulary, t.Value.Node, check})
	}
	return found
}

// origin names where a site's vocabulary comes from: a type alias or enum (rank 0), a property (rank 1), or a variable, parameter, or function (rank 2).
type origin struct {
	label       string
	declaration *ast.Node
	rank        int
}

// origins returns where a site's vocabulary comes from. A property read through a union of object types has one origin per declaration.
func origins(c *model.Analyzer, tc *checker.Checker, n *ast.Node) []origin {
	n = model.Unwrap(n)
	if t := c.Declared(n); t != nil {
		for _, candidate := range []*checker.Type{t, tc.GetNonNullableType(t)} {
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
		if symbol := tc.GetSymbolAtLocation(n.Name()); symbol != nil {
			if symbol.Parent != nil && !strings.HasPrefix(symbol.Parent.Name, ast.InternalSymbolNamePrefix) {
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
		if symbol := tc.GetSymbolAtLocation(n); symbol != nil {
			return []origin{{symbol.Name, symbol.ValueDeclaration, 2}}
		}
	case ast.KindCallExpression:
		if signature := tc.GetResolvedSignature(n); signature != nil && signature.Declaration() != nil && signature.Declaration().Name() != nil {
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
	first  Site
	checks map[*ast.Node]bool
	files  map[*ast.SourceFile]bool
	sites  []Site
}

type measure struct {
	ctx      rule.RuleContext
	analyzer *model.Analyzer
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

// origins returns the group's distinct origins, by how many sites read them, then type aliases and enums before properties before other values, and the first origin's label.
func (m *measure) origins(sites []Site) ([]string, string) {
	type entry struct {
		origin
		text  string
		sites int
	}
	byText := map[string]*entry{}
	for _, s := range sites {
		for _, o := range origins(m.analyzer, m.ctx.TypeChecker, s.Node) {
			text := o.label
			if o.declaration != nil {
				text += " (" + model.NewPlaces(m.ctx).Relative(ast.GetSourceFileOfNode(o.declaration).FileName()) + ")"
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
	if len(g.checks) > 1 {
		checks = fmt.Sprintf("%d such checks", len(g.checks))
	}
	values := m.values(g.first.Vocabulary.Type)
	shown := strings.Join(values, " | ")
	help := fmt.Sprintf("Checked in %d files. Other files, nearest first: %s. Declared as: %s.", len(g.files), model.Listed(model.NewPlaces(m.ctx).Nearest(g.files), 3), model.Listed(origins, 4))
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
	index := model.IndexFor(m.ctx.Program)
	groups := map[string]*group{}
	order := []string{}
	for _, root := range index.File(m.ctx.SourceFile) {
		for _, s := range sites(m.analyzer, root) {
			if groups[s.Key] == nil {
				groups[s.Key] = &group{first: s, checks: map[*ast.Node]bool{}, files: map[*ast.SourceFile]bool{}}
				order = append(order, s.Key)
			}
			groups[s.Key].checks[s.Check] = true
		}
	}
	if len(order) == 0 {
		return
	}
	seen := map[*ast.Node]bool{}
	visit := func(root *ast.Node) {
		if seen[root] {
			return
		}
		seen[root] = true
		for _, s := range sites(m.analyzer, root) {
			if g := groups[s.Key]; g != nil {
				g.files[ast.GetSourceFileOfNode(root)] = true
				g.sites = append(g.sites, s)
			}
		}
	}
	for _, key := range order {
		for _, value := range groups[key].first.Vocabulary.Values {
			for _, root := range index.Keyed(value) {
				visit(root)
			}
		}
	}
	for _, root := range index.Other() {
		visit(root)
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
		(&measure{ctx: ctx, analyzer: model.NewAnalyzer(ctx)}).run(limit)
		return rule.RuleListeners{}
	},
}
