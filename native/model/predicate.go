package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// Predicate is a function whose whole result is a test of a field: a member without parameters testing a field of
// this, or a helper of one parameter testing a field of it.
type Predicate struct {
	Name        string
	Declaration *ast.Node
	Body        *ast.Node
	Test        Test
	Narrows     bool
	Getter      bool
	Helper      bool
}

func (m *Analyzer) predicate(name string, declaration *ast.Node) (Predicate, bool) {
	fn := declaration
	switch declaration.Kind {
	case ast.KindMethodDeclaration, ast.KindGetAccessor:
	case ast.KindPropertyDeclaration:
		fn = declaration.Initializer()
		if fn == nil || fn.Kind != ast.KindArrowFunction && fn.Kind != ast.KindFunctionExpression {
			return Predicate{}, false
		}
	default:
		return Predicate{}, false
	}
	if len(fn.Parameters()) != 0 || !ProjectFile(ast.GetSourceFileOfNode(fn)) {
		return Predicate{}, false
	}
	e := result(fn)
	if e == nil {
		return Predicate{}, false
	}
	t, ok := m.FieldTest(e)
	if !ok || m.Receiver(t) == nil || m.Receiver(t).Kind != ast.KindThisKeyword {
		return Predicate{}, false
	}
	narrows := declaresPredicate(fn)
	return Predicate{Name: name, Declaration: declaration, Body: e, Test: t, Narrows: narrows, Getter: declaration.Kind == ast.KindGetAccessor}, true
}

// PredicateFor returns a predicate of t asking the same question as test, preferring one that accepts the same values.
func (m *Analyzer) PredicateFor(t *checker.Type, test Test) (Predicate, bool) {
	found := []Predicate{}
	for _, member := range checker.Checker_getPropertiesOfType(m.ctx.TypeChecker, t) {
		if member.Flags&(ast.SymbolFlagsMethod|ast.SymbolFlagsGetAccessor|ast.SymbolFlagsProperty) == 0 {
			continue
		}
		if p, ok := m.answers(member, test); ok {
			found = append(found, p)
		}
	}
	found = m.preferred(found, test)
	if len(found) == 0 {
		return Predicate{}, false
	}
	return found[0], true
}

func (m *Analyzer) preferred(found []Predicate, test Test) []Predicate {
	slices.SortStableFunc(found, func(a, b Predicate) int {
		x, y := !m.Opposite(a.Test, test), !m.Opposite(b.Test, test)
		switch {
		case x && !y:
			return -1
		case y && !x:
			return 1
		}
		return 0
	})
	return found
}

// answers returns the predicate member provides for test when every declaration a read of member can run, including
// overrides in project subclasses, asks the same question in the same direction.
func (m *Analyzer) answers(member *ast.Symbol, test Test) (Predicate, bool) {
	var first *Predicate
	answer := func(declaration *ast.Node) bool {
		p, ok := m.predicate(member.Name, declaration)
		if !ok || !m.SameQuestion(p.Test, test) || first != nil && m.Opposite(p.Test, test) != m.Opposite(first.Test, test) {
			return false
		}
		if first == nil {
			first = &p
		}
		return true
	}
	declarations := slices.DeleteFunc(slices.Clone(member.Declarations), func(d *ast.Node) bool { return !runs(d) })
	for _, declaration := range declarations {
		if !answer(declaration) {
			return Predicate{}, false
		}
	}
	for _, declaration := range declarations {
		if !ast.IsClassLike(declaration.Parent) {
			continue
		}
		for _, override := range m.overrides(declaration.Parent, member.Name) {
			if runs(override) && !answer(override) {
				return Predicate{}, false
			}
		}
	}
	if first == nil {
		return Predicate{}, false
	}
	return *first, true
}

// runs reports whether a member declaration has code a read can run: not a setter, an overload signature, or an
// abstract or declared member.
func runs(declaration *ast.Node) bool {
	if declaration.ModifierFlags()&(ast.ModifierFlagsAbstract|ast.ModifierFlagsAmbient) != 0 {
		return false
	}
	switch declaration.Kind {
	case ast.KindSetAccessor:
		return false
	case ast.KindMethodDeclaration:
		return declaration.Body() != nil
	}
	return true
}

