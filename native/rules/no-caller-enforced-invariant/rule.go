package no_caller_enforced_invariant

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/rules/no-forgeable-validated-result"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

type measure struct {
	ctx      rule.RuleContext
	analyzer *no_forgeable_validated_result.Analyzer
}

func (m *measure) line(node *ast.Node) int {
	file := ast.GetSourceFileOfNode(node)
	return scanner.GetECMALineOfPosition(file, utils.TrimNodeTextRange(file, node).Pos()) + 1
}

func (m *measure) text(node *ast.Node) string {
	text := m.ctx.SourceFile.Text()[utils.TrimNodeTextRange(m.ctx.SourceFile, node).Pos():node.End()]
	if len(text) > 40 {
		return text[:37] + "..."
	}
	return text
}

func className(class *ast.Node) string {
	if name := class.Name(); name != nil {
		return name.Text()
	}
	return "(anonymous class)"
}

func calleeName(class *ast.Node, declaration *ast.Node) string {
	if declaration.Kind == ast.KindConstructor {
		return "new " + className(class)
	}
	return className(class) + "." + declaration.Name().Text()
}

// discarded reports whether a construction or static factory's result is unused: nothing receives the instance, as
// when a factory is called only to see whether it throws.
func discarded(call *ast.Node, declaration *ast.Node) bool {
	if declaration.Kind != ast.KindConstructor && !ast.HasSyntacticModifier(declaration, ast.ModifierFlagsStatic) {
		return false
	}
	n := call
	for n.Parent != nil && (n.Parent.Kind == ast.KindAwaitExpression || n.Parent.Kind == ast.KindParenthesizedExpression) {
		n = n.Parent
	}
	return n.Parent != nil && n.Parent.Kind == ast.KindExpressionStatement
}

func statements(n *ast.Node) []*ast.Node {
	switch n.Kind {
	case ast.KindBlock:
		return n.AsBlock().Statements.Nodes
	case ast.KindCaseClause, ast.KindDefaultClause:
		return n.AsCaseOrDefaultClause().Statements.Nodes
	case ast.KindSourceFile:
		return n.AsSourceFile().Statements.Nodes
	case ast.KindModuleBlock:
		return n.AsModuleBlock().Statements.Nodes
	}
	return nil
}

func encloses(outer *ast.Node, inner *ast.Node) bool {
	return outer.Pos() <= inner.Pos() && inner.End() <= outer.End()
}

// exits reports whether a statement list ends by leaving it, through a statement that does not contain check: the
// check's own throw ending its guard does not count.
func exits(list []*ast.Node, check *ast.Node) bool {
	if len(list) == 0 {
		return false
	}
	last := list[len(list)-1]
	switch last.Kind {
	case ast.KindReturnStatement, ast.KindThrowStatement, ast.KindBreakStatement, ast.KindContinueStatement:
		return !encloses(last, check)
	}
	return false
}

// precedes reports whether a check can run before a call on the way to it: the statement holding the check comes
// before the call in a statement list enclosing both, and no statement list between the check and that statement ends
// by leaving it, as a branch that returns does. A check in another branch of the same if, or in a branch that returns
// before the call, does not precede it. A check nested in conditions (applied only for some actors or values) still
// precedes a call after them.
func precedes(check *ast.Node, call *ast.Node) bool {
	for list := call.Parent; list != nil; list = list.Parent {
		if statements(list) == nil || !encloses(list, check) {
			continue
		}
		statement := check
		for statement.Parent != nil && statement.Parent != list {
			statement = statement.Parent
		}
		if statement.End() > call.Pos() {
			return false
		}
		for n := check.Parent; n != nil && n != list; n = n.Parent {
			if exits(statements(n), check) {
				return false
			}
		}
		return true
	}
	return false
}

// ownReceiver reports whether a call invokes a member of the calling object itself, through `this` or `super`, which is
// the owner's own code even when the class's base cannot be resolved, as with a mixin (`extends Mixin(Base)`).
func ownReceiver(call *ast.Node) bool {
	if call.Kind != ast.KindCallExpression {
		return false
	}
	callee := call.Expression()
	if callee.Kind != ast.KindPropertyAccessExpression && callee.Kind != ast.KindElementAccessExpression {
		return false
	}
	receiver := callee.Expression()
	return receiver.Kind == ast.KindThisKeyword || receiver.Kind == ast.KindSuperKeyword
}

