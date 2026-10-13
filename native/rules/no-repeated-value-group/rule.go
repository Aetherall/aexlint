package no_repeated_value_group

import (
	"fmt"
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type measure struct {
	ctx        rule.RuleContext
	classifier *model.Classifier
	index      *model.Index
	places     model.Places
}

func complement(s *model.Site) []string {
	result := []string{}
	for _, value := range s.Leaves[0].Vocabulary.Values {
		if !slices.Contains(s.Leaves[0].Accepted, value) {
			result = append(result, value)
		}
	}
	return result
}

// matches returns the other unnamed sites and the named sites spelling out s's group.
func (m *measure) matches(s *model.Site) ([]*ast.Node, []*model.Site) {
	seen := map[*ast.Node]bool{s.Root: true}
	others, named := []*ast.Node{}, []*model.Site{}
	visit := func(root *ast.Node) {
		if seen[root] {
			return
		}
		seen[root] = true
		if other := m.classifier.Classify(root); other != nil && other.Group == s.Group {
			if other.Named != "" {
				named = append(named, other)
			} else {
				others = append(others, root)
			}
		}
	}
	for _, values := range [][]string{s.Leaves[0].Accepted, complement(s)} {
		for _, root := range m.index.Keyed(values[0]) {
			visit(root)
		}
	}
	for _, root := range m.index.Other() {
		visit(root)
	}
	return others, named
}

func (m *measure) report(s *model.Site, others []*ast.Node, named []*model.Site) {
	group := s.Leaves[0].Vocabulary.Display(s.Leaves[0].Accepted)
	var description string
	switch s.Root.Kind {
	case ast.KindArrayLiteralExpression:
		description = fmt.Sprintf("This list spells out the value group %s, which is also written elsewhere.", group)
	case ast.KindCaseClause:
		description = fmt.Sprintf("These cases group the values %s of `%s`, a group also written elsewhere.", group, model.Text(s.Leaves[0].Subject, 40))
	default:
		description = fmt.Sprintf("`%s` is tested against the value group %s, which is also written elsewhere.", model.Text(s.Leaves[0].Subject, 40), group)
	}
	m.places.Sort(others)
	locations := []string{}
	for _, n := range others {
		locations = append(locations, m.places.Location(n))
	}
	help := fmt.Sprintf("Written in %d places. Others, nearest first: %s.", len(others)+1, model.Listed(locations, 3))
	if len(named) > 0 {
		names := []string{}
		for _, n := range named {
			names = append(names, fmt.Sprintf("%s (%s)", n.Named, m.places.Location(n.Root)))
		}
		slices.Sort(names)
		help += " Already named by " + model.Listed(slices.Compact(names), 3) + "."
	}
	help += " When the group changes, every copy has to be found and changed the same way."
	m.ctx.ReportRange(core.NewTextRange(s.Pos, s.End), rule.RuleMessage{Id: "repeatedValueGroup", Description: description, Help: help})
}

func (m *measure) run() {
	for _, root := range m.index.File(m.ctx.SourceFile) {
		s := m.classifier.Classify(root)
		if s == nil || s.Group == "" || s.Named != "" {
			continue
		}
		if others, named := m.matches(s); len(others) > 0 {
			m.report(s, others, named)
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-repeated-value-group",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-repeated-value-group", options)
		(&measure{ctx: ctx, classifier: model.NewClassifier(ctx.TypeChecker), index: model.IndexFor(ctx.Program), places: model.NewPlaces(ctx)}).run()
		return rule.RuleListeners{}
	},
}
