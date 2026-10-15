package max_projection_spread

import (
	"fmt"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const defaultMax = 2

func parseOptions(options any) int {
	return model.ParseMax("max-projection-spread", options, defaultMax)
}

// derivation is a property of an object literal that derives its value from a member chain of an owned value.
type derivation struct {
	property *ast.Node
	key      string
	root     *ast.Symbol
	owner    *ast.Symbol
	// identity is equal for derivations that restate the same derivation: same key, owner type, and form.
	identity string
}

type measure struct {
	ctx         rule.RuleContext
	analyzer    *model.Analyzer
	derivations map[*ast.Node]*derivation
	projections map[*ast.Node]bool
}

// derive returns the derivation a property assignment makes, or nil.
func (m *measure) derive(property *ast.Node) *derivation {
	if d, ok := m.derivations[property]; ok {
		return d
	}
	m.derivations[property] = nil
	key, ok := KeyOf(property)
	if !ok {
		return nil
	}
	chain, ok := ChainOf(property.Initializer())
	if !ok {
		return nil
	}
	root := m.analyzer.Variable(chain.Root)
	owner, t := m.analyzer.ObjectOwner(chain.Root)
	if root == nil || owner == nil {
		return nil
	}
	identity := fmt.Sprintf("%s|%p|%s", key, t, m.analyzer.Form(property.Initializer(), m.analyzer.ValueOf(chain.Root)))
	d := &derivation{property: property, key: key, root: root, owner: owner, identity: identity}
	m.derivations[property] = d
	return d
}

// projected reports whether a derivation belongs to a projection: its object literal has another derivation from the
// same variable.
func (m *measure) projected(d *derivation) bool {
	literal := d.property.Parent
	if done, ok := m.projections[d.property]; ok {
		return done
	}
	count := 0
	for _, property := range literal.AsObjectLiteralExpression().Properties.Nodes {
		if other := m.derive(property); other != nil && other.root == d.root && other.owner == d.owner {
			count++
		}
	}
	m.projections[d.property] = count >= 2
	return count >= 2
}

// chainText returns a derivation's value as written, without its root.
func (m *measure) chainText(d *derivation) string {
	value := skipOuter(d.property.Initializer())
	text := m.ctx.SourceFile.Text()[utils.TrimNodeTextRange(m.ctx.SourceFile, value).Pos():value.End()]
	chain, _ := ChainOf(value)
	text = strings.TrimPrefix(text, chain.Root.Text())
	text = strings.TrimPrefix(text, "!")
	text = strings.TrimPrefix(text, "?")
	return strings.TrimPrefix(text, ".")
}

type spread struct {
	d     *derivation
	files map[*ast.SourceFile]bool
}

func (m *measure) run(limit int) {
	index := IndexFor(m.ctx.Program)
	literals := []*ast.Node{}
	local := map[*ast.Node][]*derivation{}
	for _, property := range index.File(m.ctx.SourceFile) {
		d := m.derive(property)
		if d == nil || !m.projected(d) {
			continue
		}
		literal := property.Parent
		if local[literal] == nil {
			literals = append(literals, literal)
		}
		local[literal] = append(local[literal], d)
	}
	counted := map[string]map[*ast.SourceFile]bool{}
	for _, literal := range literals {
		for _, d := range local[literal] {
			if counted[d.identity] != nil {
				continue
			}
			files := map[*ast.SourceFile]bool{}
			for _, candidate := range index.Keyed(d.property) {
				if other := m.derive(candidate); other != nil && other.identity == d.identity && m.projected(other) {
					files[ast.GetSourceFileOfNode(candidate)] = true
				}
			}
			counted[d.identity] = files
		}
	}
	for _, literal := range literals {
		spreads := []spread{}
		for _, d := range local[literal] {
			if files := counted[d.identity]; len(files) > limit {
				spreads = append(spreads, spread{d: d, files: files})
			}
		}
		if len(spreads) == 0 {
			continue
		}
		m.report(spreads, limit)
	}
}

func (m *measure) report(spreads []spread, limit int) {
	first := spreads[0].d
	slices.SortStableFunc(spreads, func(a, b spread) int {
		if byFiles := len(b.files) - len(a.files); byFiles != 0 {
			return byFiles
		}
		return strings.Compare(a.d.key, b.d.key)
	})
	counted := []string{}
	files := map[*ast.SourceFile]bool{}
	for _, s := range spreads {
		counted = append(counted, fmt.Sprintf("`%s` in %d files", s.d.key, len(s.files)))
		for file := range s.files {
			files[file] = true
		}
	}
	slices.SortStableFunc(spreads, func(a, b spread) int {
		return strings.Compare(a.d.key, b.d.key)
	})
	described := []string{}
	for _, s := range spreads {
		described = append(described, fmt.Sprintf("`%s` (`%s`)", s.d.key, m.chainText(s.d)))
	}
	count := "1 derivation"
	if len(spreads) > 1 {
		count = fmt.Sprintf("%d derivations", len(spreads))
	}
	limited := fmt.Sprintf("%d files", limit)
	if limit == 1 {
		limited = "1 file"
	}
	m.ctx.ReportNode(first.property.Name(), rule.RuleMessage{
		Id: "spreadProjection",
		Description: fmt.Sprintf("This projection of `%s` restates %s, each found in more than %s: %s.",
			first.owner.Name, count, limited, model.Listed(described, 3)),
		Help: fmt.Sprintf("Changing one of these derivations requires finding each file by hand. Restated in: %s. Other files, nearest first: %s.",
			model.Listed(counted, 3), model.Listed(model.NewPlaces(m.ctx).Nearest(files), 3)),
	})
}

var Rule = rule.Rule{
	Name: "aexlint/max-projection-spread",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		limit := parseOptions(options)
		(&measure{ctx: ctx, analyzer: model.NewAnalyzer(ctx), derivations: map[*ast.Node]*derivation{}, projections: map[*ast.Node]bool{}}).run(limit)
		return rule.RuleListeners{}
	},
}