// derived lists the classes of a program's project files that extend another class. It is built once per program from
// syntax alone and never mutated afterwards.
type derived struct {
	classes []*ast.Node
}

var derivedClasses = NewShared(func(program *compiler.Program) *derived {
	_, partial := Collect(program, func(n *ast.Node) bool {
		return ast.IsClassLike(n) && ast.GetClassExtendsHeritageElement(n) != nil
	})
	index := &derived{}
	for _, classes := range partial {
		index.classes = append(index.classes, classes...)
	}
	return index
})

// subclasses returns the project classes extending class directly, resolving each derived class's base once.
func (m *Analyzer) subclasses(class *ast.Node) []*ast.Node {
	if m.derivedFrom == nil {
		m.derivedFrom = map[*ast.Node][]*ast.Node{}
		for _, c := range derivedClasses.For(m.ctx.Program).classes {
			if base := m.baseOf(c); base != nil {
				m.derivedFrom[base] = append(m.derivedFrom[base], c)
			}
		}
	}
	return m.derivedFrom[class]
}

// overrides returns the instance members named name declared by project classes extending class, at any depth.
func (m *Analyzer) overrides(class *ast.Node, name string) []*ast.Node {
	found := []*ast.Node{}
	seen := map[*ast.Node]bool{class: true}
	var below func(*ast.Node)
	below = func(base *ast.Node) {
		for _, c := range m.subclasses(base) {
			if seen[c] {
				continue
			}
			seen[c] = true
			for _, member := range c.Members() {
				if member.Name() != nil && member.Name().Kind == ast.KindIdentifier && member.Name().Text() == name && member.ModifierFlags()&ast.ModifierFlagsStatic == 0 {
					found = append(found, member)
				}
			}
			below(c)
		}
	}
	below(class)
	return found
}

// Answers is what answers the question a field test asks: a predicate member of its receiver's type, or otherwise the
// program's helper predicates accepting that type. Union reports whether the receiver's type is a union.
type Answers struct {
	Member  *Predicate
	Helpers []Predicate
	Union   bool
}

// AnswersOf returns what answers t's question on its receiver's apparent type without null and undefined, or false
// when that type is any, unknown or never.
func (m *Analyzer) AnswersOf(t Test) (Answers, bool) {
	tc := m.ctx.TypeChecker
	receiver := tc.GetTypeAtLocation(m.Receiver(t))
	if receiver == nil {
		return Answers{}, false
	}
	receiver = checker.Checker_getApparentType(tc, tc.GetNonNullableType(receiver))
	if receiver.Flags()&(checker.TypeFlagsAny|checker.TypeFlagsUnknown|checker.TypeFlagsNever) != 0 {
		return Answers{}, false
	}
	answers := Answers{Union: receiver.Flags()&checker.TypeFlagsUnion != 0}
	if p, ok := m.PredicateFor(receiver, t); ok {
		answers.Member = &p
		return answers, true
	}
	answers.Helpers = m.HelpersOf(receiver, t)
	return answers, true
}

// Accepts reports whether helper, a helper predicate, accepts the receiver t tests as it is typed where t reads it.
func (m *Analyzer) Accepts(helper Predicate, t Test) bool {
	tc := m.ctx.TypeChecker
	actual := tc.GetTypeAtLocation(m.Receiver(t))
	return actual != nil && checker.Checker_isTypeAssignableTo(tc, actual, tc.GetTypeOfSymbol(helper.Declaration.Parameters()[0].Symbol()))
}

// Flag reports whether a property declaration's type, apart from null and undefined, is boolean: the property is its
// own answer.
func (m *Analyzer) Flag(declaration *ast.Node) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(declaration.Name())
	return t != nil && m.ctx.TypeChecker.GetNonNullableType(t).Flags()&checker.TypeFlagsBooleanLike != 0
}

// BooleanTyped reports whether n's type where it is written is boolean.
func (m *Analyzer) BooleanTyped(n *ast.Node) bool {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	return t != nil && t.Flags()&checker.TypeFlagsBooleanLike != 0
}

// Resolves reports whether a property read resolves, where it is read, to declaration.
func (m *Analyzer) Resolves(read, declaration *ast.Node) bool {
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(read.Name())
	return symbol != nil && slices.Contains(symbol.Declarations, declaration)
}
