package model

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/bundled"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/compiler"
	"github.com/microsoft/typescript-go/shim/vfs/osvfs"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

func summaryProgram(t *testing.T, source string) (*compiler.Program, *ast.SourceFile) {
	t.Helper()
	root := t.TempDir()
	for name, text := range map[string]string{
		"tsconfig.json": `{"compilerOptions":{"strict":true,"noLib":true,"types":[]},"files":["shared.ts"]}`,
		"shared.ts":     source,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	fs := bundled.WrapFS(osvfs.FS())
	program, diagnostics, err := utils.CreateProgram(true, fs, root, "tsconfig.json", utils.CreateCompilerHost(root, fs), true)
	if err != nil || program == nil || len(diagnostics) != 0 {
		t.Fatalf("create program: %v, %v", err, diagnostics)
	}
	return program, program.GetSourceFile(filepath.Join(root, "shared.ts"))
}

func forEachChecker(t *testing.T, program *compiler.Program, file *ast.SourceFile, check func(rule.RuleContext)) {
	t.Helper()
	var checked atomic.Int64
	program.ForEachCheckerParallel(func(_ int, tc *checker.Checker) {
		checked.Add(1)
		check(rule.RuleContext{Program: program, TypeChecker: tc, SourceFile: file})
	})
	if checked.Load() == 0 {
		t.Fatal("no checker ran")
	}
}

func TestSummarySharedAcrossAnalyzersOfOneChecker(t *testing.T) {
	program, file := summaryProgram(t, "export function set(x: {left: number}) { x.left = 1; }\n")
	fn := file.Statements.Nodes[0]
	forEachChecker(t, program, file, func(ctx rule.RuleContext) {
		first := NewAnalyzer(ctx).Summary(fn)
		second := NewAnalyzer(ctx).Summary(fn)
		if len(first.Effects) != 1 || len(second.Effects) != 1 || &first.Effects[0] != &second.Effects[0] {
			t.Errorf("summary was not shared: %+v, %+v", first, second)
		}
	})
}

func TestRecursiveSummaryStaysWithItsAnalyzer(t *testing.T) {
	program, file := summaryProgram(t, `
export function first(x: {left: number; right: number}) { second(x); x.left = 1; }
export function second(x: {left: number; right: number}) { first(x); x.right = 2; }
`)
	forEachChecker(t, program, file, func(ctx rule.RuleContext) {
		NewAnalyzer(ctx).Summary(file.Statements.Nodes[0])
		if got := NewAnalyzer(ctx).Summary(file.Statements.Nodes[1]); len(got.Effects) != 2 {
			t.Errorf("a summary read during another function's cycle reached a new analyzer: %+v", got)
		}
	})
}
