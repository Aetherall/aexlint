package no_external_variant_behavior

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type measure struct {
	ctx   rule.RuleContext
	model *model.Analyzer
}

func (m *measure) report(d *model.Decision) {
	arms := []model.Branch{}
	for _, b := range d.Arms {
		if len(b.Body) > 0 {
			arms = append(arms, b)
		}
	}
	if len(arms) < 2 || !slices.ContainsFunc(arms, func(a model.Branch) bool { return a.Tested }) || d.Owner == nil || d.Inside {
		return
	}
	subject, owner := d.Node, d.Owner
	fields := []string{}
	destinations := []*ast.Symbol{}
	for _, a := range arms {
		for _, result := range m.model.Built(d, a) {
			for _, field := range m.model.ConstructionFields(result, d.Object, d.Tag) {
				if !slices.Contains(fields, field) {
					fields = append(fields, field)
				}
			}
			if destination := m.model.Destination(result); destination != nil && destination != owner && !slices.Contains(destinations, destination) {
				destinations = append(destinations, destination)
			}
		}
	}
	if len(fields) == 0 || slices.ContainsFunc(destinations, func(destination *ast.Symbol) bool { return m.model.InsideOwner(subject, destination) }) {
		return
	}
	slices.Sort(fields)
	dispatch := model.Text(subject, 60)
	if len(destinations) == 0 {
		m.ctx.ReportNode(subject, rule.RuleMessage{
			Id:          "externalVariantBehavior",
			Description: fmt.Sprintf("This code reconstructs behavior for `%s` outside its owner.", owner.Name),
			Help:        fmt.Sprintf("Branches on `%s` construct results using %s. Declare the operation %s instead of unpacking its variants in a caller.", dispatch, strings.Join(fields, ", "), placement(owner)),
		})
		return
	}
	m.ctx.ReportNode(subject, rule.RuleMessage{
		Id:          "externalVariantConversion",
		Description: "This code does a semantic transformation that weakens ownership.",
		Help:        "Find a suitable owner that fits the model or reconsider the approach.",
	})
}

func placement(owner *ast.Symbol) string {
	if owner.Flags&ast.SymbolFlagsValue == 0 {
		return fmt.Sprintf("in the module declaring `%s`", owner.Name)
	}
	return fmt.Sprintf("on `%s`", owner.Name)
}

var Rule = rule.Rule{
	Name: "aexlint/no-external-variant-behavior",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-external-variant-behavior", options)
		m := &measure{ctx: ctx, model: model.NewAnalyzer(ctx)}
		for _, d := range model.Dispatches(m.model) {
			m.report(d)
		}
		return rule.RuleListeners{}
	},
}
