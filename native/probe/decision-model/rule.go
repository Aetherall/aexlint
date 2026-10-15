package decision_model

import (
	"fmt"
	"strings"

	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func values(vs []string) string {
	out := []string{}
	for _, v := range vs {
		out = append(out, strings.TrimPrefix(v, "s:"))
	}
	return strings.Join(out, ", ")
}

var Rule = rule.Rule{
	Name: "aexlint/decision-model",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		file := model.Build(model.NewAnalyzer(ctx))
		for _, f := range file.Functions {
			for _, d := range f.Domains {
				id, owner := "ownNarrowing", "its own guard"
				if d.Selector != "" {
					id, owner = "selectorNarrowing", fmt.Sprintf("sibling selector `%s`", d.Selector)
				}
				ctx.ReportNode(d.Node, rule.RuleMessage{Id: id, Description: fmt.Sprintf("`%s` is narrowed to %s by %s.", d.Property, values(d.Claim.Accepted), owner)})
			}
		}
		for _, r := range file.Relations {
			if r.Kind == model.Conflicts {
				ctx.ReportNode(file.Decisions[r.A].Node, rule.RuleMessage{Id: "conflict", Description: fmt.Sprintf("Conflicting choice: %s.", r.Detail)})
			}
		}
		return rule.RuleListeners{}
	},
}
