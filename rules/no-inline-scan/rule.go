package no_inline_scan

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func quoted(names []string) string {
	return "`" + strings.Join(names, "`, `") + "`"
}

func names(symbols []*ast.Symbol) []string {
	found := []string{}
	for _, symbol := range symbols {
		found = append(found, symbol.Name)
	}
	return found
}

func check(ctx rule.RuleContext, a *model.Analyzer, fn *ast.Node) {
	for _, s := range a.ScansIn(fn) {
		carried := names(s.Carried)
		receivers := []string{}
		for _, e := range s.Emits {
			if name := e.Receiver.String(); !strings.Contains(strings.Join(receivers, "\x00"), name) {
				receivers = append(receivers, name)
			}
		}
		if len(carried) == 0 {
			continue
		}
		source := "it"
		switch {
		case len(s.Computed) < len(s.Carried):
			source = quoted(names(s.Computed))
		case len(carried) > 1:
			source = "them"
		}
		help := fmt.Sprintf("Each value depends on what earlier passes did, so a reader runs the loop in their head to know it, down to whether it emits before or after advancing %s. A generator owning %s and yielding those values names the sequence and keeps its state with it.", quoted(carried), quoted(carried))
		if len(s.Counters) == len(s.Carried) {
			help = fmt.Sprintf("%s only counts passes, yet a reader checks that every pass reaches its increment, and whether values see it before or after. The iteration can own it: a `for` incrementor, `entries()`, or an `enumerate` generator for other iterables.", quoted(carried))
		}
		if len(s.Other) > 0 {
			other := []string{}
			for _, n := range s.Other {
				other = append(other, "`"+model.Text(n, 30)+"`")
			}
			help += fmt.Sprintf(" The loop also does other work (%s), which would then only see finished values.", strings.Join(other, ", "))
		}
		ctx.ReportNode(s.Emits[0].Node, rule.RuleMessage{
			Id:          "inlineScan",
			Description: fmt.Sprintf("This loop carries %s from one pass to the next and puts values computed from %s into %s: a sequence with its own state, written inline.", quoted(carried), source, quoted(receivers)),
			Help:        help,
		})
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-inline-scan",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-inline-scan", options)
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
