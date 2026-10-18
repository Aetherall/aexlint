package no_repeated_value_group

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type measure struct {
	ctx      rule.RuleContext
	analyzer *model.Analyzer
	index    *model.Index
	places   model.Places
}

// matches returns the other unnamed sites and the named sites spelling out s's group, apart from those of s's own
// definition.
func (m *measure) matches(s *model.Site) ([]*ast.Node, []*model.Site) {
	seen := map[*ast.Node]bool{s.Root: true}
	others, named := []*ast.Node{}, []*model.Site{}
	visit := func(root *ast.Node) {
		if seen[root] {
			return
		}
		seen[root] = true
		other := m.analyzer.Classify(root)
		switch {
		case other == nil || other.Group != s.Group:
		case other.Named == "":
			others = append(others, root)
		case s.Definition == nil || other.Definition != s.Definition:
			named = append(named, other)
		}
	}
	for _, values := range [][]string{s.Tests[0].Claim.Accepted, s.Tests[0].Claim.Rejected()} {
		for _, root := range m.index.Keyed(values[0]) {
			visit(root)
		}
	}
	for _, root := range m.index.Other() {
		visit(root)
	}
	return others, named
}

func (m *measure) names(named []*model.Site) string {
	names := []string{}
	for _, n := range named {
		names = append(names, fmt.Sprintf("%s (%s)", n.Named, m.places.Location(n.Definition)))
	}
	slices.Sort(names)
	return model.Listed(slices.Compact(names), 3)
}

func (m *measure) report(s *model.Site, others []*ast.Node, named []*model.Site) {
	group := s.Tests[0].Claim.Vocabulary.Display(s.Tests[0].Claim.Accepted)
	var description string
	switch s.Root.Kind {
	case ast.KindArrayLiteralExpression:
		description = fmt.Sprintf("This list spells out the value group %s, which is also written elsewhere.", group)
	case ast.KindCaseClause:
		description = fmt.Sprintf("These cases group the values %s of `%s`, a group also written elsewhere.", group, model.Text(s.Tests[0].Value.Node, 40))
	default:
		description = fmt.Sprintf("`%s` is tested against the value group %s, which is also written elsewhere.", model.Text(s.Tests[0].Value.Node, 40), group)
	}
	m.places.Sort(others)
	locations := []string{}
	for _, n := range others {
		locations = append(locations, m.places.Location(n))
	}
	parts := []string{}
	if len(others) > 0 {
		parts = append(parts, fmt.Sprintf("Written in %d places. Others, nearest first: %s.", len(others)+1, model.Listed(locations, 3)))
	}
	if len(named) > 0 {
		parts = append(parts, "Already named by "+m.names(named)+".")
	}
	parts = append(parts, "When the group changes, every copy has to be found and changed the same way.")
	m.ctx.ReportRange(core.NewTextRange(s.Pos, s.End), rule.RuleMessage{Id: "repeatedValueGroup", Description: description, Help: strings.Join(parts, " ")})
}

func (m *measure) reportDefinition(s *model.Site, named []*model.Site) {
	group := s.Tests[0].Claim.Vocabulary.Display(s.Tests[0].Claim.Accepted)
	m.ctx.ReportRange(core.NewTextRange(s.Pos, s.End), rule.RuleMessage{
		Id:          "repeatedValueGroup",
		Description: fmt.Sprintf("`%s` defines the value group %s, which is also defined elsewhere.", s.Named, group),
		Help:        "Also defined by " + m.names(named) + ". Separate definitions of one group drift apart when it changes; keep one and use it everywhere.",
	})
}

func (m *measure) run() {
	definitions := map[*ast.Node]bool{}
	for _, root := range m.index.File(m.ctx.SourceFile) {
		s := m.analyzer.Classify(root)
		if s == nil || s.Group == "" || definitions[s.Definition] {
			continue
		}
		others, named := m.matches(s)
		switch {
		case s.Named == "":
			if len(others)+len(named) > 0 {
				m.report(s, others, named)
			}
		case len(named) > 0:
			definitions[s.Definition] = true
			m.reportDefinition(s, named)
		default:
			definitions[s.Definition] = true
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-repeated-value-group",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-repeated-value-group", options)
		(&measure{ctx: ctx, analyzer: model.NewAnalyzer(ctx), index: model.IndexFor(ctx.Program), places: model.NewPlaces(ctx)}).run()
		return rule.RuleListeners{}
	},
}
