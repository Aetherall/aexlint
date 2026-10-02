package type_probe

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

var Rule = rule.Rule{
	Name: "aexlint/type-probe",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		return rule.RuleListeners{
			ast.KindCallExpression: func(node *ast.Node) {
				t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, node)
				if utils.IsTypeFlagSet(t, checker.TypeFlagsNumberLike) {
					ctx.ReportNode(node, rule.RuleMessage{
						Id: "numberResult", Description: "Type checker resolved a number result.",
					})
				}
			},
		}
	},
}
