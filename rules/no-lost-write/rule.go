package no_lost_write

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func check(ctx rule.RuleContext, a *model.Analyzer, fn *ast.Node) {
	if fn.Body() == nil {
		return
	}
	for _, o := range a.OverwrittenIn(fn) {
		if !a.Lost(o) || a.Fabricated(o) {
			continue
		}
		var written string
		if o.Store.Node.Kind == ast.KindCallExpression {
			written = fmt.Sprintf("`%s` stores %s under `%s`", o.Store.Receiver, model.Text(o.Store.Stored, 30), model.Text(o.Store.Key, 30))
		} else {
			written = fmt.Sprintf("`%s` is set to %s", model.Text(o.Store.Node.AsBinaryExpression().Left, 40), model.Text(o.Store.Stored, 30))
		}
		ctx.ReportNode(o.Store.Node, rule.RuleMessage{
			Id:          "lostWrite",
			Description: fmt.Sprintf("%s here, but `%s` replaces it before anything can read it.", written, model.Text(o.Fills[0].Node, 40)),
			Help:        "Nothing can see this value, so the write does nothing: either it is a leftover, or the code meant to read it in between. Remove it, or read it where it matters.",
		})
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-lost-write",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-lost-write", options)
		a := model.NewAnalyzer(ctx)
		var walk func(*ast.Node) bool
		walk = func(n *ast.Node) bool {
			if ast.IsFunctionLike(n) {
				check(ctx, a, n)
			}
			n.ForEachChild(walk)
			return false
		}
		walk(ctx.SourceFile.AsNode())
		return rule.RuleListeners{}
	},
}
