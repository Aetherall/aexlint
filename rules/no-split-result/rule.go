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
			n.ForEachChild(walk)
			return false
		}
		walk(ctx.SourceFile.AsNode())
		return rule.RuleListeners{}
	},
}
