package no_procedural_value_dispatch

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func counted(d *model.Decision, tested int) bool {
	switch d.Root.Kind {
	case ast.KindSwitchStatement:
		return true
	case ast.KindIfStatement:
		return tested >= 2 || d.Alternative != nil
	}
	return tested >= 2
}

func report(ctx rule.RuleContext, d *model.Decision) {
	tested, procedural := 0, []string{}
	for _, a := range d.Arms {
		if !a.Tested {
			continue
		}
		tested++
		if len(a.Logic) > 0 {
			procedural = append(procedural, a.Claim.Vocabulary.Display(a.Claim.Accepted))
		}
	}
	if len(procedural) == 0 || !counted(d, tested) {
		return
	}
	ctx.ReportNode(d.Node, rule.RuleMessage{
		Id:          "proceduralBranch",
		Description: fmt.Sprintf("Branches on `%s` contain their own control flow.", model.Text(d.Node, 40)),
		Help: fmt.Sprintf("Branches with nested decisions, loops or error handling: %s. Each is a procedure for its values, written inline instead of standing on its own.",
			strings.Join(procedural, "; ")),
	})
}

var Rule = rule.Rule{
	Name: "aexlint/no-procedural-value-dispatch",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-procedural-value-dispatch", options)
		for _, d := range model.Dispatches(model.NewAnalyzer(ctx, model.NewClassifier(ctx.TypeChecker))) {
			report(ctx, d)
		}
		return rule.RuleListeners{}
	},
}
