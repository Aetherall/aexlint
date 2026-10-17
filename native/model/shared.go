package model

import (
	"runtime"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// Shared is a value built once per program from syntax alone and never mutated afterwards. Values are keyed weakly
// and released when their program is collected.
type Shared[T any] struct {
	build   func(*compiler.Program) *T
	mu      sync.Mutex
	entries map[weak.Pointer[compiler.Program]]*sharedEntry[T]
}

type sharedEntry[T any] struct {
	once    sync.Once
	value   *T
	failure any
}

func NewShared[T any](build func(*compiler.Program) *T) *Shared[T] {
	return &Shared[T]{build: build, entries: map[weak.Pointer[compiler.Program]]*sharedEntry[T]{}}
}

// For returns the program's value, building it on first use. A build that panics panics again for every caller.
func (s *Shared[T]) For(program *compiler.Program) *T {
	key := weak.Make(program)
	s.mu.Lock()
	entry, ok := s.entries[key]
	if !ok {
		entry = &sharedEntry[T]{}
		s.entries[key] = entry
		runtime.AddCleanup(program, func(key weak.Pointer[compiler.Program]) {
			s.mu.Lock()
			delete(s.entries, key)
			s.mu.Unlock()
		}, key)
	}
	s.mu.Unlock()
	entry.once.Do(func() {
		defer func() { entry.failure = recover() }()
		entry.value = s.build(program)
	})
	if entry.failure != nil {
		panic(entry.failure)
	}
	return entry.value
}

// Collect walks a program's project files in parallel and returns them in program order, each with the nodes keep
// accepts in source order.
func Collect(program *compiler.Program, keep func(*ast.Node) bool) ([]*ast.SourceFile, [][]*ast.Node) {
	files := []*ast.SourceFile{}
	for _, file := range program.SourceFiles() {
		if ProjectFile(file) {
			files = append(files, file)
		}
	}
	found := make([][]*ast.Node, len(files))
	workers := min(runtime.GOMAXPROCS(0), max(len(files), 1))
	var group sync.WaitGroup
	for worker := range workers {
		group.Go(func() {
			for i := worker; i < len(files); i += workers {
				nodes := []*ast.Node{}
				var walk func(*ast.Node) bool
				walk = func(n *ast.Node) bool {
					if keep(n) {
						nodes = append(nodes, n)
					}
					n.ForEachChild(walk)
					return false
				}
				walk(files[i].AsNode())
				found[i] = nodes
			}
		})
	}
	group.Wait()
	return files, found
}
