package no_default_outside_owner

import (
	"fmt"
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

func check(ctx rule.RuleContext, a *model.Analyzer, places model.Places, limit int, n *ast.Node) {
	d, ok := a.DefaultOf(n)
	if !ok || d.Inside {
		return
	}
	name, owner := d.Name, d.Owner.Name
	help := fmt.Sprintf("What an absent `%s` stands for belongs to `%s`. Resolve it there once, with a member or a function beside the type, or make the property required and set it where values are made, so its readers cannot disagree.", name, owner)
	others := []string{}
	same := []string{}
	repeats := 0
	for _, other := range a.DefaultsLike(d) {
		switch {
		case other.Node == d.Node:
		case !a.SameDefault(other, d):
			if text := fmt.Sprintf("%s at %s", model.Text(other.Value, 20), places.Location(other.Node)); !slices.Contains(others, text) {
				others = append(others, text)
			}
		case !other.Inside:
			repeats++
			if location := places.Location(other.Node); !slices.Contains(same, location) {
				same = append(same, location)
			}
		}
	}
	if len(others) > 0 {
		ctx.ReportNode(n, rule.RuleMessage{
			Id:          "conflictingDefault",
			Description: fmt.Sprintf("An absent `%s` of `%s` stands for %s here, but for %s: its readers disagree on what absence means.", name, owner, model.Text(d.Value, 20), model.Listed(others, 3)),
			Help:        help,
		})
		return
	}
	if repeats+1 > limit {
		where := "here"
		if len(same) > 0 {
			where = "here and at " + model.Listed(same, 3)
		}
		ctx.ReportNode(n, rule.RuleMessage{
			Id:          "repeatedDefault",
			Description: fmt.Sprintf("An absent `%s` of `%s` is taken for %s %s, outside the type's owner.", name, owner, model.Text(d.Value, 20), where),
			Help:        help,
		})
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-default-outside-owner",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		limit := model.ParseLimit("no-default-outside-owner", options, 1, 0)
		a, places := model.NewAnalyzer(ctx), model.NewPlaces(ctx)
		visit := func(n *ast.Node) { check(ctx, a, places, limit, n) }
		return rule.RuleListeners{ast.KindBinaryExpression: visit, ast.KindBindingElement: visit}
	},
}
