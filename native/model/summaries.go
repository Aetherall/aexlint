package model

import (
	"runtime"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

var sharedByChecker sync.Map

type sharedKey struct {
	checker weak.Pointer[checker.Checker]
	program weak.Pointer[compiler.Program]
}

// sharedFacts holds the facts analyzers using one checker for one program share: function summaries, classified sites
// and the classes derived from each class. A checker can serve several programs' files, and these facts depend on the
// program through the subclasses and callers it sees.
type sharedFacts struct {
	summaries   sync.Map
	sites       sync.Map
	derived     sync.Once
	derivedFrom map[*ast.Node][]*ast.Node
}

// sharedFactsOf returns the facts shared by analyzers using tc for program, keyed weakly and released when the checker
// is collected.
func sharedFactsOf(tc *checker.Checker, program *compiler.Program) *sharedFacts {
	if tc == nil || program == nil {
		return nil
	}
	key := sharedKey{weak.Make(tc), weak.Make(program)}
	if found, ok := sharedByChecker.Load(key); ok {
		return found.(*sharedFacts)
	}
	found, loaded := sharedByChecker.LoadOrStore(key, &sharedFacts{})
	if !loaded {
		runtime.AddCleanup(tc, func(key sharedKey) { sharedByChecker.Delete(key) }, key)
	}
	return found.(*sharedFacts)
}
