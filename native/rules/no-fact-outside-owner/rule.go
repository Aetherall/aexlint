package no_fact_outside_owner

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type checkRun struct {
	ctx     rule.RuleContext
	a       *model.Analyzer
	places  model.Places
	covered map[*ast.Node]bool
}

func (r *checkRun) declares(field *ast.Symbol) bool {
	return slices.ContainsFunc(field.Declarations, func(d *ast.Node) bool { return ast.GetSourceFileOfNode(d) == r.ctx.SourceFile })
}

func (r *checkRun) check(test, reported *ast.Node) bool {
	fact, ok := r.a.FieldTest(test)
	if !ok || fact.Receiver().Kind == ast.KindThisKeyword {
		return false
	}
	t := r.ctx.TypeChecker.GetTypeAtLocation(fact.Receiver())
	if t == nil {
		return false
	}
	t = checker.Checker_getApparentType(r.ctx.TypeChecker, r.ctx.TypeChecker.GetNonNullableType(t))
	if t.Flags()&(checker.TypeFlagsAny|checker.TypeFlagsUnknown|checker.TypeFlagsNever) != 0 {
		return false
	}
	p, ok := r.a.PredicateFor(t, fact)
	if ok && r.a.InsideMemberOwner(test, p.Declaration) {
		return false
	}
	if !ok {
		if r.declares(r.a.Field(fact)) {
			return false
		}
		helpers := r.a.HelpersOf(t, fact)
		if len(helpers) == 0 || slices.ContainsFunc(helpers, func(h model.Predicate) bool { return ast.GetSourceFileOfNode(h.Declaration) == r.ctx.SourceFile }) {
			return false
		}
		p = helpers[0]
	}
	if r.a.Relied(fact) || !fact.Claim.Presence() && t.Flags()&checker.TypeFlagsUnion != 0 && !p.Narrows {
		return false
	}
	negation := ""
	if r.a.Opposite(fact, p.Test) {
		negation = "!"
	}
	receiver := model.Text(fact.Receiver(), 60)
	description := fmt.Sprintf("`%s` rewrites `%s`, which its type already provides.", model.Text(reported, 60), p.Name)
	definition := "the type's definition stays the only one"
	present := receiver + "." + p.Name
	missing := receiver + "?." + p.Name
	if !p.Getter && !p.Helper {
		present, missing = present+"()", missing+"()"
	}
	expected, known := false, true
	if p.Helper {
		description = fmt.Sprintf("`%s` rewrites `%s`, which `%s` already provides.", model.Text(reported, 60), p.Name, r.places.Relative(ast.GetSourceFileOfNode(p.Declaration).FileName()))
		definition = "its definition stays the only one"
		present = fmt.Sprintf("%s(%s)", p.Name, receiver)
		missing = present
		parameter := p.Declaration.Parameters()[0]
		actual := r.ctx.TypeChecker.GetTypeAtLocation(fact.Receiver())
		known = actual != nil && checker.Checker_isTypeAssignableTo(r.ctx.TypeChecker, actual, r.ctx.TypeChecker.GetTypeOfSymbol(parameter.Symbol()))
		if known {
			expected, known = r.a.WhenMissing(p.Body, parameter.Name())
		}
	}
	help := fmt.Sprintf("Use `%s%s` so %s.", negation, present, definition)
	if held, ok := r.a.Holds(fact.Receiver()); ok && held&(model.AbsentUndefined|model.AbsentNull) != 0 {
		result, ok := r.a.WhenMissing(reported, fact.Receiver())
		same := known && ok && result == (expected != (negation != ""))
		if same && !model.Boolean(reported) {
			same, missing = r.stored(reported, p, negation, missing)
		}
		if same {
			help = fmt.Sprintf("Use `%s%s` so %s.", negation, missing, definition)
		} else {
			help = fmt.Sprintf("Check that `%s` is present, then use `%s%s`, so %s.", receiver, negation, present, definition)
		}
	}
	r.ctx.ReportNode(reported, rule.RuleMessage{Id: "bypassedPredicate", Description: description, Help: help})
	return true
}

// stored adapts the suggestion for a missing receiver where the test's value is kept rather than used only as a
// boolean: the value must stay a boolean, so an optional call, which gives undefined, falls back to false.
func (r *checkRun) stored(reported *ast.Node, p model.Predicate, negation, missing string) (bool, string) {
	switch {
	case !r.boolean(reported):
		return false, missing
	case p.Helper:
		return r.boolean(p.Body), missing
	case negation == "":
		return true, missing + " ?? false"
	}
	return true, missing
}

