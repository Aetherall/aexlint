package no_repeated_conversion

import (
	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/model"
)

// shapes lists the object literals of a program's project files by shape. It is built once per program from syntax
// alone and never mutated afterwards.
type shapes struct {
	byShape map[string][]*ast.Node
}

var indexes = model.NewShared(buildShapes)

func shapesFor(program *compiler.Program) *shapes { return indexes.For(program) }

func buildShapes(program *compiler.Program) *shapes {
	index := &shapes{}
	_, partial := model.Collect(program, func(n *ast.Node) bool { return n.Kind == ast.KindObjectLiteralExpression })
	index.byShape = map[string][]*ast.Node{}
	for _, literals := range partial {
		for _, literal := range literals {
			shape := model.Shape(literal)
			index.byShape[shape] = append(index.byShape[shape], literal)
		}
	}
	return index
}

// shaped returns the object literals with the given shape. Callers must not modify the result.
func (index *shapes) shaped(shape string) []*ast.Node { return index.byShape[shape] }
