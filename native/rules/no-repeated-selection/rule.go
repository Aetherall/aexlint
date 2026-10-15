package no_repeated_selection

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func api(ctx rule.RuleContext, analyzer *model.Analyzer, container *ast.Node) (string, bool) {
	if container == nil || container.Kind != ast.KindObjectLiteralExpression {
		return "", false
	}
	t := checker.Checker_getContextualType(ctx.TypeChecker, container, checker.ContextFlagsNone)
	if t == nil {
		return "", false
	}
	t = ctx.TypeChecker.GetNonNullableType(t)
	symbol := t.Alias().Symbol()
	if symbol == nil {
		symbol = t.Symbol()
	}
	if owner := analyzer.ProjectOwner(symbol); owner != nil {
		return owner.Name, false
	}
	return "", true
}

var Rule = rule.Rule{
	Name: "aexlint/no-repeated-selection",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-repeated-selection", options)
		analyzer := model.NewAnalyzer(ctx)
		file := model.Build(analyzer)
		for _, f := range file.Functions {
			reported := map[*ast.Symbol]bool{}
			for _, d := range f.Domains {
				if d.Selector == "" || reported[d.Value.Root] {
					continue
				}
				reported[d.Value.Root] = true
				cause := fmt.Sprintf("`%s` selects values, but `%s` receives the whole type and decides again.", d.Selector, d.Handler)
				help := fmt.Sprintf("%s Carry the narrowed selected value into `%s`, or reuse a shared selection helper instead of repeating its test.", cause, d.Handler)
				switch name, dependency := api(ctx, analyzer, d.Container); {
				case name != "":
					help = fmt.Sprintf("%s Carry the narrowed selected value through `%s` into `%s`, or reuse a shared selection helper.", cause, name, d.Handler)
				case dependency:
					help = fmt.Sprintf("%s Use a local selected-value adapter, or reuse a shared selection helper in `%s`.", cause, d.Handler)
				}
				help += " Use a type predicate only when false excludes every value of its target too."
				ctx.ReportNode(d.Node, rule.RuleMessage{
					Id:          "repeatedSelection",
					Description: fmt.Sprintf("This repeats a decision already made by `%s`.", d.Selector),
					Help:        help,
				})
			}
		}
		return rule.RuleListeners{}
	},
}
