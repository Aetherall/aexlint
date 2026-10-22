package no_detached_key

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func check(ctx rule.RuleContext, a *model.Analyzer, n *ast.Node) {
	own, ok := a.OwnKeyOf(n)
	if !ok {
		return
	}
	for _, call := range own.Calls {
		ctx.ReportNode(call, rule.RuleMessage{
			Id:          "detachedKey",
			Description: fmt.Sprintf("`%s` is given `%s`, the key `%s` is stored under in `%s`: the value needs its own key, so every reader of `%s` passes the two together.", model.Text(call.Expression(), 40), own.Key.Name, own.Value.Name, own.Container, own.Container),
			Help:        "Each value is made for its key, so the key is part of what it is. Give it to the value where the value is made, and keep the map, if lookups need it, as an index of values that know their key.",
		})
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-detached-key",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-detached-key", options)
		a := model.NewAnalyzer(ctx)
		visit := func(n *ast.Node) { check(ctx, a, n) }
		return rule.RuleListeners{ast.KindForOfStatement: visit, ast.KindCallExpression: visit}
	},
}