func (r *checkRun) boolean(n *ast.Node) bool {
	t := r.ctx.TypeChecker.GetTypeAtLocation(n)
	return t != nil && t.Flags()&checker.TypeFlagsBooleanLike != 0
}

func (r *checkRun) cover(n *ast.Node) {
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		r.covered[c] = true
		c.ForEachChild(walk)
		return false
	}
	walk(n)
}

func (r *checkRun) visit(n *ast.Node) {
	if n.Name().Kind != ast.KindIdentifier {
		return
	}
	whole := r.a.WholeTest(n)
	if whole == nil || r.covered[whole] {
		return
	}
	if r.check(whole, whole) {
		r.cover(whole)
	}
}

// owner returns the name of the type declaring a property and whether code at n is inside that owner: the class and
// its subclasses for a class property, the declaring module for an interface or type alias property.
func (r *checkRun) owner(declaration *ast.Node) string {
	switch container := declaration.Parent; {
	case container.Kind == ast.KindInterfaceDeclaration:
		return container.Name().Text()
	case ast.IsClassLike(container) && container.Name() != nil:
		return container.Name().Text()
	case container.Kind == ast.KindTypeLiteral && container.Parent.Kind == ast.KindTypeAliasDeclaration:
		return container.Parent.Name().Text()
	}
	return ""
}

func (r *checkRun) inside(n, declaration *ast.Node) bool {
	if ast.IsClassLike(declaration.Parent) {
		return r.a.InsideMemberOwner(n, declaration)
	}
	return ast.GetSourceFileOfNode(n) == ast.GetSourceFileOfNode(declaration)
}

type question struct {
	test  model.Test
	files map[*ast.SourceFile]bool
	reads []*ast.Node
	// shared names the other types in this file whose declarations the same tests ask about.
	shared []string
}

