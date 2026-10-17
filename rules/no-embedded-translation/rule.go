package no_embedded_translation

import (
	"fmt"
	"strings"

	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func embedded(a *model.Analyzer, d *model.Decision, b model.Branch) bool {
	reads := b.Outcome.Source == model.PartsResult || b.Outcome.Source == model.MixedResult
	return b.Tested && reads && a.KeepsResult(b.Body) && len(a.VariantReads(d, b)) > 0
}

func report(ctx rule.RuleContext, a *model.Analyzer, d *model.Decision) {
	values := []string{}
	for _, b := range d.Arms {
		if embedded(a, d, b) {
			values = append(values, b.Claim.Vocabulary.Display(b.Claim.Accepted))
		}
	}
	if len(values) == 0 {
		return
	}
	ctx.ReportNode(d.Node, rule.RuleMessage{
		Id:          "embeddedTranslation",
		Description: fmt.Sprintf("Branches on `%s` read its fields into this code's own state.", model.Text(d.Node, 40)),
		Help: fmt.Sprintf("Branches for %s turn the value into this code's own model in the middle of other work. A projection, a function from the value to that model that returns it, keeps the translation in one place and leaves this logic free of variant tests.",
			strings.Join(values, "; ")),
	})
}

var Rule = rule.Rule{
	Name: "aexlint/no-embedded-translation",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-embedded-translation", options)
		a := model.NewAnalyzer(ctx)
		for _, d := range model.Dispatches(a) {
			report(ctx, a, d)
		}
		return rule.RuleListeners{}
	},
}
