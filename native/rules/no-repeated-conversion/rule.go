package no_repeated_conversion

import (
	"fmt"
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type measure struct {
	ctx           rule.RuleContext
	analyzer      *model.Analyzer
	index         *shapes
	places        model.Places
	dispatches    map[*ast.SourceFile][]*model.Decision
	constructions map[*model.Decision]map[string][]model.Construction
}

func (m *measure) dispatchesIn(file *ast.SourceFile) []*model.Decision {
	if found, ok := m.dispatches[file]; ok {
		return found
	}
	found := model.DispatchesIn(m.analyzer, file)
	m.dispatches[file] = found
	return found
}

func (m *measure) constructionsOf(d *model.Decision) map[string][]model.Construction {
	if found, ok := m.constructions[d]; ok {
		return found
	}
	found := m.analyzer.Constructions(d)
	m.constructions[d] = found
	return found
}

func identical(a, b map[string][]model.Construction) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	for value, built := range a {
		other, ok := b[value]
		if !ok || !slices.EqualFunc(built, other, func(x, y model.Construction) bool { return x.Signature == y.Signature }) {
			return false
		}
	}
	return true
}

func (m *measure) copies(d *model.Decision, built map[string][]model.Construction) []*ast.Node {
	files := []*ast.SourceFile{}
	for _, constructions := range built {
		for _, c := range constructions {
			if c.Node.Kind != ast.KindObjectLiteralExpression {
				continue
			}
			for _, literal := range m.index.shaped(model.Shape(c.Node)) {
				if file := ast.GetSourceFileOfNode(literal); !slices.Contains(files, file) {
					files = append(files, file)
				}
			}
		}
	}
	found := []*ast.Node{}
	for _, file := range files {
		for _, other := range m.dispatchesIn(file) {
			if other.Node == d.Node || !d.Field.Same(other.Field) {
				continue
			}
			if identical(built, m.constructionsOf(other)) {
				found = append(found, other.Node)
			}
		}
	}
	return found
}

func (m *measure) run() {
	for _, d := range m.dispatchesIn(m.ctx.SourceFile) {
		built := m.constructionsOf(d)
		if len(built) == 0 {
			continue
		}
		others := m.copies(d, built)
		if len(others) == 0 {
			continue
		}
		m.places.Sort(others)
		locations := []string{}
		for _, other := range others {
			locations = append(locations, m.places.Location(other))
		}
		description := fmt.Sprintf("This conversion of `%s` is written identically in another place.", model.Text(d.Node, 80))
		if len(others) > 1 {
			description = fmt.Sprintf("This conversion of `%s` is written identically in %d other places.", model.Text(d.Node, 80), len(others))
		}
		m.ctx.ReportNode(d.Node, rule.RuleMessage{
			Id:          "repeatedConversion",
			Description: description,
			Help:        fmt.Sprintf("Other copies, nearest first: %s. Write the conversion once, with the owner of the type or as a shared function, and call it from each place.", model.Listed(locations, 3)),
		})
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-repeated-conversion",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		model.ParseOptions("no-repeated-conversion", options)
		analyzer := model.NewAnalyzer(ctx, model.NewClassifier(ctx.TypeChecker))
		(&measure{ctx: ctx, analyzer: analyzer, index: shapesFor(ctx.Program), places: model.NewPlaces(ctx),
			dispatches: map[*ast.SourceFile][]*model.Decision{}, constructions: map[*model.Decision]map[string][]model.Construction{}}).run()
		return rule.RuleListeners{}
	},
}