// within reports whether caller is class or derives from it, at any depth: the check is then already made by the
// class's own code, through inheritance.
func (m *measure) within(caller *ast.Node, class *ast.Node) bool {
	seen := map[*ast.Node]bool{}
	for caller != nil && !seen[caller] {
		if caller == class {
			return true
		}
		seen[caller] = true
		base := ast.GetClassExtendsHeritageElement(caller)
		if base == nil {
			return false
		}
		symbol := m.ctx.TypeChecker.GetSymbolAtLocation(base.Expression())
		if symbol == nil {
			return false
		}
		symbol = checker.SkipAlias(symbol, m.ctx.TypeChecker)
		caller = nil
		for _, declaration := range symbol.Declarations {
			if ast.IsClassLike(declaration) {
				caller = declaration
			}
		}
	}
	return false
}

func (m *measure) check(fn *ast.Node) {
	caller := ast.GetContainingClass(fn)
	for _, call := range m.analyzer.Calls(fn) {
		if len(call.Checks) == 0 {
			continue
		}
		class, declaration := m.analyzer.Owner(call.Node)
		if class == nil || m.within(caller, class) || ownReceiver(call.Node) || discarded(call.Node, declaration) {
			continue
		}
		handed := m.handed(call, declaration)
		for index, argument := range call.Node.Arguments() {
			if index >= len(call.Arguments) || !m.analyzer.StoresParameter(declaration, index) || m.analyzer.ChecksParameter(declaration, index) {
				continue
			}
			unchanged := no_forgeable_validated_result.Exact(call.Arguments[index])
			for _, check := range call.Checks {
				if !no_forgeable_validated_result.Definite(check.Read, unchanged) || !precedes(check.Node, call.Node) {
					continue
				}
				m.report(argument, check, class, declaration, handed)
				break
			}
		}
	}
}

// handed returns the data a hand-off gives the class: the arguments at the parameters its callee stores.
func (m *measure) handed(call no_forgeable_validated_result.Call, declaration *ast.Node) [][]no_forgeable_validated_result.Path {
	handed := [][]no_forgeable_validated_result.Path{}
	for index := range call.Arguments {
		if m.analyzer.StoresParameter(declaration, index) {
			handed = append(handed, call.Arguments[index])
		}
	}
	return handed
}

// uncovered returns the first datum a check reads that is not handed to the class, or false when every one is, so the
// class could make the check itself.
func uncovered(check no_forgeable_validated_result.Check, handed [][]no_forgeable_validated_result.Path) (no_forgeable_validated_result.Path, bool) {
	for i := range check.Read {
		found := false
		for _, argument := range handed {
			found = found || no_forgeable_validated_result.Definite(check.Read[i:i+1], argument)
		}
		if !found {
			return check.Read[i], true
		}
	}
	return no_forgeable_validated_result.Path{}, false
}

func (m *measure) report(argument *ast.Node, check no_forgeable_validated_result.Check, class *ast.Node, declaration *ast.Node, handed [][]no_forgeable_validated_result.Path) {
	callee := calleeName(class, declaration)
	name := className(class)
	fact := fmt.Sprintf("The check reads only data `%s` receives.", callee)
	if missing, ok := uncovered(check, handed); ok {
		fact = fmt.Sprintf("The check also reads %s, which `%s` does not receive.", describe(missing), callee)
	}
	m.ctx.ReportNode(argument, rule.RuleMessage{
		Id: "checkedBeforeHandOff",
		Description: fmt.Sprintf("`%s` is checked at line %d, then handed to `%s`, which stores it without that check.",
			m.text(argument), m.line(check.Node), callee),
		Help: fmt.Sprintf("The rule this check enforces lives with this caller rather than with the data, so other code that creates or changes a `%s` is not held to it. %s This often means `%s`, or a value type it takes, should own the rule; that its parameters carry information the caller must keep consistent; or that a domain operation is missing. If the rule only applies to this operation, it belongs here.",
			name, fact, name),
	})
}

// describe names a datum in a message: the path as written, in backticks.
func describe(p no_forgeable_validated_result.Path) string {
	text := p.String()
	if rest, ok := strings.CutPrefix(text, "part of "); ok {
		return "part of `" + rest + "`"
	}
	return "`" + text + "`"
}

func (m *measure) run() {
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) && n.Body() != nil {
			m.check(n)
		}
		n.ForEachChild(walk)
		return false
	}
	walk(m.ctx.SourceFile.AsNode())
}

var Rule = rule.Rule{
	Name: "aexlint/no-caller-enforced-invariant",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		if options != nil {
			object, ok := options.(map[string]any)
			if !ok || len(object) != 0 {
				panic("aexlint/no-caller-enforced-invariant accepts no options; omit options or use an empty object")
			}
		}
		(&measure{ctx: ctx, analyzer: no_forgeable_validated_result.NewAnalyzer(ctx)}).run()
		return rule.RuleListeners{}
	},
}
