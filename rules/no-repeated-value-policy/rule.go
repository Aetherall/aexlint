package no_repeated_value_policy

import (
	"fmt"
	"slices"

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

// policy is a question and the outcome it leads to, as one site asks it: when true, or for an early-exit guard, when
// false.
type policy struct {
	key     string
	outcome string
	claim   model.Claim
}

func policies(s *model.Site) []policy {
	result := []policy{}
	if s.Outcome != "" {
		result = append(result, policy{s.Canon + "\x00" + s.Outcome, s.Outcome, s.Tests[0].Claim})
	}
	if s.Rest != "" {
		result = append(result, policy{s.Inverse + "\x00" + s.Rest, s.Rest, s.Tests[0].Claim.Not()})
	}
	return result
}

// others returns one site per other file asking p's question with p's outcome.
func (m *measure) others(p policy) []*ast.Node {
	byFile := map[*ast.SourceFile]*ast.Node{}
	for _, root := range m.index.Outcome(p.outcome) {
		file := ast.GetSourceFileOfNode(root)
		if file == m.ctx.SourceFile || byFile[file] != nil {
			continue
		}
		other := m.analyzer.Classify(root)
		if other != nil && slices.ContainsFunc(policies(other), func(o policy) bool { return o.key == p.key }) {
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

func (m *measure) report(s *model.Site, p policy, others []*ast.Node) {
	m.places.Sort(others)
	locations := []string{}
	for _, n := range others {
		locations = append(locations, m.places.Location(n))
	}
	m.ctx.ReportRange(core.NewTextRange(s.Pos, s.End), rule.RuleMessage{
		Id:          "repeatedValuePolicy",
		Description: fmt.Sprintf("Testing `%s` for %s leads to the same outcome in other files.", model.Text(s.Tests[0].Value.Node, 40), values(p.claim)),
		Help: fmt.Sprintf("The same test and outcome appear in %d files. Others, nearest first: %s. When this rule changes, every copy has to be found and changed the same way.",
			len(others)+1, model.Listed(locations, 3)),
	})
}

func (m *measure) run() {
	for _, root := range m.index.File(m.ctx.SourceFile) {
		s := m.analyzer.Classify(root)
		if s == nil {
			continue
		}
		for _, p := range policies(s) {
			if others := m.others(p); len(others) > 0 {
				m.report(s, p, others)
				break
			}
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