// key identifies the tests a question groups.
func (q *question) key() string {
	parts := []string{}
	for _, read := range q.reads {
		parts = append(parts, fmt.Sprintf("%s:%d", ast.GetSourceFileOfNode(read).FileName(), read.Pos()))
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}

// questions groups the facts that tests outside the owner establish about a property declaration.
func (r *checkRun) questions(declaration *ast.Node, reads []*ast.Node) []*question {
	found := []*question{}
	for _, read := range reads {
		symbol := r.ctx.TypeChecker.GetSymbolAtLocation(read.Name())
		if symbol == nil || !slices.Contains(symbol.Declarations, declaration) || r.inside(read, declaration) {
			continue
		}
		whole := r.a.WholeTest(read)
		if whole == nil {
			continue
		}
		t, ok := r.a.FieldTest(whole)
		if !ok || t.Value.Node != model.Unwrap(read) || !r.a.About(t, declaration) {
			continue
		}
		i := slices.IndexFunc(found, func(q *question) bool { return r.a.SameQuestion(q.test, t) })
		if i < 0 {
			found = append(found, &question{test: t, files: map[*ast.SourceFile]bool{}})
			i = len(found) - 1
		}
		found[i].files[ast.GetSourceFileOfNode(read)] = true
		found[i].reads = append(found[i].reads, read)
	}
	return found
}

func (r *checkRun) answered(t model.Test) bool {
	receiver := r.ctx.TypeChecker.GetTypeAtLocation(t.Receiver())
	if receiver == nil {
		return false
	}
	receiver = checker.Checker_getApparentType(r.ctx.TypeChecker, r.ctx.TypeChecker.GetNonNullableType(receiver))
	if _, ok := r.a.PredicateFor(receiver, t); ok {
		return true
	}
	return len(r.a.HelpersOf(receiver, t)) > 0
}

func (r *checkRun) unanswered(limit int) {
	declarations := map[string][]*ast.Node{}
	names := []string{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if (n.Kind == ast.KindPropertySignature || n.Kind == ast.KindPropertyDeclaration) && n.Name().Kind == ast.KindIdentifier && r.owner(n) != "" {
			name := n.Name().Text()
			if declarations[name] == nil {
				names = append(names, name)
			}
			declarations[name] = append(declarations[name], n)
		}
		n.ForEachChild(walk)
		return false
	}
	walk(r.ctx.SourceFile.AsNode())
	for _, name := range names {
		reads := r.a.TestedReads(name)
		if len(reads) <= limit {
			continue
		}
		found := map[*ast.Node][]*question{}
		first := map[string]*question{}
		for _, declaration := range declarations[name] {
			if r.flag(declaration) || declaration.Initializer() != nil && ast.IsFunctionLike(model.Unwrap(declaration.Initializer())) {
				continue
			}
			for _, q := range r.questions(declaration, reads) {
				if len(q.files) <= limit || r.answered(q.test) {
					continue
				}
				if earlier, ok := first[q.key()]; ok {
					earlier.shared = append(earlier.shared, "`"+r.owner(declaration)+"`")
					continue
				}
				first[q.key()] = q
				found[declaration] = append(found[declaration], q)
			}
		}
		for _, declaration := range declarations[name] {
			if len(found[declaration]) > 0 {
				r.reportUnanswered(declaration, found[declaration])
			}
		}
	}
}

// flag reports whether a property's type, apart from null and undefined, is boolean: the property is its own answer.
func (r *checkRun) flag(declaration *ast.Node) bool {
	t := r.ctx.TypeChecker.GetTypeAtLocation(declaration.Name())
	return t != nil && r.ctx.TypeChecker.GetNonNullableType(t).Flags()&checker.TypeFlagsBooleanLike != 0
}

// asked describes the fact a test establishes: the missing values a presence test splits off, or the literal values a
// comparison names.
func (r *checkRun) asked(t model.Test) string {
	if !t.Claim.Presence() {
		return "compared with " + t.Leaf.Vocabulary.Display(strings.Split(t.Named, ","))
	}
	claim := r.a.DeclaredClaim(t)
	missing := claim.Accepted
	if slices.Contains(missing, model.Present) {
		missing = claim.Rejected()
	}
	names := []string{}
	for _, key := range []string{model.MissingNull, model.MissingUndefined, model.MissingFalsy} {
		if slices.Contains(missing, key) {
			names = append(names, map[string]string{model.MissingNull: "`null`", model.MissingUndefined: "`undefined`", model.MissingFalsy: "other falsy values"}[key])
		}
	}
	return "checked for " + strings.Join(names, " and ")
}

func (r *checkRun) reportUnanswered(declaration *ast.Node, unanswered []*question) {
	slices.SortStableFunc(unanswered, func(a, b *question) int { return len(b.files) - len(a.files) })
	facts := []string{}
	files := map[*ast.SourceFile]bool{}
	for _, q := range unanswered {
		facts = append(facts, fmt.Sprintf("%s in %d files", r.asked(q.test), len(q.files)))
		for file := range q.files {
			files[file] = true
		}
	}
	name, owner := declaration.Name().Text(), r.owner(declaration)
	them, listed := "it", model.Listed(facts, 3)
	if len(facts) == 2 {
		them, listed = "them", facts[0]+" and "+facts[1]
	} else if len(facts) > 2 {
		them = "them"
	}
	shared := []string{}
	for _, q := range unanswered {
		for _, other := range q.shared {
			if !slices.Contains(shared, other) {
				shared = append(shared, other)
			}
		}
	}
	also := ""
	if len(shared) > 0 {
		also = fmt.Sprintf(" The same tests also ask about `%s` of %s.", name, model.Listed(shared, 3))
	}
	r.ctx.ReportNode(declaration.Name(), rule.RuleMessage{
		Id:          "unansweredFact",
		Description: fmt.Sprintf("`%s` is %s outside `%s`, which has no predicate for %s.%s", name, listed, owner, them, also),
		Help: fmt.Sprintf("Each of those files decides on its own what this test means, so changing that meaning means finding every copy. A predicate on `%s`, or an exported function in its module, would hold the answer in one place. Files, nearest first: %s.",
			owner, model.Listed(r.places.Nearest(files), 3)),
	})
}

var Rule = rule.Rule{
	Name: "aexlint/no-fact-outside-owner",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		limit := model.ParseMax("no-fact-outside-owner", options, 4)
		classifier := model.NewClassifier(ctx.TypeChecker)
		r := &checkRun{ctx: ctx, a: model.NewAnalyzer(ctx, classifier), places: model.NewPlaces(ctx), covered: map[*ast.Node]bool{}}
		r.unanswered(limit)
		return rule.RuleListeners{ast.KindPropertyAccessExpression: r.visit}
	},
}
