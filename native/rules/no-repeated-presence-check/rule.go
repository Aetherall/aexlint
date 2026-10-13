package no_repeated_presence_check

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func safe(a *model.Analyzer, n *ast.Node, subject model.Value) bool {
	switch n = model.Unwrap(n); n.Kind {
	case ast.KindStringLiteral, ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindNullKeyword:
		return true
	}
	v := a.ValueOf(n)
	return v.Named() && !v.Within(subject)
}

func check(ctx rule.RuleContext, a *model.Analyzer, g model.Guard) {
	if !g.Whole || !g.Claim.Presence() || g.Fallback == "" || len(g.Work) != 1 {
		return
	}
	call, subject := model.Unwrap(g.Work[0]), g.Test.Value
	if call.Kind != ast.KindCallExpression || ast.IsOptionalChain(call) || !safe(a, call.Expression(), subject) {
		return
	}
	var argument *ast.Node
	for _, n := range call.Arguments() {
		switch {
		case argument == nil && a.ValueOf(n).Same(subject):
			argument = n
		case !safe(a, n, subject):
			return
		}
	}
	if argument == nil {
		return
	}
	b, ok := a.BoundaryOf(call, argument)
	held, typed := a.Held(subject.Node)
	caller := g.Claim.Under(held)
	if !ok || !typed || !caller.Splits() || a.Keeps(b, g.Test) {
		return
	}
	for _, e := range a.Rechecks(b, g.Test, caller) {
		if e.Fallback == g.Fallback {
			callee := model.Text(call.Expression(), 60)
			ctx.ReportNode(g.Test.Node, rule.RuleMessage{
				Id:          "repeatedPresenceCheck",
				Description: fmt.Sprintf("`%s` is checked here and again by `%s`.", model.Text(subject.Node, 60), callee),
				Help: fmt.Sprintf("`%s` already returns `%s` when `%s` is missing, so this check changes nothing. Call it directly, or make its parameter required so only one side checks.",
					callee, e.Shown, b.Parameter.Name().Text()),
			})
			return
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-repeated-presence-check",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-repeated-presence-check", options)
		a := model.NewAnalyzer(ctx, model.NewClassifier(ctx.TypeChecker))
		visit := func(n *ast.Node) {
			for _, g := range a.Guards(n) {
				check(ctx, a, g)
			}
		}
		return rule.RuleListeners{ast.KindConditionalExpression: visit, ast.KindBinaryExpression: visit}
	},
}
