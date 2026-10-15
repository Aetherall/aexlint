package model

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// Helper is an exported function of one parameter whose body is a single expression, named by the path its callers use.
type Helper struct {
	Function *ast.Node
	Path     string
}

// Helpers lists a program's exported one-parameter functions by the property names their bodies read on that parameter.
// It is built once per program from syntax alone and never mutated afterwards.
type Helpers struct {
	byProperty map[string][]Helper
}

var helpers = NewShared(buildHelpers)

// HelpersFor returns the program's shared helper index.
func HelpersFor(program *compiler.Program) *Helpers { return helpers.For(program) }

func exported(n *ast.Node) bool {
	return n.ModifierFlags()&ast.ModifierFlagsExport != 0
}

func function(n *ast.Node) bool {
	return n != nil && (n.Kind == ast.KindArrowFunction || n.Kind == ast.KindFunctionExpression)
}

func (index *Helpers) add(fn *ast.Node, path string) {
	if len(fn.Parameters()) != 1 || fn.Parameters()[0].Name().Kind != ast.KindIdentifier || result(fn) == nil {
		return
	}
	name := fn.Parameters()[0].Name().Text()
	seen := map[string]bool{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if n.Kind == ast.KindPropertyAccessExpression && n.Name().Kind == ast.KindIdentifier {
			if receiver := Unwrap(n.Expression()); receiver.Kind == ast.KindIdentifier && receiver.Text() == name && !seen[n.Name().Text()] {
				seen[n.Name().Text()] = true
				index.byProperty[n.Name().Text()] = append(index.byProperty[n.Name().Text()], Helper{Function: fn, Path: path})
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(result(fn))
}

func buildHelpers(program *compiler.Program) *Helpers {
	index := &Helpers{byProperty: map[string][]Helper{}}
	for _, file := range program.SourceFiles() {
		if !ProjectFile(file) {
			continue
		}
		for _, statement := range file.Statements.Nodes {
			if !exported(statement) {
				continue
			}
			switch statement.Kind {
			case ast.KindFunctionDeclaration:
				if statement.Name() != nil {
					index.add(statement, statement.Name().Text())
				}
			case ast.KindClassDeclaration:
				if statement.Name() == nil {
					continue
				}
				for _, member := range statement.Members() {
					if member.ModifierFlags()&ast.ModifierFlagsStatic == 0 || member.Name() == nil || member.Name().Kind != ast.KindIdentifier {
						continue
					}
					path := statement.Name().Text() + "." + member.Name().Text()
					switch {
					case member.Kind == ast.KindMethodDeclaration:
						index.add(member, path)
					case member.Kind == ast.KindPropertyDeclaration && function(member.Initializer()):
						index.add(member.Initializer(), path)
					}
				}
			case ast.KindVariableStatement:
				for _, declaration := range statement.AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes {
					if declaration.Name().Kind != ast.KindIdentifier {
						continue
					}
					name := declaration.Name().Text()
					initializer := declaration.Initializer()
					for initializer != nil && (initializer.Kind == ast.KindAsExpression || initializer.Kind == ast.KindSatisfiesExpression || initializer.Kind == ast.KindParenthesizedExpression) {
						initializer = initializer.Expression()
					}
					switch {
					case function(initializer):
						index.add(initializer, name)
					case initializer != nil && initializer.Kind == ast.KindObjectLiteralExpression:
						for _, property := range initializer.Properties() {
							if property.Name() == nil || property.Name().Kind != ast.KindIdentifier {
								continue
							}
							path := name + "." + property.Name().Text()
							switch {
							case property.Kind == ast.KindMethodDeclaration:
								index.add(property, path)
							case property.Kind == ast.KindPropertyAssignment && function(property.Initializer()):
								index.add(property.Initializer(), path)
							}
						}
					}
				}
			}
		}
	}
	return index
}

func (m *Analyzer) helper(h Helper) (Predicate, bool) {
	fn := h.Function
	parameter := fn.Parameters()[0].Symbol()
	e := Unwrap(result(fn))
	if e.Kind == ast.KindBinaryExpression && e.AsBinaryExpression().OperatorToken.Kind == ast.KindAmpersandAmpersandToken {
		if p, ok := PresenceTest(e.AsBinaryExpression().Left); ok && !p.Missing && p.Subject.Kind == ast.KindIdentifier && m.ctx.TypeChecker.GetSymbolAtLocation(p.Subject) == parameter {
			e = e.AsBinaryExpression().Right
		}
	}
	t, ok := m.FieldTest(e)
	if !ok || t.Receiver().Kind != ast.KindIdentifier || m.ctx.TypeChecker.GetSymbolAtLocation(t.Receiver()) != parameter {
		return Predicate{}, false
	}
	narrows := declaresPredicate(fn)
	return Predicate{Name: h.Path, Declaration: fn, Body: result(fn), Test: t, Narrows: narrows, Helper: true}, true
}

// HelpersOf returns the program's helper predicates accepting a value of type t and asking the same question as test,
// in program file order.
func (m *Analyzer) HelpersOf(t *checker.Type, test Test) []Predicate {
	found := []Predicate{}
	for _, h := range HelpersFor(m.ctx.Program).byProperty[m.Property(test).Name] {
		p, ok := m.helper(h)
		if !ok || !m.SameQuestion(p.Test, test) {
			continue
		}
		parameter := m.ctx.TypeChecker.GetTypeOfSymbol(h.Function.Parameters()[0].Symbol())
		if checker.Checker_isTypeAssignableTo(m.ctx.TypeChecker, t, parameter) {
			found = append(found, p)
		}
	}
	return m.preferred(found, test)
}
