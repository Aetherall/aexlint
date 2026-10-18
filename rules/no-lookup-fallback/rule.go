package no_lookup_fallback

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func picks(o model.Outcome) bool {
	return !o.Commands && (o.Source == model.ConstantResult || o.Source == model.EmptyResult)
}

// fallback returns the arm a decision takes when no test holds, and its tested arms.
func fallback(d *model.Decision) (*model.Branch, []model.Branch) {
	if n := len(d.Arms); n > 0 && !d.Arms[n-1].Tested {
		return &d.Arms[n-1], d.Arms[:n-1]
	}
	return d.Following, d.Arms
}

func head(n *ast.Node) *ast.Node {
	switch {
	case n.Kind == ast.KindBlock && len(n.Statements()) == 1:
		return head(n.Statements()[0])
	case n.Kind == ast.KindReturnStatement && n.Expression() != nil:
		return n.Expression()
	}
	return n
}

func report(ctx rule.RuleContext, a *model.Analyzer, d *model.Decision) {
	rest, tested := fallback(d)
	if rest == nil || len(rest.Body) == 0 || len(tested) < 2 || !picks(rest.Outcome) {
		return
	}
	named := 0
	for _, b := range tested {
		if !b.Complete || !picks(b.Outcome) {
			return
		}
		named += len(b.Claim.Accepted)
	}
	absorbed := rest.Claim.Accepted
	if len(absorbed) < 2 || len(absorbed) >= named || a.FromDependency(d.Node) {
		return
	}
	n := head(rest.Body[0])
	ctx.ReportNode(n, rule.RuleMessage{
		Id:          "lookupFallback",
		Description: fmt.Sprintf("`%s` answers for %s of `%s` without naming them.", model.Text(n, 40), d.Vocabulary.Display(absorbed), model.Text(d.Node, 40)),
		Help:        "The other branches answer for one value each, and a value added to the union will get this answer too. Name these values, or use a complete table or exhaustive switch so a new value is a type error.",
	})
}

var Rule = rule.Rule{
	Name: "aexlint/no-lookup-fallback",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-lookup-fallback", options)
		a := model.NewAnalyzer(ctx)
		for _, d := range model.Dispatches(a) {
			report(ctx, a, d)
		}
		return rule.RuleListeners{}
	},
}
