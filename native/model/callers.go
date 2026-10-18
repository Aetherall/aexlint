package model

import (
	"slices"
	"strconv"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// references lists, by name, the identifiers of a program's project files named like a function that compares a
// parameter with a string literal or declares a string result. It is built once per program from syntax alone and never
// mutated afterwards.
type references struct {
	byName map[string][]*ast.Node
}

var referenced = NewShared(func(program *compiler.Program) *references {
	_, functions := Collect(program, func(n *ast.Node) bool {
		return functionName(n) != "" && (writesString(n.Type()) || slices.ContainsFunc(n.Parameters(), func(p *ast.Node) bool {
			return p.Name().Kind == ast.KindIdentifier && len(literalComparisons(n.Body(), func(r *ast.Node) bool { return r.Kind == ast.KindIdentifier && r.Text() == p.Name().Text() })) > 0
		}))
	})
	names := map[string]bool{}
	for _, found := range functions {
		for _, fn := range found {
			names[functionName(fn)] = true
		}
	}
	_, nodes := Collect(program, func(n *ast.Node) bool {
		return (n.Kind == ast.KindIdentifier || n.Kind == ast.KindPrivateIdentifier) && names[n.Text()]
	})
	index := &references{byName: map[string][]*ast.Node{}}
	for _, found := range nodes {
		for _, n := range found {
			index.byName[n.Text()] = append(index.byName[n.Text()], n)
		}
	}
	return index
})

// writesString reports whether a written type is string, alone or in a union.
func writesString(t *ast.Node) bool {
	if t == nil {
		return false
	}
	if t.Kind == ast.KindUnionType {
		return slices.ContainsFunc(t.AsUnionTypeNode().Types.Nodes, writesString)
	}
	return t.Kind == ast.KindStringKeyword
}

// functionName returns the name a function with a body is called by: its own, or that of the variable or class field
// it initializes. It returns "" for other functions, such as object literal members and callbacks.
func functionName(fn *ast.Node) string {
	if !ast.IsFunctionLike(fn) || fn.Body() == nil {
		return ""
	}
	name := fn.Name()
	switch fn.Kind {
	case ast.KindFunctionDeclaration, ast.KindMethodDeclaration:
	case ast.KindArrowFunction, ast.KindFunctionExpression:
		if p := fn.Parent; p.Kind == ast.KindVariableDeclaration || p.Kind == ast.KindPropertyDeclaration {
			name = p.Name()
		}
	default:
		return ""
	}
	if name == nil || name.Kind != ast.KindIdentifier && name.Kind != ast.KindPrivateIdentifier {
		return ""
	}
	return name.Text()
}

// literalComparisons returns where root compares an expression is accepts with a string literal: an equality, a case
// label of a switch on it, or includes on a literal list.
func literalComparisons(root *ast.Node, is func(*ast.Node) bool) []*ast.Node {
	found := []*ast.Node{}
	reference := func(n *ast.Node) bool { return is(Unwrap(n)) }
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		switch n.Kind {
		case ast.KindBinaryExpression:
			b := n.AsBinaryExpression()
			switch b.OperatorToken.Kind {
			case ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsEqualsToken, ast.KindEqualsEqualsToken, ast.KindExclamationEqualsToken:
				if reference(b.Left) && ast.IsStringLiteralLike(Unwrap(b.Right)) || reference(b.Right) && ast.IsStringLiteralLike(Unwrap(b.Left)) {
					found = append(found, n)
				}
			}
		case ast.KindSwitchStatement:
			if reference(n.Expression()) {
				for _, clause := range n.AsSwitchStatement().CaseBlock.AsCaseBlock().Clauses.Nodes {
					if clause.Kind == ast.KindCaseClause && ast.IsStringLiteralLike(Unwrap(clause.Expression())) {
						found = append(found, clause)
					}
				}
			}
		case ast.KindCallExpression:
			callee, arguments := Unwrap(n.Expression()), n.Arguments()
			if callee.Kind == ast.KindPropertyAccessExpression && callee.Name().Text() == "includes" && len(arguments) == 1 && reference(arguments[0]) {
				if list := Unwrap(callee.Expression()); list.Kind == ast.KindArrayLiteralExpression && len(list.Elements()) > 0 && !slices.ContainsFunc(list.Elements(), func(e *ast.Node) bool { return !ast.IsStringLiteralLike(e) }) {
					found = append(found, n)
				}
			}
		}
		n.ForEachChild(walk)
		return false
	}
	walk(root)
	return found
}

// LiteralComparisons returns where fn compares parameter with a string literal: an equality, a case label of a switch
// on it, or includes on a literal list.
func (m *Analyzer) LiteralComparisons(fn, parameter *ast.Node) []*ast.Node {
	symbol := parameter.Symbol()
	if symbol == nil || parameter.Name().Kind != ast.KindIdentifier {
		return nil
	}
	return literalComparisons(fn.Body(), func(n *ast.Node) bool {
		return n.Kind == ast.KindIdentifier && m.ctx.TypeChecker.GetSymbolAtLocation(n) == symbol
	})
}

