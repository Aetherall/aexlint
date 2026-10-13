package no_overbroad_type_predicate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func rejectedValue(key string) string {
	if value, ok := strings.CutPrefix(key, "s:"); ok {
		return strconv.Quote(value)
	}
	return strings.TrimPrefix(key, "n:")
}

var Rule = rule.Rule{
	Name: "aexlint/no-overbroad-type-predicate",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-overbroad-type-predicate", options)
		compilerOptions := ctx.Program.Options()
		if !utils.IsStrictCompilerOptionEnabled(compilerOptions, compilerOptions.StrictNullChecks) {
			ctx.ReportRange(core.NewTextRange(0, 0), rule.RuleMessage{
				Id: "strictNullChecksRequired", Description: "no-overbroad-type-predicate requires strictNullChecks to be enabled.",
			})
			return nil
		}
		a := model.NewAnalyzer(ctx, model.NewClassifier(ctx.TypeChecker))
		var visit func(*ast.Node) bool
		visit = func(n *ast.Node) bool {
			if ast.IsFunctionLike(n) {
				if contract := a.PredicateContractOf(n); contract != nil && len(contract.Restrictions) > 0 {
					restriction := contract.Restrictions[0]
					subject := model.Text(restriction.Subject, int(restriction.Subject.End()-restriction.Subject.Pos())+3)
					ctx.ReportNode(restriction.Node, rule.RuleMessage{
						Id: "overbroadTypePredicate",
						Description: fmt.Sprintf("This comparison rejects %s = %s, which the declared predicate target %q permits; a false result does not exclude that target.",
							subject, rejectedValue(restriction.Rejected), ctx.TypeChecker.TypeToString(contract.Target)),
						Help: "A type predicate promises that false excludes its target too. Review the declared target or use a boolean policy; no automatic fix is offered.",
					})
				}
			}
			n.ForEachChild(visit)
			return false
		}
		visit(ctx.SourceFile.AsNode())
		return rule.RuleListeners{}
	},
}
