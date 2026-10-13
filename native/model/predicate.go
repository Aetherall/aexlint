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
	if !ok || t.Receiver().Kind != ast.KindThisKeyword {
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

// overrides returns the instance members named name declared by project classes extending class.
func (m *Analyzer) overrides(class *ast.Node, name string) []*ast.Node {
	found := []*ast.Node{}
	for _, c := range derivedClasses.For(m.ctx.Program).classes {
		if c == class || !m.Extends(c, class) {
			continue
		}
		for _, member := range c.Members() {
			if member.Name() != nil && member.Name().Kind == ast.KindIdentifier && member.Name().Text() == name && member.ModifierFlags()&ast.ModifierFlagsStatic == 0 {
				found = append(found, member)
			}
		}
	}
	return found
}