// functionSymbol returns the symbol fn is called through: its own, or that of the variable or class field it
// initializes.
func functionSymbol(fn *ast.Node) *ast.Symbol {
	if fn.Kind == ast.KindArrowFunction || fn.Kind == ast.KindFunctionExpression {
		return fn.Parent.Symbol()
	}
	return fn.Symbol()
}

// Callers returns the calls to fn in the program's project files, and whether they are all its uses. They are known
// only for a named function that compares a parameter with a string literal or declares a string result. A call to a
// member of a type that fn's class instances are assignable to, such as an interface, counts as a call to a method.
// They are complete only when fn is used by being called: not passed, stored, default-exported, renamed on import or
// export, destructured or overloaded, and not a member a base class also declares, whose own calls are not seen. Calls
// on any and through element access are not seen.
func (m *Analyzer) Callers(fn *ast.Node) ([]*ast.Node, bool) {
	name := functionName(fn)
	symbol := functionSymbol(fn)
	if name == "" || symbol == nil {
		return nil, false
	}
	complete := len(symbol.Declarations) == 1 && !ast.HasSyntacticModifier(fn, ast.ModifierFlagsDefault) && !m.inherited(fn)
	instance := m.instance(fn)
	declared := fn.Name()
	if declared == nil {
		declared = fn.Parent.Name()
	}
	tc := m.ctx.TypeChecker
	calls := []*ast.Node{}
	for _, n := range referenced.For(m.ctx.Program).byName[name] {
		if n == declared || ast.IsPartOfTypeNode(n) {
			continue
		}
		switch n.Parent.Kind {
		case ast.KindBindingElement, ast.KindShorthandPropertyAssignment:
			complete = false
			continue
		}
		found := tc.GetSymbolAtLocation(n)
		if found == nil {
			continue
		}
		if checker.SkipAlias(found, tc) != symbol && !m.reaches(n, instance) {
			continue
		}
		switch p := n.Parent; p.Kind {
		case ast.KindImportSpecifier, ast.KindExportSpecifier:
			if p.PropertyName() != nil && p.PropertyName().Text() != p.Name().Text() {
				complete = false
			}
			continue
		}
		callee := n
		if p := n.Parent; p.Kind == ast.KindPropertyAccessExpression && p.Name() == n {
			callee = p
		}
		if call := callee.Parent; call.Kind == ast.KindCallExpression && call.Expression() == callee {
			calls = append(calls, call)
			continue
		}
		complete = false
	}
	return calls, complete
}

// instance returns the type of the instances of the class fn is a method of, or nil for static methods and other
// functions.
func (m *Analyzer) instance(fn *ast.Node) *checker.Type {
	if fn.Kind != ast.KindMethodDeclaration || !ast.IsClassLike(fn.Parent) || ast.HasSyntacticModifier(fn, ast.ModifierFlagsStatic) || fn.Parent.Symbol() == nil {
		return nil
	}
	return m.ctx.TypeChecker.GetDeclaredTypeOfSymbol(fn.Parent.Symbol())
}

// reaches reports whether n names a member read from a value that instance is assignable to, so a call through it
// can reach a method of instance's class.
func (m *Analyzer) reaches(n *ast.Node, instance *checker.Type) bool {
	p := n.Parent
	if instance == nil || p.Kind != ast.KindPropertyAccessExpression || p.Name() != n {
		return false
	}
	tc := m.ctx.TypeChecker
	receiver := tc.GetTypeAtLocation(p.Expression())
	if receiver == nil || receiver.Flags()&checker.TypeFlagsAny != 0 {
		return false
	}
	return tc.IsTypeAssignableTo(instance, tc.GetNonNullableType(receiver))
}

// inherited reports whether fn is a class member that a class it extends also declares, or one whose base is unknown.
func (m *Analyzer) inherited(fn *ast.Node) bool {
	owner := fn.Parent
	if !ast.IsClassLike(owner) || fn.Name() == nil || fn.Name().Kind == ast.KindPrivateIdentifier {
		return false
	}
	clause := ast.GetHeritageClause(owner, ast.KindExtendsKeyword)
	if clause == nil {
		return false
	}
	tc := m.ctx.TypeChecker
	for _, element := range clause.AsHeritageClause().Types.Nodes {
		base := tc.GetTypeFromTypeNode(element)
		if ast.HasSyntacticModifier(fn, ast.ModifierFlagsStatic) {
			base = tc.GetTypeAtLocation(element.Expression())
		}
		if base == nil || base.Flags()&checker.TypeFlagsAny != 0 || tc.GetPropertyOfType(base, fn.Name().Text()) != nil {
			return true
		}
	}
	return false
}

