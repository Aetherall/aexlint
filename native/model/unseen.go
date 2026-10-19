package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

// Unseen lists where nodes may run code the model does not see: an await or yield, which lets other code run, a call
// or construction neither resolving to a project function with a body nor declared by the default library, or one
// resolving to a project function whose summary is opaque. Such code may read or change any data visible to it.
// Default library functions only reach data passed to them, and the callbacks they call are read where they are
// written. With nested set, the functions nodes declare are read too.
func (m *Analyzer) Unseen(nodes []*ast.Node, nested bool) []*ast.Node {
	found := []*ast.Node{}
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if !nested && (ast.IsFunctionLike(c) || ast.IsClassLike(c)) {
			return false
		}
		switch c.Kind {
		case ast.KindAwaitExpression, ast.KindYieldExpression:
			found = append(found, c)
		case ast.KindCallExpression, ast.KindNewExpression, ast.KindTaggedTemplateExpression:
			if fn := m.followed(c); fn != nil {
				if m.Summary(fn).Opaque {
					found = append(found, c)
				}
			} else if !m.library(c) {
				found = append(found, c)
			}
		}
		c.ForEachChild(walk)
		return false
	}
	for _, n := range nodes {
		if n != nil {
			walk(n)
		}
	}
	return found
}

// library reports whether a call resolves to a function the default library declares.
func (m *Analyzer) library(call *ast.Node) bool {
	signature := m.ctx.TypeChecker.GetResolvedSignature(call)
	if signature == nil || signature.Declaration() == nil {
		return false
	}
	return utils.IsSourceFileDefaultLibrary(m.ctx.Program, ast.GetSourceFileOfNode(signature.Declaration()))
}

// private reports whether only code the model sees can reach the data v names until to: its root is a variable of
// fn, never rebound, initialized with a new object, not referenced in the functions fn declares, and, before to, only
// used as the head of a property read, element read or method call, returned, which ends the function, or as an
// argument to a project function whose summary is not opaque and carries none of its arguments into other data.
func (m *Analyzer) private(v Value, fn *ast.Node, to int) bool {
	if v.Root == nil || v.Root.ValueDeclaration == nil {
		return false
	}
	d := v.Root.ValueDeclaration
	if d.Kind != ast.KindVariableDeclaration || enclosingFunction(d) != fn || m.assigned(v.Root) || m.ProvenanceOf(d.Initializer()) != FreshValue {
		return false
	}
	if fresh := m.followed(Unwrap(d.Initializer())); fresh != nil && m.Summary(fresh).Opaque {
		return false
	}
	private := true
	var walk func(*ast.Node) bool
	walk = func(c *ast.Node) bool {
		if !private {
			return true
		}
		if c.Kind == ast.KindIdentifier && c.Parent != nil && c.Parent != d && m.ctx.TypeChecker.GetSymbolAtLocation(c) == v.Root {
			if enclosingFunction(c) != fn || c.Pos() < to && !m.contained(c) {
				private = false
			}
			return !private
		}
		c.ForEachChild(walk)
		return !private
	}
	walk(fn.Body())
	return private
}

// contained reports whether a reference only reads through its data or calls on it, as private allows.
func (m *Analyzer) contained(c *ast.Node) bool {
	p := c.Parent
	for p.Kind == ast.KindParenthesizedExpression || p.Kind == ast.KindNonNullExpression {
		c, p = p, p.Parent
	}
	switch {
	case (p.Kind == ast.KindPropertyAccessExpression || p.Kind == ast.KindElementAccessExpression) && p.Expression() == c:
		return true
	case p.Kind == ast.KindReturnStatement:
		return true
	case p.Kind == ast.KindCallExpression && slices.Contains(p.Arguments(), c):
		fn := m.followed(p)
		if fn == nil {
			return false
		}
		summary := m.Summary(fn)
		position := slices.Index(p.Arguments(), c)
		return !summary.Opaque && !slices.ContainsFunc(summary.Effects, func(e SummaryEffect) bool { return slices.Contains(e.Carried, position) })
	}
	return false
}
