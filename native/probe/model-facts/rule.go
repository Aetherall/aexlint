package model_facts

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

var kinds = []string{"site", "test", "decision", "function", "gate", "conflict", "guard", "entry", "boundary", "flow", "contract"}

// selected reads the fact kinds to report from {"facts": [...]}, defaulting to all of them.
func selected(options any) map[string]bool {
	chosen := map[string]bool{}
	if object, ok := options.(map[string]any); ok {
		switch facts := object["facts"].(type) {
		case []string:
			for _, kind := range facts {
				chosen[kind] = true
			}
		case []any:
			for _, kind := range facts {
				if text, ok := kind.(string); ok {
					chosen[text] = true
				}
			}
		}
	}
	for kind := range chosen {
		if !slices.Contains(kinds, kind) {
			panic(fmt.Sprintf("model-facts: unknown fact kind %q", kind))
		}
	}
	if len(chosen) == 0 {
		for _, kind := range kinds {
			chosen[kind] = true
		}
	}
	return chosen
}

type probe struct {
	ctx    rule.RuleContext
	a      *model.Analyzer
	places model.Places
	kinds  map[string]bool
}

func (p *probe) report(kind string, n *ast.Node, description string) {
	if p.kinds[kind] {
		p.ctx.ReportNode(n, rule.RuleMessage{Id: kind, Description: clean(description)})
	}
}

func (p *probe) reportHead(kind string, fn *ast.Node, description string) {
	if !p.kinds[kind] {
		return
	}
	if name := fn.Name(); name != nil {
		p.report(kind, name, description)
		return
	}
	start := scanner.SkipTrivia(p.ctx.SourceFile.Text(), fn.Pos())
	end := fn.End()
	if body := fn.Body(); body != nil {
		end = max(start+1, body.Pos())
	}
	p.ctx.ReportRange(core.NewTextRange(start, end), rule.RuleMessage{Id: kind, Description: clean(description)})
}

func clean(text string) string {
	return strings.ReplaceAll(text, ast.InternalSymbolNamePrefix, "#")
}

func claim(c model.Claim) string {
	accepted := "nothing"
	if len(c.Accepted) > 0 {
		accepted = c.Vocabulary.Display(c.Accepted)
	}
	return fmt.Sprintf("{%s} of {%s}", accepted, c.Vocabulary.Display(c.Domain))
}

func values(vs []model.Value) string {
	texts := []string{}
	for _, v := range vs {
		texts = append(texts, v.String())
	}
	return "[" + strings.Join(texts, ", ") + "]"
}

