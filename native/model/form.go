package model

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/microsoft/typescript-go/shim/tspath"
)

func (m *Analyzer) Form(n *ast.Node, source Value) string {
	n = Unwrap(n)
	if source.Named() && m.ValueOf(n).Same(source) {
		return "$"
	}
	if text, ok := literalForm(n); ok {
		return text
	}
	switch n.Kind {
	case ast.KindIdentifier:
		return m.reference(n)
	case ast.KindPropertyAccessExpression:
		if n.Name().Kind == ast.KindIdentifier {
			return m.Form(n.Expression(), source) + access(n, "[") + strconv.Quote(n.Name().Text()) + "]"
		}
	case ast.KindElementAccessExpression:
		return m.Form(n.Expression(), source) + access(n, "[") + m.Form(n.AsElementAccessExpression().ArgumentExpression, source) + "]"
	case ast.KindCallExpression, ast.KindNewExpression:
		arguments := []string{}
		for _, argument := range n.Arguments() {
			arguments = append(arguments, m.Form(argument, source))
		}
		prefix := ""
		if n.Kind == ast.KindNewExpression {
			prefix = "new "
		}
		return prefix + m.Form(n.Expression(), source) + access(n, "(") + strings.Join(arguments, ",") + ")"
	case ast.KindArrayLiteralExpression:
		elements := []string{}
		for _, element := range n.Elements() {
			elements = append(elements, m.Form(element, source))
		}
		return "[" + strings.Join(elements, ",") + "]"
	case ast.KindObjectLiteralExpression:
		entries := []string{}
		keys := map[string]bool{}
		reorder := true
		for _, property := range n.Properties() {
			key := propertyKey(property.Name())
			if name := property.Name(); name != nil && name.Kind == ast.KindComputedPropertyName {
				key = "[" + m.Form(name.Expression(), source) + "]"
			}
			switch property.Kind {
			case ast.KindPropertyAssignment:
				reorder = reorder && property.Name().Kind != ast.KindComputedPropertyName && !keys[key] && m.pure(property.Initializer())
				keys[key] = true
				entries = append(entries, key+":"+m.Form(property.Initializer(), source))
			case ast.KindShorthandPropertyAssignment:
				reorder = reorder && !keys[key] && m.pure(property.Name())
				keys[key] = true
				entries = append(entries, key+":"+m.Form(property.Name(), source))
			case ast.KindSpreadAssignment:
				reorder = false
				entries = append(entries, "..."+m.Form(property.Expression(), source))
			default:
				reorder = false
				entries = append(entries, key+":~"+m.replaced(property, source))
			}
		}
		if reorder {
			slices.Sort(entries)
		}
		return "{" + strings.Join(entries, ",") + "}"
	}
	return "~" + m.replaced(n, source)
}

// SyntacticForm is Form from syntax alone, for shared indexes built without a checker: root becomes $ and other
// identifiers compare by name. Expressions with the same Form relative to root's value have the same SyntacticForm,
// unless one reaches that value through another variable naming it.
func SyntacticForm(n, root *ast.Node) string {
	n = Unwrap(n)
	if n == root {
		return "$"
	}
	if text, ok := literalForm(n); ok {
		return text
	}
	switch n.Kind {
	case ast.KindIdentifier:
		return n.Text()
	case ast.KindPropertyAccessExpression:
		if n.Name().Kind == ast.KindIdentifier {
			return SyntacticForm(n.Expression(), root) + access(n, "[") + strconv.Quote(n.Name().Text()) + "]"
		}
	case ast.KindElementAccessExpression:
		return SyntacticForm(n.Expression(), root) + access(n, "[") + SyntacticForm(n.AsElementAccessExpression().ArgumentExpression, root) + "]"
	case ast.KindCallExpression, ast.KindNewExpression:
		arguments := []string{}
		for _, argument := range n.Arguments() {
			arguments = append(arguments, SyntacticForm(argument, root))
		}
		prefix := ""
		if n.Kind == ast.KindNewExpression {
			prefix = "new "
		}
		return prefix + SyntacticForm(n.Expression(), root) + access(n, "(") + strings.Join(arguments, ",") + ")"
	}
	return "~" + written(n)
}

// access returns the operator opening an access or call, optional when it is written with ?.
func access(n *ast.Node, operator string) string {
	if n.Kind != ast.KindNewExpression && n.QuestionDotToken() != nil {
		return "?." + operator
	}
	return operator
}

