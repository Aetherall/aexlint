package no_repeated_value_group

import (
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/scanner"
)

func Squash(s string) string { return strings.Join(strings.Fields(s), " ") }

func source(n *ast.Node) string {
	text := ast.GetSourceFileOfNode(n).Text()
	return text[scanner.SkipTrivia(text, n.Pos()):n.End()]
}

// Text returns n's source text with whitespace collapsed, shortened to limit bytes.
func Text(n *ast.Node, limit int) string {
	s := Squash(source(n))
	if len(s) > limit {
		return s[:limit-3] + "..."
	}
	return s
}

func parent(n *ast.Node) (*ast.Node, *ast.Node) {
	child, p := n, n.Parent
	for p != nil && p.Kind == ast.KindParenthesizedExpression {
		child, p = p, p.Parent
	}
	return child, p
}

func question(n *ast.Node) bool {
	switch n = Unwrap(n); n.Kind {
	case ast.KindBinaryExpression:
		binary := n.AsBinaryExpression()
		if ok, _ := equality(binary.OperatorToken.Kind); ok {
			return true
		}
		switch binary.OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
			return question(binary.Left) || question(binary.Right)
		}
	case ast.KindPrefixUnaryExpression:
		unary := n.AsPrefixUnaryExpression()
		return unary.Operator == ast.KindExclamationToken && question(unary.Operand)
	case ast.KindCallExpression:
		_, ok := includes(n)
		return ok
	}
	return false
}

func valueLike(n *ast.Node) bool {
	switch Unwrap(n).Kind {
	case ast.KindJsxElement, ast.KindJsxSelfClosingElement, ast.KindJsxFragment, ast.KindObjectLiteralExpression, ast.KindArrayLiteralExpression,
		ast.KindArrowFunction, ast.KindFunctionExpression, ast.KindStringLiteral, ast.KindTemplateExpression, ast.KindNumericLiteral:
		return true
	}
	return false
}

func partOfQuestion(n *ast.Node) bool {
	_, p := parent(n)
	switch p.Kind {
	case ast.KindPrefixUnaryExpression:
		return p.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken
	case ast.KindBinaryExpression:
		switch p.AsBinaryExpression().OperatorToken.Kind {
		case ast.KindAmpersandAmpersandToken, ast.KindBarBarToken:
			return !valueLike(p.AsBinaryExpression().Right)
		}
	}
	return false
}

func listContext(n *ast.Node) (*ast.Node, *ast.Node) {
	outer, p := n, n.Parent
	for p != nil && (p.Kind == ast.KindParenthesizedExpression || p.Kind == ast.KindAsExpression || p.Kind == ast.KindSatisfiesExpression) {
		outer, p = p, p.Parent
	}
	return outer, p
}

func listElement(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindPropertyAccessExpression:
		return true
	case ast.KindPrefixUnaryExpression:
		return n.AsPrefixUnaryExpression().Operand.Kind == ast.KindNumericLiteral
	}
	return false
}

func list(n *ast.Node) bool {
	elements := n.Elements()
	if len(elements) < 2 {
		return false
	}
	for _, element := range elements {
		if !listElement(element) {
			return false
		}
	}
	outer, p := listContext(n)
	return p != nil && (p.Kind != ast.KindPropertyAccessExpression || p.Expression() != outer)
}

func groupStart(clause *ast.Node) bool {
	if clause.Kind != ast.KindCaseClause {
		return false
	}
	clauses := clause.Parent.AsCaseBlock().Clauses.Nodes
	for i, c := range clauses {
		if c == clause {
			return i == 0 || len(clauses[i-1].Statements()) > 0 || clauses[i-1].Kind == ast.KindDefaultClause
		}
	}
	return false
}

// Root reports whether n is a node the classifier may reduce to a site, from syntax alone.
func Root(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindCaseClause:
		return groupStart(n)
	case ast.KindArrayLiteralExpression:
		return list(n)
	case ast.KindParenthesizedExpression:
		return false
	case ast.KindBinaryExpression:
		if binary := n.AsBinaryExpression(); binary.OperatorToken.Kind == ast.KindAmpersandAmpersandToken && valueLike(binary.Right) {
			return false
		}
	}
	return ast.IsExpression(n) && question(n) && !partOfQuestion(n)
}

func trivialValue(n *ast.Node) bool {
	switch n = Unwrap(n); n.Kind {
	case ast.KindNullKeyword, ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindVoidExpression:
		return true
	case ast.KindIdentifier:
		return n.Text() == "undefined"
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return n.Text() == ""
	case ast.KindNumericLiteral:
		return n.Text() == "0"
	case ast.KindPrefixUnaryExpression:
		return n.AsPrefixUnaryExpression().Operand.Kind == ast.KindNumericLiteral && n.AsPrefixUnaryExpression().Operand.Text() == "1"
	case ast.KindArrayLiteralExpression:
		return len(n.Elements()) == 0
	case ast.KindObjectLiteralExpression:
		return len(n.AsObjectLiteralExpression().Properties.Nodes) == 0
	}
	return false
}

