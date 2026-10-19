package no_split_result

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

const minimumReturns = 3

// class is a set of optional properties that the same returns set.
type class struct {
	names   []string
	pattern []bool
}

// classes groups the optional properties each return either sets or leaves out, by the returns that set them, and
// drops those set by every return or by none.
func classes(optional []*ast.Symbol, returns [][]model.Setting) []*class {
	found := []*class{}
	for i, property := range optional {
		pattern := make([]bool, len(returns))
		known, set, unset := true, false, false
		for r, literal := range returns {
			known = known && literal[i] != model.MaybeSet
			pattern[r] = literal[i] == model.Set
			set = set || pattern[r]
			unset = unset || literal[i] == model.Unset
		}
		if !known || !set || !unset {
			continue
		}
		index := slices.IndexFunc(found, func(c *class) bool { return slices.Equal(c.pattern, pattern) })
		if index < 0 {
			found = append(found, &class{pattern: pattern})
			index = len(found) - 1
		}
		found[index].names = append(found[index].names, property.Name)
	}
	return found
}

func exclusive(a, b *class) bool {
	for i := range a.pattern {
		if a.pattern[i] && b.pattern[i] {
			return false
		}
	}
	return true
}

func listed(names []string) string {
	quoted := []string{}
	for _, name := range names {
		quoted = append(quoted, "`"+name+"`")
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return strings.Join(quoted[:len(quoted)-1], ", ") + " and " + quoted[len(quoted)-1]
}

func check(ctx rule.RuleContext, a *model.Analyzer, fn *ast.Node) {
	optional, returns, ok := a.OptionalSettings(fn)
	if !ok || len(returns) < minimumReturns {
		return
	}
	found := classes(optional, returns)
	facts, union := []string{}, []string{}
	for _, c := range found {
		if len(c.names) > 1 {
			facts = append(facts, fmt.Sprintf("The returns set %s together or not at all.", listed(c.names)))
		}
	}
	for i, c := range found {
		for _, other := range found[i+1:] {
			if exclusive(c, other) {
				facts = append(facts, fmt.Sprintf("No return sets %s with %s.", listed(c.names), listed(other.names)))
				union = append(union, fmt.Sprintf("`{ %s } | { %s }`", strings.Join(c.names, ", "), strings.Join(other.names, ", ")))
			}
		}
	}
	if len(facts) == 0 {
		return
	}
	shape := "one optional value holding the fields set together"
	if len(union) > 0 {
		shape = "a union of what the returns build, such as " + union[0]
	}
	node := fn.Type()
	if node == nil {
		node = fn
	}
	ctx.ReportNode(node, rule.RuleMessage{
		Id:          "splitResult",
		Description: strings.Join(facts, " "),
		Help: fmt.Sprintf("The result type allows combinations no return builds, so each reader tests these fields one by one and decides what a partial combination means. With %s, the type states it once.",
			shape),
	})
}

func checkDecision(ctx rule.RuleContext, a *model.Analyzer, n *ast.Node) {
	left := a.LeftValues(n)
	if len(left) < 2 {
		return
	}
	names := []string{}
	for _, l := range left {
		names = append(names, l.Value.String())
	}
	if fabrication := a.FabricationOf(n); fabrication.Fabricated() {
		made := []string{}
		for _, h := range fabrication.Placeholders {
			made = append(made, fmt.Sprintf("appends %s to `%s` only to reserve a position `%s` fills", model.Text(h.Append.Stored, 20), h.Append.Receiver, model.Text(h.Fills[0].Node, 40)))
		}
		for _, k := range fabrication.Keys {
			made = append(made, fmt.Sprintf("stores entries of `%s` under keys from `%s`, which no lookup of `%s` uses", k.Container, model.Text(k.Origin, 30), k.Container))
		}
		ctx.ReportNode(model.DecisionCondition(n), rule.RuleMessage{
			Id:          "fabricatedOutcome",
			Description: fmt.Sprintf("This decision's outcome is split across %s, and a branch makes up what the code after it expects: it %s.", listed(names), strings.Join(made, ", and ")),
			Help:        "The branch produces a different kind of result from the others, and fabricates what the shared code needs so it can treat that result like theirs. Do that branch's work inside it, or give its result a path of its own.",
		})
		return
	}
	ctx.ReportNode(model.DecisionCondition(n), rule.RuleMessage{
		Id:          "splitOutcome",
		Description: fmt.Sprintf("This decision's outcome is split across %s: its branches change them, and the code after it reads them as separate values.", listed(names)),
		Help:        "Only the branches know which values go together, so the code after the decision relies on a pairing nothing states, and changing what one branch sets breaks it silently. Make the outcome one value, or use it inside each branch.",
	})
}

var Rule = rule.Rule{
	Name: "aexlint/no-split-result",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-split-result", options)
		a := model.NewAnalyzer(ctx)
		var walk func(*ast.Node) bool
		walk = func(n *ast.Node) bool {
			if ast.IsFunctionLike(n) && n.Body() != nil {
				check(ctx, a, n)
			}
			if n.Kind == ast.KindIfStatement || n.Kind == ast.KindSwitchStatement || n.Kind == ast.KindExpressionStatement {
				checkDecision(ctx, a, n)
			}
			n.ForEachChild(walk)
			return false
		}
		walk(ctx.SourceFile.AsNode())
		return rule.RuleListeners{}
	},
}
