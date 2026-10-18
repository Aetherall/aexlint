package no_widened_union

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const help = "Comparing it with literals loses what the union checks: a misspelled value compiles, and a value added to the union takes the fallback."

func result(ctx rule.RuleContext, a *model.Analyzer, fn *ast.Node) {
	if !a.StringResult(fn) {
		return
	}
	returned, ok := a.Returned(fn)
	if !ok || len(returned) < 2 {
		return
	}
	calls, _ := a.Callers(fn)
	if !slices.ContainsFunc(calls, a.ResultCompared) {
		return
	}
	ctx.ReportNode(fn.Type(), rule.RuleMessage{
		Id:          "widenedResult",
		Description: fmt.Sprintf("The result is declared `%s`, but every return gives %s, and a caller compares it with literals.", model.Text(fn.Type(), 40), strings.Join(returned, " | ")),
		Help:        help + " Declare the result as the union it returns.",
	})
}

func check(ctx rule.RuleContext, a *model.Analyzer, fn *ast.Node) {
	if fn.Type() != nil && fn.Type().Kind == ast.KindTypePredicate {
		return
	}
	result(ctx, a, fn)
	parameters := slices.DeleteFunc(slices.Clone(fn.Parameters()), ast.IsThisParameter)
	for index, parameter := range parameters {
		if !a.StringParameter(parameter) || len(a.LiteralComparisons(fn, parameter)) == 0 {
			continue
		}
		calls, complete := a.Callers(fn)
		if !complete || len(calls) == 0 {
			continue
		}
		passed, widest, ok := a.Passed(calls, index)
		if !ok || widest < 2 {
			continue
		}
		ctx.ReportNode(parameter.Type(), rule.RuleMessage{
			Id:          "widenedUnion",
			Description: fmt.Sprintf("`%s` is declared `%s`, but every call passes %s.", parameter.Name().Text(), model.Text(parameter.Type(), 40), strings.Join(passed, " | ")),
			Help:        help + " Declare the parameter as the union the calls pass.",
		})
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-widened-union",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-widened-union", options)
		a := model.NewAnalyzer(ctx)
		var walk func(*ast.Node) bool
		walk = func(n *ast.Node) bool {
			if ast.IsFunctionLike(n) && n.Body() != nil {
				check(ctx, a, n)
			}
			n.ForEachChild(walk)
			return false
		}
		walk(ctx.SourceFile.AsNode())
		return rule.RuleListeners{}
	},
}
