package model

import (
	"slices"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/scanner"
)

// Form is how n is computed from source. Reads within source become $ followed by their path; literals are compared by
// value and kind, member reads and calls by member names and the forms of their parts, and object literals by their
// entries in any order. Other code is its written text with the reads of source replaced.
func (m *Analyzer) Form(n *ast.Node, source Value) string {
	n = Unwrap(n)
	if v := m.ValueOf(n); source.Named() && v.Within(source) {
		return strings.Join(append([]string{"$"}, v.Path[len(source.Path):]...), ".")
	}
	switch n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return strconv.Quote(n.Text())
	case ast.KindNumericLiteral, ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindNullKeyword:
		return written(n)
	case ast.KindPropertyAccessExpression:
		if n.Name().Kind == ast.KindIdentifier {
			return m.Form(n.Expression(), source) + "." + n.Name().Text()
		}
	case ast.KindCallExpression, ast.KindNewExpression:
		arguments := []string{}
		for _, argument := range n.Arguments() {
			arguments = append(arguments, m.Form(argument, source))
		}
		prefix := ""
		if n.Kind == ast.KindNewExpression {
			prefix = "new "
		}
		return prefix + m.Form(n.Expression(), source) + "(" + strings.Join(arguments, ",") + ")"
	case ast.KindArrayLiteralExpression:
		elements := []string{}
		for _, element := range n.Elements() {
			elements = append(elements, m.Form(element, source))
		}
		return "[" + strings.Join(elements, ",") + "]"
	case ast.KindObjectLiteralExpression:
		entries := []string{}
		for _, property := range n.Properties() {
			switch property.Kind {
			case ast.KindPropertyAssignment:
				entries = append(entries, propertyKey(property.Name())+":"+m.Form(property.Initializer(), source))
			case ast.KindShorthandPropertyAssignment:
				entries = append(entries, property.Name().Text()+":~"+property.Name().Text())
			case ast.KindSpreadAssignment:
				entries = append(entries, "..."+m.Form(property.Expression(), source))
			default:
				entries = append(entries, propertyKey(property.Name())+":~"+m.replaced(property, source))
			}
		}
		slices.Sort(entries)
		return "{" + strings.Join(entries, ",") + "}"
	}
	return "~" + m.replaced(n, source)
}

func propertyKey(name *ast.Node) string {
	switch name.Kind {
	case ast.KindIdentifier, ast.KindStringLiteral, ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return name.Text()
	}
	return written(name)
}

// replaced returns n's written text with each read of source replaced by $.
func (m *Analyzer) replaced(n *ast.Node, source Value) string {
	text := ast.GetSourceFileOfNode(n).Text()
	at := scanner.SkipTrivia(text, n.Pos())
	var b strings.Builder
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		switch c.Kind {
		case ast.KindIdentifier, ast.KindThisKeyword, ast.KindPropertyAccessExpression, ast.KindElementAccessExpression:
			if source.Named() && m.ValueOf(c).Same(source) {
				b.WriteString(text[at:scanner.SkipTrivia(text, c.Pos())])
				b.WriteString("$")
				at = c.End()
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
