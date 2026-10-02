package prefer_truthy_presence_check

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/jsnum"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func unparenthesize(node *ast.Node) *ast.Node {
	for node.Kind == ast.KindParenthesizedExpression {
		node = node.AsParenthesizedExpression().Expression
	}
	return node
}

func isBooleanContext(node *ast.Node) bool {
	for parent := node.Parent; parent != nil; parent = node.Parent {
		switch parent.Kind {
		case ast.KindParenthesizedExpression:
		case ast.KindBinaryExpression:
			op := parent.AsBinaryExpression().OperatorToken.Kind
			if op != ast.KindAmpersandAmpersandToken && op != ast.KindBarBarToken {
				return false
			}
		case ast.KindPrefixUnaryExpression:
			return parent.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken
		case ast.KindIfStatement:
			return parent.AsIfStatement().Expression == node
		case ast.KindWhileStatement:
			return parent.AsWhileStatement().Expression == node
		case ast.KindDoStatement:
			return parent.AsDoStatement().Expression == node
		case ast.KindForStatement:
			return parent.AsForStatement().Condition == node
		case ast.KindConditionalExpression:
			return parent.AsConditionalExpression().Condition == node
		default:
			return false
		}
		node = parent
	}
	return false
}

func isUndefined(node *ast.Node, tc *checker.Checker) bool {
	node = unparenthesize(node)
	return node.Kind == ast.KindIdentifier && node.Text() == "undefined" && tc.GetTypeAtLocation(node).Flags() == checker.TypeFlagsUndefined
}

func isTruthy(t *checker.Type, tc *checker.Checker, htmlAll *checker.Type) bool {
	flags := t.Flags()
	if flags&checker.TypeFlagsEnumLike != 0 {
		return false
	}
	switch {
	case flags&checker.TypeFlagsStringLiteral != 0:
		return t.AsLiteralType().Value().(string) != ""
	case flags&checker.TypeFlagsNumberLiteral != 0:
		value := t.AsLiteralType().Value().(jsnum.Number)
		return value != 0 && !value.IsNaN() && !value.IsInf()
	case flags&checker.TypeFlagsBigIntLiteral != 0:
		return t.AsLiteralType().Value().(jsnum.PseudoBigInt).Sign() != 0
	case flags&checker.TypeFlagsBooleanLiteral != 0:
		return t.AsLiteralType().Value() == true
	case flags&checker.TypeFlagsESSymbolLike != 0:
		return true
	case flags&(checker.TypeFlagsObject|checker.TypeFlagsNonPrimitive) != 0:
		if htmlAll != nil && tc.IsTypeAssignableTo(htmlAll, t) {
			return false
		}
		for _, primitive := range []*checker.Type{tc.GetStringType(), tc.GetNumberType(), tc.GetBooleanType(), tc.GetBigIntType()} {
			if tc.IsTypeAssignableTo(primitive, t) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func canUseTruthiness(t *checker.Type, loose bool, tc *checker.Checker, htmlAll *checker.Type) bool {
	hasUndefined, hasPresent := false, false
	for _, part := range t.Distributed() {
		switch part.Flags() {
		case checker.TypeFlagsUndefined:
			hasUndefined = true
		case checker.TypeFlagsNull:
			if !loose {
				return false
			}
		default:
			if !isTruthy(part, tc, htmlAll) {
				return false
			}
			hasPresent = true
		}
	}
	return hasUndefined && hasPresent
}

var Rule = rule.Rule{
	Name: "aexlint/prefer-truthy-presence-check",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		if options != nil {
			object, ok := options.(map[string]any)
			if !ok || len(object) != 0 {
				panic("aexlint/prefer-truthy-presence-check accepts no options; omit options or use an empty object")
			}
		}
		compilerOptions := ctx.Program.Options()
		if !utils.IsStrictCompilerOptionEnabled(compilerOptions, compilerOptions.StrictNullChecks) {
			ctx.ReportRange(core.NewTextRange(0, 0), rule.RuleMessage{
				Id: "strictNullChecksRequired", Description: "prefer-truthy-presence-check requires strictNullChecks to be enabled.",
			})
			return nil
		}
		var htmlAll *checker.Type
		if symbol := ctx.TypeChecker.ResolveName("HTMLAllCollection", nil, ast.SymbolFlagsType, false); symbol != nil {
			htmlAll = ctx.TypeChecker.GetDeclaredTypeOfSymbol(symbol)
		}
		return rule.RuleListeners{
			ast.KindBinaryExpression: func(node *ast.Node) {
				binary := node.AsBinaryExpression()
				loose, equal := false, false
				switch binary.OperatorToken.Kind {
				case ast.KindEqualsEqualsToken:
					loose, equal = true, true
				case ast.KindExclamationEqualsToken:
					loose = true
				case ast.KindEqualsEqualsEqualsToken:
					equal = true
				case ast.KindExclamationEqualsEqualsToken:
				default:
					return
				}
				if !isBooleanContext(node) {
					return
				}
				var value *ast.Node
				if isUndefined(binary.Left, ctx.TypeChecker) {
					value = binary.Right
				} else if isUndefined(binary.Right, ctx.TypeChecker) {
					value = binary.Left
				} else {
					return
				}
				t := utils.GetConstrainedTypeAtLocation(ctx.TypeChecker, unparenthesize(value))
				if !canUseTruthiness(t, loose, ctx.TypeChecker, htmlAll) {
					return
				}
				message := rule.RuleMessage{Id: "preferTruthyCheck", Description: "Use a truthiness check instead of comparing with undefined; every present value of this type is truthy."}
				if equal {
					message = rule.RuleMessage{Id: "preferFalsyCheck", Description: "Use a negated truthiness check instead of comparing with undefined; every present value of this type is truthy."}
				}
				ctx.ReportNode(node, message)
			},
		}
	},
}
