package no_caller_enforced_invariant

import (
	"fmt"
	"strings"

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

func (m *measure) check(fn *ast.Node) {
	caller, f := ast.GetContainingClass(fn), m.a.Flow(fn)
	if f == nil {
		return
	}
	for _, call := range f.Calls {
		if len(call.Rejections) == 0 || call.Thrown {
			continue
		}
		class, declaration := m.a.Member(call.Node)
		if class == nil || m.a.Extends(caller, class) || model.OwnReceiver(call.Node) || discarded(call.Node, declaration) {
			continue
		}
		handed := m.handed(call, declaration)
		for index, argument := range call.Node.Arguments() {
			if index >= len(call.Arguments) || !m.a.Stores(declaration, index) || m.checks(declaration, index) {
				continue
			}
			unchanged := model.Exact(call.Arguments[index])
			for _, check := range call.Rejections {
				if !model.Definite(check.Read, unchanged) || !m.a.Precedes(check.Node, call.Node) || m.a.Exclusive(check.Node, call.Node) {
					continue
				}
				m.report(argument, check, class, declaration, handed)
				break
			}
		}
	}
}

// checks reports whether a member checks what it receives at index, directly or through calls.
func (m *measure) checks(member *ast.Node, index int) bool {
	f := m.a.Flow(member)
	return f != nil && len(f.Read[index]) > 0
}

// handed returns the data a hand-off gives the class: the arguments at the parameters its callee stores.
func (m *measure) handed(call model.Call, declaration *ast.Node) [][]model.Value {
	handed := [][]model.Value{}
	for index := range call.Arguments {
		if m.a.Stores(declaration, index) {
			handed = append(handed, call.Arguments[index])
		}
	}
	return handed
}

// uncovered returns the first datum a check reads that is not handed to the class, or false when every one is, so the
// class could make the check itself.
func uncovered(check *model.Rejection, handed [][]model.Value) (model.Value, bool) {
	for i := range check.Read {
		found := false
		for _, argument := range handed {
			found = found || model.Definite(check.Read[i:i+1], argument)
		}
		if !found {
			return check.Read[i], true
		}
	}
	return model.Value{}, false
}

func (m *measure) report(argument *ast.Node, check *model.Rejection, class *ast.Node, declaration *ast.Node, handed [][]model.Value) {
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
func describe(p model.Value) string {
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
		model.ParseOptions("no-caller-enforced-invariant", options)
		(&measure{ctx: ctx, a: model.NewAnalyzer(ctx)}).run()
		return rule.RuleListeners{}
	},
}
