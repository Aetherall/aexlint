package model

import (
	"runtime"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
)

var checkerSummaries sync.Map

type summariesKey struct {
	checker weak.Pointer[checker.Checker]
	program weak.Pointer[compiler.Program]
}

// summariesOf returns the function summaries shared by analyzers using tc for program, keyed weakly and released when
// the checker is collected. A checker can serve several programs' files, and a summary depends on the program through
// the subclasses it sees.
func summariesOf(tc *checker.Checker, program *compiler.Program) *sync.Map {
	if tc == nil || program == nil {
		return nil
	}
	key := summariesKey{weak.Make(tc), weak.Make(program)}
	if found, ok := checkerSummaries.Load(key); ok {
		return found.(*sync.Map)
	}
	found, loaded := checkerSummaries.LoadOrStore(key, &sync.Map{})
	if !loaded {
		runtime.AddCleanup(tc, func(key summariesKey) { checkerSummaries.Delete(key) }, key)
	}
	return found.(*sync.Map)
}
