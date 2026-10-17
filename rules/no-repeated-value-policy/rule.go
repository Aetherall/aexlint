package no_repeated_value_policy

import (
	"fmt"

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

func key(s *model.Site) string { return s.Canon + "\x00" + s.Outcome }

// others returns one site per other file asking s's question with s's outcome.
func (m *measure) others(s *model.Site) []*ast.Node {
	byFile := map[*ast.SourceFile]*ast.Node{}
	for _, root := range m.index.Outcome(s.Outcome) {
		file := ast.GetSourceFileOfNode(root)
		if file == m.ctx.SourceFile || byFile[file] != nil {
			continue
		}
		if other := m.analyzer.Classify(root); other != nil && key(other) == key(s) {
			byFile[file] = root
		}
	}
	result := []*ast.Node{}
	for _, root := range byFile {
		result = append(result, root)
	}
	return result
}

func values(c model.Claim) string {
	if len(c.Accepted) <= len(c.Rejected()) {
		return c.Vocabulary.Display(c.Accepted)
	}
	return "anything but " + c.Vocabulary.Display(c.Rejected())
}

func (m *measure) report(s *model.Site, others []*ast.Node) {
	t := s.Tests[0]
	m.places.Sort(others)
	locations := []string{}
	for _, n := range others {
		locations = append(locations, m.places.Location(n))
	}
	m.ctx.ReportRange(core.NewTextRange(s.Pos, s.End), rule.RuleMessage{
		Id:          "repeatedValuePolicy",
		Description: fmt.Sprintf("Testing `%s` for %s leads to the same outcome in other files.", model.Text(t.Value.Node, 40), values(t.Claim)),
		Help: fmt.Sprintf("The same test and outcome appear in %d files. Others, nearest first: %s. When this rule changes, every copy has to be found and changed the same way.",
			len(others)+1, model.Listed(locations, 3)),
	})
}

func (m *measure) run() {
	for _, root := range m.index.File(m.ctx.SourceFile) {
		s := m.analyzer.Classify(root)
		if s == nil || s.Outcome == "" {
			continue
		}
		if others := m.others(s); len(others) > 0 {
			m.report(s, others)
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-repeated-value-policy",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-repeated-value-policy", options)
		(&measure{
			ctx:      ctx,
			analyzer: model.NewAnalyzer(ctx),
			index:    model.IndexFor(ctx.Program),
			places:   model.NewPlaces(ctx),
		}).run()
		return rule.RuleListeners{}
	},
}
