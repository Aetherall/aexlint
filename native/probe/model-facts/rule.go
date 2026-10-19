package model_facts

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

var kinds = []string{"site", "test", "decision", "function", "gate", "guard", "entry", "boundary", "flow", "rejection", "contract", "callers", "result", "owner", "optional", "left", "write", "ending", "merge", "effect"}

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
	message := fmt.Sprintf("site %s; tests %s; group %q; named %q; outcome %q", s.Canon, strings.Join(tests, ", "), s.Group, s.Named, s.Outcome)
	if s.Definition != nil {
		message += fmt.Sprintf("; defined at line %d", scanner.GetECMALineOfPosition(ast.GetSourceFileOfNode(s.Definition), scanner.SkipTrivia(ast.GetSourceFileOfNode(s.Definition).Text(), s.Definition.Pos()))+1)
	}
	if s.Rest != "" {
		message += fmt.Sprintf("; otherwise %s leads to %q", s.Inverse, s.Rest)
	}
	p.report("site", n, message)
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
		parts = append(parts, "property "+p.a.FieldOf(t.Value.Node).Name, "declared "+claim(p.a.DeclaredClaim(t)))
	}
	if fact, ok := p.a.FieldTest(n); ok {
		parts = append(parts, "answered by "+p.answered(fact))
	}
	p.report("test", n, "test "+strings.Join(parts, "; "))
}

func (p *probe) answered(t model.Test) string {
	answers, ok := p.a.AnswersOf(t)
	if !ok {
		return "unknown"
	}
	found := []string{}
	if answers.Member != nil {
		found = append(found, "member "+answers.Member.Name)
	}
	for _, helper := range answers.Helpers {
		found = append(found, "helper "+helper.Name)
	}
	if len(found) == 0 {
		return "none"
	}
	return strings.Join(found, ", ")
}