// literalForm returns the form of a literal: strings quoted, numbers by value, keywords as written.
func literalForm(n *ast.Node) (string, bool) {
	switch n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return strconv.Quote(n.Text()), true
	case ast.KindNumericLiteral:
		if value, ok := numberValue(n); ok {
			return strconv.FormatFloat(value, 'g', -1, 64), true
		}
		return written(n), true
	case ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindNullKeyword:
		return written(n), true
	}
	return "", false
}

// SymbolOf returns the symbol an identifier refers to. The name of a shorthand property refers to the variable it reads.
func (m *Analyzer) SymbolOf(n *ast.Node) *ast.Symbol {
	if parent := n.Parent; parent != nil && parent.Kind == ast.KindShorthandPropertyAssignment && parent.Name() == n {
		return m.ctx.TypeChecker.GetShorthandAssignmentValueSymbol(parent)
	}
	if parent := n.Parent; parent != nil && parent.Kind == ast.KindParameter && parent.Name() == n {
		return parameterSymbol(parent)
	}
	return m.ctx.TypeChecker.GetSymbolAtLocation(n)
}

// parameterSymbol returns the symbol a function's body reads parameter through. A parameter property also declares a
// class property, which the binder gives the parameter node; its body reads the constructor's own local instead.
func parameterSymbol(parameter *ast.Node) *ast.Symbol {
	if name := parameter.Name(); ast.IsParameterPropertyDeclaration(parameter, parameter.Parent) && name.Kind == ast.KindIdentifier {
		if local := parameter.Parent.Locals()[name.Text()]; local != nil {
			return local
		}
	}
	return parameter.Symbol()
}

func (m *Analyzer) reference(n *ast.Node) string {
	symbol := m.SymbolOf(n)
	if symbol != nil && symbol.Flags&ast.SymbolFlagsAlias != 0 {
		symbol = m.ctx.TypeChecker.GetAliasedSymbol(symbol)
	}
	if symbol == nil {
		return "unresolved:" + m.declared(n)
	}
	if len(symbol.Declarations) == 0 {
		return "symbol:" + symbol.Name
	}
	return "symbol:" + symbol.Name + "@" + m.declared(symbol.Declarations[0])
}

// declared identifies a node by its file, relative to the program's directory, and position, so references compare
// alike across invocations and machines.
func (m *Analyzer) declared(n *ast.Node) string {
	directory := m.ctx.Program.GetCurrentDirectory()
	file := tspath.GetRelativePathFromDirectory(directory, ast.GetSourceFileOfNode(n).FileName(), tspath.ComparePathsOptions{
		UseCaseSensitiveFileNames: m.ctx.Program.UseCaseSensitiveFileNames(),
		CurrentDirectory:          directory,
	})
	return fmt.Sprintf("%s:%d", file, n.Pos())
}

func propertyKey(name *ast.Node) string {
	if name == nil {
		return ""
	}
	switch name.Kind {
	case ast.KindIdentifier, ast.KindStringLiteral, ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return name.Text()
	}
	return written(name)
}

func (m *Analyzer) replaced(n *ast.Node, source Value) string {
	file := ast.GetSourceFileOfNode(n)
	text := file.Text()
	at := scanner.SkipTrivia(text, n.Pos())
	var b strings.Builder
	replace := func(c *ast.Node, value string) {
		b.WriteString(text[at:scanner.SkipTrivia(text, c.Pos())])
		b.WriteString(value)
		at = c.End()
	}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		switch c.Kind {
		case ast.KindIdentifier, ast.KindThisKeyword, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
			if source.Named() && m.ValueOf(c).Same(source) {
				replace(c, "$")
				return false
			}
		}
		if c.Kind == ast.KindIdentifier {
			p := c.Parent
			if p != nil && p.Name() == c && (p.Kind == ast.KindPropertyAccessExpression || p.Kind == ast.KindPropertyAssignment || ast.IsFunctionLike(p) || p.Kind == ast.KindParameter || p.Kind == ast.KindVariableDeclaration || p.Kind == ast.KindMethodDeclaration) {
				return false
			}
			symbol := m.ctx.TypeChecker.GetSymbolAtLocation(c)
			if symbol != nil && !slices.ContainsFunc(symbol.Declarations, func(d *ast.Node) bool {
				return ast.GetSourceFileOfNode(d) == file && d.Pos() >= n.Pos() && d.End() <= n.End()
			}) {
				replace(c, m.reference(c))
				return false
			}
		}
		c.ForEachChild(walk)
		return false
	}
	walk(n)
	b.WriteString(text[at:n.End()])
	return Squash(b.String())
}