func (p *probe) field(f model.Field) string {
	if f.Name == "" {
		return "none"
	}
	locations := []string{}
	for _, d := range f.Declarations {
		locations = append(locations, p.places.Location(d))
	}
	slices.Sort(locations)
	return f.Name + " @ " + strings.Join(locations, ", ")
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func (p *probe) site(n *ast.Node) {
	if !model.Root(n) {
		return
	}
	s := p.a.Classify(n)
	if s == nil {
		return
	}
	tests := []string{}
	for _, t := range s.Tests {
		tests = append(tests, t.Value.String()+" in "+claim(t.Claim))
	}
	p.report("site", n, fmt.Sprintf("site %s; tests %s; group %q; named %q; outcome %q", s.Canon, strings.Join(tests, ", "), s.Group, s.Named, s.Outcome))
}

func (p *probe) outermost(n *ast.Node) bool {
	q := n.Parent
	for q != nil && q.Kind == ast.KindParenthesizedExpression {
		q = q.Parent
	}
	if q == nil || !ast.IsExpression(q) {
		return true
	}
	_, ok := p.a.TestOf(q)
	return !ok
}

func (p *probe) test(n *ast.Node) {
	t, ok := p.a.TestOf(n)
	if !ok || !p.outermost(n) {
		return
	}
	parts := []string{"value " + t.Value.String(), "claim " + claim(t.Claim), "field " + p.field(p.a.FieldOf(t.Value.Node))}
	if t.Named != "" {
		parts = append(parts, "named "+t.Claim.Vocabulary.Display(strings.Split(t.Named, ",")))
	}
	if t.Claim.Presence() {
		if held, ok := p.a.Held(t.Value.Node); ok {
			parts = append(parts, "held "+t.Claim.Vocabulary.Display(held))
		}
		parts = append(parts, "relied "+yes(p.a.Relied(t)))
	}
	if property := p.a.Property(t); property != nil {
		parts = append(parts, "property "+property.Name, "declared "+claim(p.a.DeclaredClaim(t)))
	}
	if fact, ok := p.a.FieldTest(n); ok {
		parts = append(parts, "answered by "+p.answered(fact))
	}
	p.report("test", n, "test "+strings.Join(parts, "; "))
}

func (p *probe) answered(t model.Test) string {
	tc := p.ctx.TypeChecker
	receiver := tc.GetTypeAtLocation(t.Receiver())
	if receiver == nil {
		return "unknown"
	}
	receiver = checker.Checker_getApparentType(tc, tc.GetNonNullableType(receiver))
	if receiver.Flags()&(checker.TypeFlagsAny|checker.TypeFlagsUnknown|checker.TypeFlagsNever) != 0 {
		return "unknown"
	}
	answers := []string{}
	if predicate, ok := p.a.PredicateFor(receiver, t); ok {
		answers = append(answers, "member "+predicate.Name)
	}
	for _, helper := range p.a.HelpersOf(receiver, t) {
		answers = append(answers, "helper "+helper.Name)
	}
	if len(answers) == 0 {
		return "none"
	}
	return strings.Join(answers, ", ")
}

func (p *probe) decision(d *model.Decision) {
	arms := []string{}
	for _, b := range d.Arms {
		outcome := string(b.Outcome.Source)
		if b.Outcome.Effects {
			outcome += " with effects"
		}
		if len(b.Outcome.Values) > 0 {
			outcome += " " + strings.Join(b.Outcome.Values, " | ")
		}
		flags := []string{}
		if b.Complete {
			flags = append(flags, "complete")
		}
		if b.Tested {
			flags = append(flags, "tested")
		}
		arms = append(arms, fmt.Sprintf("%s %s -> %s, logic %d, built %d", claim(b.Claim), strings.Join(flags, " "), outcome, len(b.Logic), len(p.a.Built(d, b))))
	}
	owner := "none"
	if d.Owner != nil {
		owner = d.Owner.Name + ", inside " + yes(d.Inside)
	}
	constructions := []string{}
	built := p.a.Constructions(d)
	for _, value := range slices.Sorted(maps.Keys(built)) {
		signatures := []string{}
		for _, c := range built[value] {
			signatures = append(signatures, c.Signature)
		}
		constructions = append(constructions, d.Vocabulary.Display([]string{value})+": "+strings.Join(signatures, "; "))
	}
	p.report("decision", d.Node, fmt.Sprintf("%s on %s; field %s; owner %s; arms %s; constructions [%s]",
		d.Form, model.Text(d.Node, 60), p.field(d.Field), owner, strings.Join(arms, " / "), strings.Join(constructions, " / ")))
}

func (p *probe) function(f *model.Function) {
	answer := []string{}
	for _, t := range f.Answer {
		answer = append(answer, t.Value.String()+" in "+claim(t.Claim))
	}
	domains := []string{}
	for _, d := range f.Domains {
		narrower := "own guard"
		if d.Selector != "" {
			narrower = "selector " + d.Selector
		}
		domains = append(domains, fmt.Sprintf("%s in %s by %s", d.Value.String(), claim(d.Claim), narrower))
	}
	if f.Kind == model.PlainFunction && len(domains) == 0 {
		return
	}
	p.reportHead("function", f.Node, fmt.Sprintf("%s %s; answer [%s]; domains [%s]", f.Kind, f.Name, strings.Join(answer, ", "), strings.Join(domains, ", ")))
}

func (p *probe) guard(kind string, g model.Guard) {
	p.report(kind, g.Test.Node, fmt.Sprintf("%s on %s; holds %s; whole %s; fallback %q; work %d", kind, g.Test.Value.String(), claim(g.Claim), yes(g.Whole), g.Shown, len(g.Work)))
	if len(g.Work) != 1 {
		return
	}
	call := model.Unwrap(g.Work[0])
	if call.Kind != ast.KindCallExpression {
		return
	}
	for _, argument := range call.Arguments() {
		if !p.a.ValueOf(argument).Same(g.Test.Value) {
			continue
		}
		b, ok := p.a.BoundaryOf(call, argument)
		if !ok {
			continue
		}
		caller := g.Claim
		if held, ok := p.a.Held(g.Test.Value.Node); ok {
			caller = caller.Under(held)
		}
		rechecks := []string{}
		for _, e := range p.a.Rechecks(b, g.Test, caller) {
			rechecks = append(rechecks, fmt.Sprintf("%s fallback %q", claim(e.Claim), e.Shown))
		}
		p.report("boundary", argument, fmt.Sprintf("boundary into %s of %s; keeps %s; rechecks [%s]", b.Parameter.Name().Text(), model.Text(call.Expression(), 60), yes(p.a.Keeps(b, g.Test)), strings.Join(rechecks, ", ")))
	}
}

// indexed renders values relative to parameters, by position, with $ for the parameter.
func indexed(m map[int][]model.Value) string {
	parts := []string{}
	for _, index := range slices.Sorted(maps.Keys(m)) {
		texts := []string{}
		for _, v := range m[index] {
			text := strings.Join(append([]string{"$"}, v.Path...), ".")
			if v.Partial {
				text = "part of " + text
			}
			texts = append(texts, text)
		}
		parts = append(parts, fmt.Sprintf("%d: [%s]", index, strings.Join(texts, ", ")))
	}
	return strings.Join(parts, ", ")
}

func (p *probe) flow(fn *ast.Node) {
	f := p.a.Flow(fn)
	if f == nil || len(f.Checks) == 0 && len(f.Calls) == 0 && len(f.Results) == 0 {
		return
	}
	checks := []string{}
	for _, c := range f.Checks {
		owned := ""
		if c.Owned {
			owned = " owned"
		}
		checks = append(checks, fmt.Sprintf("%s checks %s reads %s%s", model.Text(c.Node, 40), values(c.Checked), values(c.Read), owned))
	}
	calls := []string{}
	for _, c := range f.Calls {
		arguments := []string{}
		for _, a := range c.Arguments {
			arguments = append(arguments, values(a))
		}
		calls = append(calls, fmt.Sprintf("%s with %s after %d checks", model.Text(c.Node, 40), strings.Join(arguments, " "), len(c.Checks)))
	}
	carried := []string{}
	for _, index := range slices.Sorted(maps.Keys(f.Carried)) {
		carried = append(carried, fmt.Sprint(index))
	}
	p.reportHead("flow", fn, fmt.Sprintf("flow; checks [%s]; calls [%s]; results %s; carried [%s]; checked {%s}; read {%s}",
		strings.Join(checks, " / "), strings.Join(calls, " / "), values(f.Results), strings.Join(carried, ", "), indexed(f.Checked), indexed(f.Read)))
}

func (p *probe) contract(fn *ast.Node) {
	c := p.a.PredicateContractOf(fn)
	if c == nil {
		return
	}
	for _, r := range c.Restrictions {
		p.report("contract", r.Node, fmt.Sprintf("restriction on %s rejects %s of target %s", model.Text(r.Subject, 40), strings.TrimPrefix(r.Rejected, "s:"), p.ctx.TypeChecker.TypeToString(c.Target)))
	}
}

func (p *probe) run() {
	file := model.Build(p.a)
	for _, d := range file.Decisions {
		p.decision(d)
	}
	for _, f := range file.Functions {
		p.function(f)
	}
	for _, g := range file.Gates {
		p.report("gate", g.Selector.Node, fmt.Sprintf("gate %s -> %s", g.Names[0], g.Names[1]))
	}
	for _, r := range file.Relations {
		p.report("conflict", file.Decisions[r.A].Node, fmt.Sprintf("%s with line %d: %s", r.Kind, file.Decisions[r.B].Line, r.Detail))
	}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsExpression(n) {
			p.site(n)
			p.test(n)
		} else if n.Kind == ast.KindCaseClause {
			p.site(n)
		}
		if n.Kind == ast.KindConditionalExpression || n.Kind == ast.KindBinaryExpression {
			for _, g := range p.a.Guards(n) {
				p.guard("guard", g)
			}
		}
		if ast.IsFunctionLike(n) && n.Body() != nil {
			for _, g := range p.a.Entries(n) {
				p.guard("entry", g)
			}
			p.flow(n)
			p.contract(n)
		}
		n.ForEachChild(walk)
		return false
	}
	walk(p.ctx.SourceFile.AsNode())
}

// Rule reports the decision model's facts about a file as diagnostics, so they can be reviewed on fixtures and real
// programs. It is test-only and never shipped.
var Rule = rule.Rule{
	Name: "aexlint/model-facts",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		(&probe{ctx: ctx, a: model.NewAnalyzer(ctx), places: model.NewPlaces(ctx), kinds: selected(options)}).run()
		return rule.RuleListeners{}
	},
}
