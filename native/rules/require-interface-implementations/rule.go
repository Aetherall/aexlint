package require_interface_implementations

import (
	"fmt"
	"os"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const required = 2

type measure struct {
	ctx rule.RuleContext
	// implemented caches the interfaces each class implements. An entry is stored before its base class is visited, so circular inheritance terminates.
	implemented map[*ast.Node]map[*ast.Symbol]bool
}

// resolve returns the merged symbol a heritage clause expression names, through import aliases and re-exports.
func (m *measure) resolve(expression *ast.Node) *ast.Symbol {
	if expression.Kind == ast.KindPropertyAccessExpression {
		expression = expression.Name()
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(expression)
	if symbol == nil {
		return nil
	}
	return m.ctx.TypeChecker.GetMergedSymbol(checker.SkipAlias(symbol, m.ctx.TypeChecker))
}

// addInterface adds symbol and every interface it extends, at any depth, to into.
func (m *measure) addInterface(symbol *ast.Symbol, into map[*ast.Symbol]bool) {
	if symbol == nil || symbol.Flags&ast.SymbolFlagsInterface == 0 || into[symbol] {
		return
	}
	into[symbol] = true
	for _, declaration := range symbol.Declarations {
		if declaration.Kind == ast.KindInterfaceDeclaration {
			for _, base := range ast.GetExtendsHeritageClauseElements(declaration) {
				m.addInterface(m.resolve(base.Expression()), into)
			}
		}
	}
}

// implementedBy returns the interfaces class implements through its own implements clause and its base classes.
func (m *measure) implementedBy(class *ast.Node) map[*ast.Symbol]bool {
	if result, ok := m.implemented[class]; ok {
		return result
	}
	result := map[*ast.Symbol]bool{}
	m.implemented[class] = result
	for _, element := range ast.GetImplementsHeritageClauseElements(class) {
		m.addInterface(m.resolve(element.Expression()), result)
	}
	base := ast.GetClassExtendsHeritageElement(class)
	if base == nil {
		return result
	}
	symbol := m.resolve(base.Expression())
	if symbol == nil || symbol.Flags&ast.SymbolFlagsClass == 0 {
		return result
	}
	for _, declaration := range symbol.Declarations {
		if ast.IsClassLike(declaration) {
			for inherited := range m.implementedBy(declaration) {
				result[inherited] = true
			}
		}
	}
	return result
}

// owned reports whether symbol is an interface contract the project declares: every declaration is in a project source
// file, and it is not merged with a class.
func (m *measure) owned(symbol *ast.Symbol) bool {
	if symbol.Flags&ast.SymbolFlagsClass != 0 {
		return false
	}
	for _, declaration := range symbol.Declarations {
		file := ast.GetSourceFileOfNode(declaration)
		if file.IsDeclarationFile || m.ctx.Program.IsSourceFileFromExternalLibrary(file) {
			return false
		}
	}
	return true
}

// relative returns fileName relative to the process's working directory, the configuration directory under aexlint-typed, falling back to the program's directory.
func (m *measure) relative(fileName string) string {
	directory, err := os.Getwd()
	if err != nil {
		directory = m.ctx.Program.GetCurrentDirectory()
	}
	directory = tspath.NormalizePath(directory)
	return tspath.GetRelativePathFromDirectory(directory, fileName, tspath.ComparePathsOptions{
		UseCaseSensitiveFileNames: m.ctx.Program.UseCaseSensitiveFileNames(),
		CurrentDirectory:          directory,
	})
}

func className(class *ast.Node) string {
	if name := class.Name(); name != nil {
		return name.Text()
	}
	return "(anonymous class)"
}

type target struct {
	declaration *ast.Node
	symbol      *ast.Symbol
}

func (m *measure) report(t target, implementers []*ast.Node) {
	name := t.declaration.Name().Text()
	if len(implementers) == 0 {
		m.ctx.ReportNode(t.declaration.Name(), rule.RuleMessage{
			Id:          "unimplementedInterface",
			Description: fmt.Sprintf("No class implements `%s`. Interfaces must be implemented by at least %d classes.", name, required),
			Help:        "Only `implements` clauses count, including those of base classes and of interfaces that extend this one. A shape that classes do not implement can be a `type` alias.",
		})
		return
	}
	class := implementers[0]
	m.ctx.ReportNode(t.declaration.Name(), rule.RuleMessage{
		Id: "singleImplementation",
		Description: fmt.Sprintf("`%s` is implemented only by `%s` (%s). Interfaces must be implemented by at least %d classes.",
			name, className(class), m.relative(ast.GetSourceFileOfNode(class).FileName()), required),
		Help: fmt.Sprintf("With one implementing class, nothing can be substituted for it: the interface and the class change together, and readers following `%s` reach the contract instead of the code.", name),
	})
}

func (m *measure) run() {
	targets := []target{}
	wanted := map[*ast.Symbol]bool{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if n.Kind == ast.KindInterfaceDeclaration {
			if symbol := m.ctx.TypeChecker.GetSymbolAtLocation(n.Name()); symbol != nil {
				symbol = m.ctx.TypeChecker.GetMergedSymbol(symbol)
				if m.owned(symbol) {
					targets = append(targets, target{n, symbol})
					wanted[symbol] = true
				}
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(m.ctx.SourceFile.AsNode())
	if len(targets) == 0 {
		return
	}
	implementers := map[*ast.Symbol][]*ast.Node{}
	for _, class := range IndexFor(m.ctx.Program).Classes() {
		for symbol := range m.implementedBy(class) {
			if wanted[symbol] {
				implementers[symbol] = append(implementers[symbol], class)
			}
		}
	}
	for _, t := range targets {
		if len(implementers[t.symbol]) < required {
			m.report(t, implementers[t.symbol])
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/require-interface-implementations",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		if options != nil {
			object, ok := options.(map[string]any)
			if !ok || len(object) != 0 {
				panic("aexlint/require-interface-implementations accepts no options; omit options or use an empty object")
			}
		}
		(&measure{ctx: ctx, implemented: map[*ast.Node]map[*ast.Symbol]bool{}}).run()
		return rule.RuleListeners{}
	},
}