func statementOutcome(statements []*ast.Node) string {
	if len(statements) == 1 && statements[0].Kind == ast.KindBlock {
		statements = statements[0].Statements()
	}
	if n := len(statements); n > 0 && statements[n-1].Kind == ast.KindBreakStatement && statements[n-1].Label() == nil {
		statements = statements[:n-1]
	}
	switch {
	case len(statements) == 0:
		return ""
	case len(statements) == 1:
		switch s := statements[0]; s.Kind {
		case ast.KindContinueStatement:
			return ""
		case ast.KindReturnStatement, ast.KindExpressionStatement:
			if s.Expression() == nil || trivialValue(s.Expression()) {
				return ""
			}
		}
	}
	text := ast.GetSourceFileOfNode(statements[0]).Text()
	return Squash(text[scanner.SkipTrivia(text, statements[0].Pos()):statements[len(statements)-1].End()])
}

func expressionOutcome(n *ast.Node) string {
	if trivialValue(n) {
		return ""
	}
	return Squash(source(n))
}

// Outcome returns the normalized text of what a root's question leads to, or "" when it leads to nothing or to a trivial value.
// Blocks are unwrapped and a final unlabeled break is dropped, so an if statement and a case clause doing the same work agree.
func Outcome(root *ast.Node) string {
	if root.Kind == ast.KindCaseClause {
		clauses := root.Parent.AsCaseBlock().Clauses.Nodes
		for _, clause := range clauses[indexOf(clauses, root):] {
			if statements := clause.Statements(); len(statements) > 0 {
				return statementOutcome(statements)
			}
		}
		return ""
	}
	child, p := parent(root)
	switch p.Kind {
	case ast.KindIfStatement:
		if p.AsIfStatement().Expression == child {
			return statementOutcome([]*ast.Node{p.AsIfStatement().ThenStatement})
		}
	case ast.KindConditionalExpression:
		if p.AsConditionalExpression().Condition == child {
			return expressionOutcome(p.AsConditionalExpression().WhenTrue)
		}
	case ast.KindBinaryExpression:
		if binary := p.AsBinaryExpression(); binary.OperatorToken.Kind == ast.KindAmpersandAmpersandToken && binary.Left == child {
			return expressionOutcome(binary.Right)
		}
	}
	return ""
}

func indexOf(nodes []*ast.Node, n *ast.Node) int {
	for i, m := range nodes {
		if m == n {
			return i
		}
	}
	return -1
}

func wrapper(call *ast.Node) bool {
	callee := Unwrap(call.AsCallExpression().Expression)
	if callee.Kind == ast.KindIdentifier {
		return true
	}
	return callee.Kind == ast.KindPropertyAccessExpression && Unwrap(callee.Expression()).Kind == ast.KindIdentifier && Unwrap(callee.Expression()).Text() == "React"
}

func identifier(n *ast.Node) string {
	if n != nil && n.Kind == ast.KindIdentifier {
		return n.Text()
	}
	return ""
}

func functionName(fn *ast.Node) string {
	switch fn.Kind {
	case ast.KindFunctionDeclaration:
		return identifier(fn.Name())
	case ast.KindMethodDeclaration, ast.KindGetAccessor:
		name := identifier(fn.Name())
		if class := fn.Parent; name != "" && class != nil && class.Name() != nil && class.Name().Kind == ast.KindIdentifier {
			return class.Name().Text() + "." + name
		}
		return name
	case ast.KindArrowFunction, ast.KindFunctionExpression:
		p := fn.Parent
		for p != nil && (p.Kind == ast.KindParenthesizedExpression || p.Kind == ast.KindAsExpression || p.Kind == ast.KindCallExpression && wrapper(p)) {
			p = p.Parent
		}
		if p != nil && (p.Kind == ast.KindVariableDeclaration || p.Kind == ast.KindPropertyAssignment || p.Kind == ast.KindPropertyDeclaration) {
			return identifier(p.Name())
		}
	}
	return ""
}

func predicateName(n *ast.Node) string {
	child, p := parent(n)
	switch {
	case p.Kind == ast.KindArrowFunction && p.Body() == child:
		return functionName(p)
	case p.Kind == ast.KindReturnStatement:
		block := p.Parent
		if block.Kind == ast.KindBlock && len(block.Statements()) == 1 && block.Parent != nil && ast.IsFunctionLike(block.Parent) && block.Parent.Body() == block {
			return functionName(block.Parent)
		}
	}
	return ""
}
