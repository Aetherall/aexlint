package no_forgeable_validated_result

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

type storedKey struct {
	member *ast.Node
	index  int
}

// owner returns the class declaring a hand-off's callee: a constructor or method with a body, in a project source file.
func (m *measure) owner(call *ast.Node) (*ast.Node, *ast.Node) {
	signature := m.ctx.TypeChecker.GetResolvedSignature(call)
	if signature == nil || signature.Declaration() == nil {
		return nil, nil
	}
	declaration := signature.Declaration()
	if declaration.Kind != ast.KindConstructor && declaration.Kind != ast.KindMethodDeclaration {
		return nil, nil
	}
	class := declaration.Parent
	if declaration.Body() == nil || class == nil || !ast.IsClassLike(class) || !m.projectFile(ast.GetSourceFileOfNode(declaration)) {
		return nil, nil
	}
	return class, declaration
}

// references reports whether an expression mentions one of symbols.
func (m *measure) references(n *ast.Node, symbols map[*ast.Symbol]bool) bool {
	found := false
	var walk func(*ast.Node) bool
	walk = func(child *ast.Node) bool {
		if child.Kind == ast.KindIdentifier {
			symbol := m.ctx.TypeChecker.GetSymbolAtLocation(child)
			if parent := child.Parent; parent != nil && parent.Kind == ast.KindShorthandPropertyAssignment && parent.Name() == child {
				symbol = m.ctx.TypeChecker.GetShorthandAssignmentValueSymbol(parent)
			}
			found = symbols[symbol]
		}
		if !found {
			child.ForEachChild(walk)
		}
		return found
	}
	walk(n)
	return found
}

// thisProperty reports whether an assignment target is a property of `this`, at any depth.
func thisProperty(n *ast.Node) bool {
	for n.Kind == ast.KindPropertyAccessExpression || n.Kind == ast.KindElementAccessExpression {
		n = n.Expression()
	}
	return n.Kind == ast.KindThisKeyword
}

// factory reports whether a static method returns an instance of its class, directly or as a property of the
// returned object, through a promise or not.
func (m *measure) factory(member *ast.Node) bool {
	class := member.Parent
	if !ast.HasSyntacticModifier(member, ast.ModifierFlagsStatic) || class.Name() == nil {
		return false
	}
	symbol := m.ctx.TypeChecker.GetSymbolAtLocation(class.Name())
	signature := m.ctx.TypeChecker.GetSignatureFromDeclaration(member)
	if symbol == nil || signature == nil {
		return false
	}
	t := m.ctx.TypeChecker.GetReturnTypeOfSignature(signature)
	if awaited := checker.Checker_getAwaitedType(m.ctx.TypeChecker, t); awaited != nil {
		t = awaited
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	if t.Symbol() == symbol {
		return true
	}
	for _, property := range m.ctx.TypeChecker.GetPropertiesOfType(t) {
		if m.ctx.TypeChecker.GetNonNullableType(m.ctx.TypeChecker.GetTypeOfSymbol(property)).Symbol() == symbol {
			return true
		}
	}
	return false
}

// ownReceiver reports whether a call's receiver is `this` or `super`: a member of the class or of a base class.
func ownReceiver(call *ast.Node) bool {
	callee := call.Expression()
	if callee.Kind != ast.KindPropertyAccessExpression && callee.Kind != ast.KindElementAccessExpression {
		return false
	}
	receiver := callee.Expression()
	return receiver.Kind == ast.KindThisKeyword || receiver.Kind == ast.KindSuperKeyword
}

// stores reports whether a constructor or method takes its parameter at index into its class's state: as a
// constructor parameter property; by assigning it to a property of `this`; by passing it to a method of a property of
// `this` whose result is discarded (`this.events.push(value);`, called for its effect rather than as a query); by passing it to a member called on `this` or `super`, or to a constructor or
// member of the same class, that stores it; or, for a static factory returning an instance of the class, by reading it
// at all. Nested functions are not searched.
func (m *measure) stores(member *ast.Node, index int) bool {
	key := storedKey{member, index}
	if result, ok := m.stored[key]; ok {
		return result
	}
	m.stored[key] = false
	parameters := member.Parameters()
	if index >= len(parameters) || member.Body() == nil {
		return false
	}
	parameter := parameters[index]
	if member.Kind == ast.KindConstructor && ast.IsParameterPropertyDeclaration(parameter, member) {
		m.stored[key] = true
		return true
	}
	if parameter.Name().Kind != ast.KindIdentifier {
		return false
	}
	symbols := map[*ast.Symbol]bool{m.ctx.TypeChecker.GetSymbolAtLocation(parameter.Name()): true}
	class := member.Parent
	result := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if result || ast.IsFunctionLike(n) || ast.IsClassLike(n) {
			return result
		}
		switch n.Kind {
		case ast.KindBinaryExpression:
			if binary := n.AsBinaryExpression(); ast.IsAssignmentExpression(n, false) && thisProperty(binary.Left) && m.references(binary.Right, symbols) {
				result = true
			}
		case ast.KindCallExpression, ast.KindNewExpression:
			callee := n.Expression()
			stateMethod := n.Kind == ast.KindCallExpression && callee.Kind == ast.KindPropertyAccessExpression &&
				callee.Expression().Kind != ast.KindThisKeyword && thisProperty(callee.Expression()) &&
				n.Parent != nil && n.Parent.Kind == ast.KindExpressionStatement
			owner, declaration := m.owner(n)
			for j, argument := range n.Arguments() {
				if !m.references(argument, symbols) {
					continue
				}
				switch {
				case stateMethod:
					result = true
				case declaration != nil && (owner == class || ownReceiver(n)) && m.stores(declaration, j):
					result = true
				}
			}
		}
		if !result {
			n.ForEachChild(walk)
		}
		return result
	}
	member.Body().ForEachChild(walk)
	if !result && m.factory(member) {
		result = m.references(member.Body(), symbols)
	}
	m.stored[key] = result
	return result
}
