package no_forgeable_validated_result

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type measure struct {
	ctx rule.RuleContext
	a   *model.Analyzer
}

func (m *measure) line(node *ast.Node) int {
	file := ast.GetSourceFileOfNode(node)
	return scanner.GetECMALineOfPosition(file, utils.TrimNodeTextRange(file, node).Pos()) + 1
}

func displayName(fn *ast.Node) string {
	if fn.Kind == ast.KindConstructor {
		if class := fn.Parent; class != nil && class.Name() != nil {
			return "constructor of `" + class.Name().Text() + "`"
		}
		return "constructor"
	}
	if name := fn.Name(); name != nil {
		return "`" + name.Text() + "`"
	}
	return "(anonymous function)"
}

func (m *measure) evidence(found *model.Check) string {
	if found.Callee == nil {
		return fmt.Sprintf("throw at line %d.", m.line(found.Node))
	}
	return fmt.Sprintf("call to %s (%s) at line %d.", displayName(found.Callee),
		model.NewPlaces(m.ctx).Relative(ast.GetSourceFileOfNode(found.Callee).FileName()), m.line(found.Node))
}

// operation returns the name of a checked operation and the node to report. Constructors are not operations: their
// result is a class instance, which is owned.
func (m *measure) operation(n *ast.Node) (string, *ast.Node, bool) {
	switch n.Kind {
	case ast.KindFunctionDeclaration, ast.KindMethodDeclaration:
		name := n.Name()
		if name == nil || name.Kind == ast.KindComputedPropertyName {
			return "", nil, false
		}
		return name.Text(), name, true
	case ast.KindFunctionExpression, ast.KindArrowFunction:
		parent := n.Parent
		switch parent.Kind {
		case ast.KindVariableDeclaration, ast.KindPropertyAssignment, ast.KindPropertyDeclaration:
			name := parent.Name()
			if parent.Initializer() != n || name == nil || (name.Kind != ast.KindIdentifier && name.Kind != ast.KindPrivateIdentifier) {
				return "", nil, false
			}
			return name.Text(), name, true
		}
	}
	return "", nil, false
}

func (m *measure) check(n *ast.Node) {
	name, node, ok := m.operation(n)
	if !ok || n.Body() == nil {
		return
	}
	t := m.a.Result(n)
	if t == nil || !m.a.Forgeable(t) || !m.a.Owned(t) || !m.a.Stateful(t) {
		return
	}
	f := m.a.Flow(n)
	if f == nil {
		return
	}
	var found *model.Check
	for _, candidate := range f.Checks {
		if len(candidate.Checked) > 0 && model.Overlapping(candidate.Checked, f.Results) {
			found = candidate
			break
		}
	}
	if found == nil {
		return
	}
	typeName := m.ctx.TypeChecker.TypeToString(t)
	label := "`" + name + "`"
	message := rule.RuleMessage{
		Id:          "forgeableValidatedResult",
		Description: fmt.Sprintf("%s rejects some inputs, then returns `%s`, which other code can produce without that check.", label, typeName),
		Help: fmt.Sprintf("Code that receives a value of type `%s` cannot tell whether it passed %s, and each other place that builds or changes such a value must repeat the check or skip it. Rejection: %s",
			typeName, label, m.evidence(found)),
	}
	m.ctx.ReportNode(node, message)
}

func (m *measure) run() {
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		m.check(n)
		n.ForEachChild(walk)
		return false
	}
	walk(m.ctx.SourceFile.AsNode())
}

var Rule = rule.Rule{
	Name: "aexlint/no-forgeable-validated-result",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-forgeable-validated-result", options)
		(&measure{ctx: ctx, a: model.NewAnalyzer(ctx, model.NewClassifier(ctx.TypeChecker))}).run()
		return rule.RuleListeners{}
	},
}
