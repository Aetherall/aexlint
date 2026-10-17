package model

import (
	"testing"

	"github.com/microsoft/typescript-go/shim/compiler"
)

func TestSharedRepeatsBuildFailure(t *testing.T) {
	shared := NewShared(func(*compiler.Program) *int { panic("broken index") })
	program := new(compiler.Program)
	for attempt := range 2 {
		func() {
			defer func() {
				if failure := recover(); failure != "broken index" {
					t.Fatalf("attempt %d: got %v, want the build failure", attempt, failure)
				}
			}()
			shared.For(program)
		}()
	}
}
