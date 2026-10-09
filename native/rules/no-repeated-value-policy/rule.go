package no_repeated_value_policy

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/rules/no-repeated-value-group"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type measure struct {
	ctx        rule.RuleContext
	classifier *no_repeated_value_group.Classifier
	index      *no_repeated_value_group.Index
	places     no_repeated_value_group.Places
}

func key(s *no_repeated_value_group.Site) string { return s.Canon + "\x00" + s.Outcome }

// others returns one site per other file asking s's question with s's outcome.
func (m *measure) others(s *no_repeated_value_group.Site) []*ast.Node {
	byFile := map[*ast.SourceFile]*ast.Node{}
	for _, root := range m.index.Outcome(s.Outcome) {
		file := ast.GetSourceFileOfNode(root)
		if file == m.ctx.SourceFile || byFile[file] != nil {
			continue
		}
		if other := m.classifier.Classify(root); other != nil && key(other) == key(s) {
			byFile[file] = root
		}
	}
	result := []*ast.Node{}
	for _, root := range byFile {
		result = append(result, root)
	}
	return result
}

func values(leaf *no_repeated_value_group.Leaf) string {
	if len(leaf.Accepted) <= len(leaf.Vocabulary.Values)-len(leaf.Accepted) {
		return leaf.Vocabulary.Display(leaf.Accepted)
	}
	rejected := []string{}
	for _, value := range leaf.Vocabulary.Values {
		found := false
		for _, accepted := range leaf.Accepted {
			found = found || accepted == value
		}
		if !found {
			rejected = append(rejected, value)
		}
	}
	return "anything but " + leaf.Vocabulary.Display(rejected)
}

func (m *measure) report(s *no_repeated_value_group.Site, others []*ast.Node) {
	leaf := s.Leaves[0]
	m.places.Sort(others)
	locations := []string{}
	for _, n := range others {
		locations = append(locations, m.places.Location(n))
	}
	m.ctx.ReportRange(core.NewTextRange(s.Pos, s.End), rule.RuleMessage{
		Id:          "repeatedValuePolicy",
		Description: fmt.Sprintf("Testing `%s` for %s leads to the same outcome in other files.", no_repeated_value_group.Text(leaf.Subject, 40), values(leaf)),
		Help: fmt.Sprintf("The same test and outcome appear in %d files. Others, nearest first: %s. When this rule changes, every copy has to be found and changed the same way.",
			len(others)+1, no_repeated_value_group.Listed(locations, 3)),
	})
}

func (m *measure) run() {
	for _, root := range m.index.File(m.ctx.SourceFile) {
		s := m.classifier.Classify(root)
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
		no_repeated_value_group.ParseOptions("no-repeated-value-policy", options)
		(&measure{
			ctx:        ctx,
			classifier: no_repeated_value_group.NewClassifier(ctx.TypeChecker),
			index:      no_repeated_value_group.IndexFor(ctx.Program),
			places:     no_repeated_value_group.NewPlaces(ctx),
		}).run()
		return rule.RuleListeners{}
	},
}