// StringParameter reports whether parameter is declared as string, possibly with null or undefined.
func (m *Analyzer) StringParameter(parameter *ast.Node) bool {
	if parameter.Type() == nil || parameter.AsParameterDeclaration().DotDotDotToken != nil {
		return false
	}
	t := m.ctx.TypeChecker.GetTypeAtLocation(parameter.Name())
	return t != nil && m.ctx.TypeChecker.GetNonNullableType(t).Flags()&checker.TypeFlagsString != 0
}

// Passed returns the string literals that calls pass for the parameter at index, quoted and sorted, the most of them one
// argument's type holds, and whether every call passes only those: a call leaving it out passes nothing, and null and
// undefined are ignored. An argument holding several is a value of a union, not a literal written at the call.
func (m *Analyzer) Passed(calls []*ast.Node, index int) ([]string, int, bool) {
	arguments := []*ast.Node{}
	for _, call := range calls {
		passed := call.Arguments()
		if slices.ContainsFunc(passed[:min(index+1, len(passed))], func(a *ast.Node) bool { return a.Kind == ast.KindSpreadElement }) {
			return nil, 0, false
		}
		if index < len(passed) {
			arguments = append(arguments, passed[index])
		}
	}
	return m.stringLiterals(arguments)
}

// StringResult reports whether fn declares a string result, possibly with null or undefined, and returns it once: it is
// neither async nor a generator.
func (m *Analyzer) StringResult(fn *ast.Node) bool {
	if fn.Type() == nil || fn.Body() == nil || ast.GetFunctionFlags(fn)&(ast.FunctionFlagsAsync|ast.FunctionFlagsGenerator) != 0 {
		return false
	}
	t := m.ctx.TypeChecker.GetTypeFromTypeNode(fn.Type())
	return t != nil && m.ctx.TypeChecker.GetNonNullableType(t).Flags()&checker.TypeFlagsString != 0
}

// Returned returns the string literals fn returns, quoted and sorted, and whether it returns only those, ignoring null
// and undefined.
func (m *Analyzer) Returned(fn *ast.Node) ([]string, bool) {
	body := fn.Body()
	if body.Kind != ast.KindBlock {
		values, _, ok := m.stringLiterals([]*ast.Node{body})
		return values, ok
	}
	returned := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return false
		}
		if n.Kind == ast.KindReturnStatement && n.Expression() != nil {
			returned = append(returned, n.Expression())
		}
		n.ForEachChild(walk)
		return false
	}
	body.ForEachChild(walk)
	values, _, ok := m.stringLiterals(returned)
	return values, ok
}

// ResultCompared reports whether the result of call is compared with a string literal: directly, through a const it
// initializes, or by the function it is passed to.
func (m *Analyzer) ResultCompared(call *ast.Node) bool {
	n := call
	for p := n.Parent; p != nil && Unwrap(p) != p; p = n.Parent {
		n = p
	}
	p := n.Parent
	if len(literalComparisons(p, func(e *ast.Node) bool { return e == call })) > 0 {
		return true
	}
	if p.Kind == ast.KindVariableDeclaration && p.Initializer() == n && p.Name().Kind == ast.KindIdentifier && p.Parent.Flags&ast.NodeFlagsConst != 0 {
		symbol := p.Symbol()
		scope := enclosingFunction(p)
		if scope == nil {
			scope = ast.GetSourceFileOfNode(p).AsNode()
		} else {
			scope = scope.Body()
		}
		return len(literalComparisons(scope, func(e *ast.Node) bool {
			return e.Kind == ast.KindIdentifier && m.ctx.TypeChecker.GetSymbolAtLocation(e) == symbol
		})) > 0
	}
	if p.Kind == ast.KindCallExpression && p.Expression() != n {
		if b, ok := m.BoundaryOf(p, n); ok {
			return len(m.LiteralComparisons(b.Callee, b.Parameter)) > 0
		}
	}
	return false
}

// stringLiterals returns the string literals the types of nodes hold, quoted and sorted, the most of them one node's type
// holds, and whether they hold only those, ignoring null and undefined.
func (m *Analyzer) stringLiterals(nodes []*ast.Node) ([]string, int, bool) {
	tc := m.ctx.TypeChecker
	values := []string{}
	widest := 0
	for _, n := range nodes {
		t := tc.GetTypeAtLocation(n)
		if t == nil {
			return nil, 0, false
		}
		members := []*checker.Type{t}
		if t.Flags()&checker.TypeFlagsUnion != 0 {
			members = t.Types()
		}
		held := 0
		for _, member := range members {
			if member.Flags()&(checker.TypeFlagsNull|checker.TypeFlagsUndefined) != 0 {
				continue
			}
			if member.Flags()&checker.TypeFlagsStringLiteral == 0 {
				return nil, 0, false
			}
			value, ok := member.AsLiteralType().Value().(string)
			if !ok {
				return nil, 0, false
			}
			held++
			if shown := strconv.Quote(value); !slices.Contains(values, shown) {
				values = append(values, shown)
			}
		}
		widest = max(widest, held)
	}
	slices.Sort(values)
	return values, widest, true
}
