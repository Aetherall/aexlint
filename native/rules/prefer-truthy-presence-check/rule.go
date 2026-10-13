package prefer_truthy_presence_check

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func isUndefined(node *ast.Node, tc *checker.Checker) bool {
	node = ast.SkipParentheses(node)
	return node.Kind == ast.KindIdentifier && node.Text() == "undefined" && tc.GetTypeAtLocation(node).Flags() == checker.TypeFlagsUndefined
}

func check(ctx rule.RuleContext, a *model.Analyzer, node *ast.Node) {
	binary := node.AsBinaryExpression()
	comparison, negated := false, false
	switch binary.OperatorToken.Kind {
	case ast.KindEqualsEqualsToken, ast.KindEqualsEqualsEqualsToken:
		comparison = true
	case ast.KindExclamationEqualsToken, ast.KindExclamationEqualsEqualsToken:
		comparison, negated = true, true
	}
	if !comparison || !model.Boolean(node) {
		return
	}
	value := binary.Left
	if isUndefined(binary.Left, ctx.TypeChecker) {
		value = binary.Right
	} else if !isUndefined(binary.Right, ctx.TypeChecker) {
		return
	}
	t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, ast.SkipParentheses(value))
	if slices.ContainsFunc(t.Distributed(), func(part *checker.Type) bool { return part.Flags()&checker.TypeFlagsEnumLike != 0 }) {
		return
	}
	test, ok := a.TestOf(node)
	held := a.HeldBy(t)
	claim, truthiness := test.Claim.Under(held), model.Truthiness().Under(held)
	if !ok || !slices.Contains(held, model.MissingUndefined) || !claim.Splits() || !claim.Same(truthiness) && !claim.Same(truthiness.Not()) {
		return
	}
	message := rule.RuleMessage{Id: "preferTruthyCheck", Description: "Use a truthiness check instead of comparing with undefined; every present value of this type is truthy."}
	if !negated {
		message = rule.RuleMessage{Id: "preferFalsyCheck", Description: "Use a negated truthiness check instead of comparing with undefined; every present value of this type is truthy."}
	}
	ctx.ReportNode(node, message)
}

var Rule = rule.Rule{
	Name: "aexlint/prefer-truthy-presence-check",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("prefer-truthy-presence-check", options)
		compilerOptions := ctx.Program.Options()
		if !utils.IsStrictCompilerOptionEnabled(compilerOptions, compilerOptions.StrictNullChecks) {
			ctx.ReportRange(core.NewTextRange(0, 0), rule.RuleMessage{
				Id: "strictNullChecksRequired", Description: "prefer-truthy-presence-check requires strictNullChecks to be enabled.",
			})
			return nil
		}
		a := model.NewAnalyzer(ctx, model.NewClassifier(ctx.TypeChecker))
		return rule.RuleListeners{ast.KindBinaryExpression: func(node *ast.Node) { check(ctx, a, node) }}
	},
}
