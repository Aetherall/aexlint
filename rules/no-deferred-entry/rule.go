package no_deferred_entry

import (
	"fmt"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func line(node *ast.Node) int {
	file := ast.GetSourceFileOfNode(node)
	return scanner.GetECMALineOfPosition(file, utils.TrimNodeTextRange(file, node).Pos()) + 1
}

func check(ctx rule.RuleContext, a *model.Analyzer, fn *ast.Node) {
	for _, b := range a.Buffers(fn) {
		for _, e := range b.Entries {
			if !e.Settled || len(e.Ordered) > 0 || len(e.Exposed) > 0 {
				continue
			}
			ctx.ReportNode(e.Entry.Node, rule.RuleMessage{
				Id:          "deferredEntry",
				Description: fmt.Sprintf("`%s` only holds this entry for the loop over it on line %d: the entry is complete when stored, and nothing else reaches it before that loop takes it.", b.Contents.Container, line(b.Drain)),
				Help:        "The collection holds work back without changing it, so the reader has to find the loop to learn what happens to each entry, and every entry stays alive until then. Do the loop's work where the entry is made, or yield the entry from a generator the loop consumes.",
			})
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-deferred-entry",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-deferred-entry", options)
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
