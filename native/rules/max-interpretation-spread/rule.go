package max_interpretation_spread

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
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

// origins returns the group's distinct origins, by how many sites read them, then type aliases and enums before properties before other values, and the first origin's label.
func (m *measure) origins(sites []Site) ([]string, string) {
	type entry struct {
		model.Origin
		text  string
		sites int
	}
	byText := map[string]*entry{}
	for _, s := range sites {
		for _, o := range m.analyzer.Origins(s.Node) {
			text := o.Label
			if o.Declaration != nil {
				text += " (" + model.NewPlaces(m.ctx).Relative(ast.GetSourceFileOfNode(o.Declaration).FileName()) + ")"
			}
			if byText[text] == nil {
				byText[text] = &entry{Origin: o, text: text}
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
		if a.Rank != b.Rank {
			return a.Rank - b.Rank
		}
		return strings.Compare(a.text, b.text)
	})
	result := []string{}
	for _, e := range entries {
		result = append(result, e.text)
	}
	return result, entries[0].Label
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
	values := g.first.Vocabulary.Names(g.first.Vocabulary.Values)
	slices.Sort(values)
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
