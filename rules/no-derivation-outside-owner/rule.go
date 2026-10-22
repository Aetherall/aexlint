package no_derivation_outside_owner

import (
	"fmt"
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func check(ctx rule.RuleContext, a *model.Analyzer, places model.Places, limit int, n *ast.Node) {
	d, ok := a.DerivationOf(n)
	if !ok {
		return
	}
	readers := []*ast.Node{d.Reader}
	elsewhere := []string{}
	for _, other := range a.DerivationsLike(d) {
		if slices.Contains(readers, other.Reader) {
			continue
		}
		readers = append(readers, other.Reader)
		elsewhere = append(elsewhere, places.Location(other.Node))
	}
	if len(readers) <= limit {
		return
	}
	callee, property, owner := model.Text(n.Expression(), 30), d.Property.Name().Text(), d.Owner.Name
	ctx.ReportNode(n, rule.RuleMessage{
		Id:          "repeatedDerivation",
		Description: fmt.Sprintf("`%s` of `%s` goes through `%s` here and in %d other functions outside the type's owner (%s): each reader derives the value itself.", property, owner, callee, len(readers)-1, model.Listed(elsewhere, 3)),
		Help:        fmt.Sprintf("The readers agree only as long as every copy computes it the same way, and nothing ties the copies together. Derive it once, as a member of `%s` or of a type holding it, or a function beside it, so readers ask for the value instead of computing it.", owner),
	})
}

var Rule = rule.Rule{
	Name: "aexlint/no-derivation-outside-owner",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		limit := model.ParseLimit("no-derivation-outside-owner", options, 2, 1)
		a, places := model.NewAnalyzer(ctx), model.NewPlaces(ctx)
		return rule.RuleListeners{ast.KindCallExpression: func(n *ast.Node) { check(ctx, a, places, limit, n) }}
	},
}