func (p *probe) arm(d *model.Decision, b model.Branch) string {
	outcome := string(b.Outcome.Source)
	if b.Outcome.Commands {
		outcome += " with commands"
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
	return fmt.Sprintf("%s %s -> %s, control flow %d, built %d", claim(b.Claim), strings.Join(flags, " "), outcome, len(b.ControlFlow), len(p.a.Built(d, b)))
}

func (p *probe) decision(d *model.Decision) {
	arms := []string{}
	for _, b := range d.Arms {
		arms = append(arms, p.arm(d, b))
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
	following := ""
	if d.Following != nil {
		following = "; following " + p.arm(d, *d.Following)
	}
	p.report("decision", d.Node, fmt.Sprintf("%s on %s; field %s; owner %s; arms %s; constructions [%s]%s",
		d.Form, model.Text(d.Node, 60), p.field(d.Field), owner, strings.Join(arms, " / "), strings.Join(constructions, " / "), following))
}

func (p *probe) function(f *model.Function) {
	answer := []string{}
	for _, t := range f.Answer {
		answer = append(answer, t.Value.String()+" in "+claim(t.Claim))
	}
	narrowings := []string{}
	for _, d := range f.Narrowings {
		narrower := "own guard"
		if d.Selector != "" {
			narrower = "selector " + d.Selector
		}
		narrowings = append(narrowings, fmt.Sprintf("%s in %s by %s", d.Value.String(), claim(d.Claim), narrower))
	}
	if f.Kind == model.PlainFunction && len(narrowings) == 0 {
		return
	}
	p.reportHead("function", f.Node, fmt.Sprintf("%s %s; answer [%s]; narrowings [%s]", f.Kind, f.Name, strings.Join(answer, ", "), strings.Join(narrowings, ", ")))
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
	if f == nil || len(f.Rejections) == 0 && len(f.Calls) == 0 && len(f.Results) == 0 {
		return
	}
	checks := []string{}
	for _, c := range f.Rejections {
		owned := ""
		if c.Owned {
			owned = " owned"
		}
		checks = append(checks, fmt.Sprintf("%s checks %s reads %s%s", model.Text(c.Node, 40), values(c.Checked), values(c.Read), owned))
		p.report("rejection", c.Node, fmt.Sprintf("rejection; on %s reads %s%s", values(c.Checked), values(c.Read), owned))
	}
	calls := []string{}
	for _, c := range f.Calls {
		arguments := []string{}
		for _, a := range c.Arguments {
			arguments = append(arguments, values(a))
		}
		thrown := ""
		if c.Thrown {
			thrown = " thrown"
		}
		if c.Discarded {
			thrown += " discarded"
		}
		calls = append(calls, fmt.Sprintf("%s with %s after %d rejections%s", model.Text(c.Node, 40), strings.Join(arguments, " "), len(c.Rejections), thrown))
	}
	carried := []string{}
	for _, index := range slices.Sorted(maps.Keys(f.Carried)) {
		carried = append(carried, fmt.Sprint(index))
	}
	p.reportHead("flow", fn, fmt.Sprintf("flow; rejections [%s]; calls [%s]; results %s; carried [%s]; rejected {%s}; read {%s}",
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

func (p *probe) result(fn *ast.Node) {
	if !p.a.StringResult(fn) {
		return
	}
	returned := "other"
	if values, ok := p.a.Returned(fn); ok {
		returned = "[" + strings.Join(values, ", ") + "]"
	}
	calls, complete := p.a.Callers(fn)
	compared := 0
	for _, call := range calls {
		if p.a.ResultCompared(call) {
			compared++
		}
	}
	p.report("result", fn.Type(), fmt.Sprintf("result string; returned %s; callers %d, complete %s, compared %d", returned, len(calls), yes(complete), compared))
}

func (p *probe) callers(fn *ast.Node) {
	parameters := slices.DeleteFunc(slices.Clone(fn.Parameters()), ast.IsThisParameter)
	for index, parameter := range parameters {
		compared := p.a.LiteralComparisons(fn, parameter)
		if len(compared) == 0 {
			continue
		}
		calls, complete := p.a.Callers(fn)
		passed := "other"
		if values, widest, ok := p.a.Passed(calls, index); ok {
			passed = fmt.Sprintf("[%s], widest %d", strings.Join(values, ", "), widest)
		}
		p.report("callers", parameter, fmt.Sprintf("callers %d, complete %s; parameter %s, string %s, compared %d; passed %s",
			len(calls), yes(complete), parameter.Name().Text(), yes(p.a.StringParameter(parameter)), len(compared), passed))
	}
}

func (p *probe) owner(n *ast.Node) {
	if n.Kind != ast.KindPropertySignature && n.Kind != ast.KindPropertyDeclaration {
		return
	}
	owner := "none"
	if symbol := p.a.MemberOwner(n); symbol != nil {
		owner = symbol.Name
	}
	p.report("owner", n.Name(), "owner "+owner)
}

func (p *probe) optional(fn *ast.Node) {
	optional, returns, ok := p.a.OptionalSettings(fn)
	if !ok {
		return
	}
	names := []string{}
	for _, property := range optional {
		names = append(names, property.Name)
	}
	literals := []string{}
	for _, literal := range returns {
		marks := []string{}
		for i, setting := range literal {
			marks = append(marks, []string{"!", "", "?"}[setting]+optional[i].Name)
		}
		literals = append(literals, strings.Join(marks, " "))
	}
	p.reportHead("optional", fn, fmt.Sprintf("optional [%s]; returns [%s]", strings.Join(names, ", "), strings.Join(literals, " / ")))
}

func (p *probe) left(n *ast.Node) {
	if !p.kinds["left"] {
		return
	}
	names := []string{}
	for _, l := range p.a.LeftValues(n) {
		targets := []string{}
		for _, w := range l.Writes {
			targets = append(targets, model.Text(w.Target, 30))
		}
		names = append(names, fmt.Sprintf("%s (%s)", l.Value, strings.Join(targets, ", ")))
	}
	if len(names) > 0 {
		p.report("left", model.DecisionCondition(n), "left ["+strings.Join(names, "; ")+"]")
	}
}

func (p *probe) write(n *ast.Node) {
	if !p.kinds["write"] {
		return
	}
	for _, w := range model.WritesAt(n) {
		value := "unnamed"
		if v, ok := p.a.WriteValue(w); ok {
			value = v.String()
		}
		source := ""
		if w.Source != nil {
			source = " = " + model.Text(w.Source, 30)
		}
		p.report("write", w.Target, fmt.Sprintf("write %s %s%s; discarded %t", []string{"assignment", "update", "mutation"}[w.Kind], value, source, w.Discarded()))
	}
}

func (p *probe) merge(n *ast.Node) {
	if !p.kinds["merge"] {
		return
	}
	merge, ok := p.a.MergeOf(n)
	if !ok {
		return
	}
	values := p.a.Defined(merge)
	if len(values) == 0 {
		return
	}
	arms := []string{}
	for i, arm := range merge.Arms {
		label := "no branch"
		if arm.Branch != nil {
			label = model.Text(arm.Branch, 20)
		}
		parts := []string{}
		for _, v := range values {
			phi, _ := p.a.PhiOf(merge, v)
			missing := ""
			if p.a.MissingOn(arm, phi.Reference) {
				missing = " (missing on entry)"
			}
			parts = append(parts, fmt.Sprintf("%s: %s%s", v, p.a.InputProvenance(merge, phi, i), missing))
		}
		arms = append(arms, fmt.Sprintf("%s (%s)", label, strings.Join(parts, ", ")))
	}
	description := "merge [" + strings.Join(arms, "; ") + "]"
	carried := []string{}
	for _, v := range values {
		if phi, _ := p.a.PhiOf(merge, v); phi.Carried {
			carried = append(carried, v.String())
		}
	}
	if len(carried) > 0 {
		description += "; carried " + strings.Join(carried, ", ")
	}
	p.report("merge", model.DecisionCondition(n), description)
}

func (p *probe) effect(n *ast.Node) {
	if !p.kinds["effect"] {
		return
	}
	for _, w := range model.WritesAt(n) {
		e, ok := p.a.EffectOf(w)
		if !ok {
			continue
		}
		description := fmt.Sprintf("effect %s on %s", e.Kind, e.Receiver)
		if e.Key != nil {
			description += " at " + model.Text(e.Key, 20)
		}
		if e.Stored != nil {
			description += " of " + model.Text(e.Stored, 20)
		}
		p.report("effect", w.Node, description)
	}
}

func (p *probe) ending(n *ast.Node) {
	if !p.kinds["ending"] {
		return
	}
	switch n.Kind {
	case ast.KindIfStatement, ast.KindSwitchStatement, ast.KindLabeledStatement, ast.KindTryStatement, ast.KindForStatement,
		ast.KindForInStatement, ast.KindForOfStatement, ast.KindWhileStatement, ast.KindDoStatement:
		p.report("ending", n, "ending "+p.a.EndingText(n))
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
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsExpression(n) {
			p.site(n)
			p.test(n)
		} else if n.Kind == ast.KindCaseClause {
			p.site(n)
		}
		p.owner(n)
		if n.Kind == ast.KindIfStatement || n.Kind == ast.KindSwitchStatement || n.Kind == ast.KindExpressionStatement {
			p.left(n)
			p.merge(n)
		}
		p.effect(n)
		p.write(n)
		p.ending(n)
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
			p.callers(n)
			p.result(n)
			p.optional(n)
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
