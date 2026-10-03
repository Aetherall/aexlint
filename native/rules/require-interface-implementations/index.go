package require_interface_implementations

import (
	"runtime"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// ClassIndex lists every class declaration and class expression with a heritage clause in a program's project source
// files. It is built once per program from syntax alone and never mutated afterwards.
type ClassIndex struct {
	once    sync.Once
	classes []*ast.Node
}

var (
	indexesMu sync.Mutex
	indexes   = map[weak.Pointer[compiler.Program]]*ClassIndex{}
)

// IndexFor returns the program's shared index. Entries are keyed weakly and removed when the program is collected.
func IndexFor(program *compiler.Program) *ClassIndex {
	key := weak.Make(program)
	indexesMu.Lock()
	index, ok := indexes[key]
	if !ok {
		index = &ClassIndex{}
		indexes[key] = index
		runtime.AddCleanup(program, func(key weak.Pointer[compiler.Program]) {
			indexesMu.Lock()
			delete(indexes, key)
			indexesMu.Unlock()
		}, key)
	}
	indexesMu.Unlock()
	index.once.Do(func() { index.build(program) })
	return index
}

func (index *ClassIndex) build(program *compiler.Program) {
	files := []*ast.SourceFile{}
	for _, file := range program.SourceFiles() {
		if !file.IsDeclarationFile && !program.IsSourceFileFromExternalLibrary(file) {
			files = append(files, file)
		}
	}
	partial := make([][]*ast.Node, len(files))
	workers := min(runtime.GOMAXPROCS(0), max(len(files), 1))
	var group sync.WaitGroup
	for worker := range workers {
		group.Go(func() {
			for i := worker; i < len(files); i += workers {
				nodes := []*ast.Node{}
				var walk func(*ast.Node) bool
				walk = func(n *ast.Node) bool {
					if ast.IsClassLike(n) && len(ast.GetExtendsHeritageClauseElements(n))+len(ast.GetImplementsHeritageClauseElements(n)) > 0 {
						nodes = append(nodes, n)
					}
					n.ForEachChild(walk)
					return false
				}
				walk(files[i].AsNode())
				partial[i] = nodes
			}
		})
	}
	group.Wait()
	for _, nodes := range partial {
		index.classes = append(index.classes, nodes...)
	}
}

// Classes returns the indexed classes. Callers must not modify the result.
func (index *ClassIndex) Classes() []*ast.Node { return index.classes }
