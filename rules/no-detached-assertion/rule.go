package no_detached_assertion

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func line(node *ast.Node) int {
	file := ast.GetSourceFileOfNode(node)
	return scanner.GetECMALineOfPosition(file, utils.TrimNodeTextRange(file, node).Pos()) + 1
}

var Rule = rule.Rule{
	Name: "aexlint/no-detached-assertion",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-detached-assertion", options)
		a := model.NewAnalyzer(ctx)
		return rule.RuleListeners{ast.KindNonNullExpression: func(node *ast.Node) {
			backing, ok := a.BackingOf(node)
			if !ok || !backing.Guard {
				return
			}
			ctx.ReportNode(node, rule.RuleMessage{
				Id:          "detachedAssertion",
				Description: fmt.Sprintf("This assertion holds only because of the check `%s` on line %d, which the type of `%s` does not carry.", model.Text(backing.Check, 40), line(backing.Check), backing.Container),
				Help:        "Nothing ties the assertion to the check: moving, changing or dropping the check leaves the assertion in place and wrong. Check the entry itself rather than the size or key, or keep a value that only exists when the entry does, such as the entry read once, or a type for a list that is not empty.",
			})
		}}
	},
}
