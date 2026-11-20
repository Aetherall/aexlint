package model

import (
	"runtime"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/checker"
)

var checkerSummaries sync.Map

// summariesOf returns the function summaries shared by analyzers using tc, keyed weakly by the checker and released
// when it is collected.
func summariesOf(tc *checker.Checker) *sync.Map {
	if tc == nil {
		return nil
	}
	key := weak.Make(tc)
	if found, ok := checkerSummaries.Load(key); ok {
		return found.(*sync.Map)
	}
	found, loaded := checkerSummaries.LoadOrStore(key, &sync.Map{})
	if !loaded {
		runtime.AddCleanup(tc, func(key weak.Pointer[checker.Checker]) { checkerSummaries.Delete(key) }, key)
	}
	return found.(*sync.Map)
}
