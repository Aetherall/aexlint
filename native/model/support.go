package model

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// ParseOptions accepts omitted options, null and {}, and rejects anything else.
// ParseOptions validates the options of a rule that takes none: omitted, null, or an empty object.
func ParseOptions(name string, options any) {
	if options == nil {
		return
	}
	if object, ok := options.(map[string]any); ok && len(object) == 0 {
		return
	}
	panic(fmt.Sprintf("aexlint/%s accepts no options; omit options or use an empty object", name))
}

// ParseMax reads the options of a rule with one optional limit, max, a positive safe integer.
func ParseMax(name string, options any, fallback int) int {
	return ParseLimit(name, options, fallback, 1)
}

// ParseLimit reads the options of a rule with one optional limit, max, a safe integer of at least minimum.
func ParseLimit(name string, options any, fallback, minimum int) int {
	kind := "a positive"
	if minimum == 0 {
		kind = "a non-negative"
	}
	if options == nil {
		return fallback
	}
	object, ok := options.(map[string]any)
	if !ok {
		panic(fmt.Sprintf(`aexlint/%s accepts an optional options object: { "max"?: <%s integer> }`, name, kind[2:]))
	}
	for key := range object {
		if key != "max" {
			panic(fmt.Sprintf("aexlint/%s: unknown option %q; only \"max\" is accepted", name, key))
		}
	}
	raw, present := object["max"]
	if !present {
		return fallback
	}
	value, ok := raw.(float64)
	if !ok || value < float64(minimum) || value > 1<<53-1 || value != float64(int64(value)) {
		panic(fmt.Sprintf(`aexlint/%s: "max" must be %s safe integer`, name, kind))
	}
	return int(value)
}

// Places describes sites relative to the linted file: same file first, then by shared leading directories, then by path and position.
type Places struct {
	ctx rule.RuleContext
}

func NewPlaces(ctx rule.RuleContext) Places { return Places{ctx: ctx} }

// Relative returns fileName relative to the process's working directory, the configuration directory under aexlint-typed, falling back to the program's directory.
func (p Places) Relative(fileName string) string {
	directory, err := os.Getwd()
	if err != nil {
		directory = p.ctx.Program.GetCurrentDirectory()
	}
	directory = tspath.NormalizePath(directory)
	return tspath.GetRelativePathFromDirectory(directory, fileName, tspath.ComparePathsOptions{
		UseCaseSensitiveFileNames: p.ctx.Program.UseCaseSensitiveFileNames(),
		CurrentDirectory:          directory,
	})
}

func (p Places) shared(file *ast.SourceFile) int {
	own := strings.Split(tspath.GetDirectoryPath(p.ctx.SourceFile.FileName()), "/")
	directories := strings.Split(tspath.GetDirectoryPath(file.FileName()), "/")
	count := 0
	for count < len(own) && count < len(directories) && own[count] == directories[count] {
		count++
	}
	if file == p.ctx.SourceFile {
		count = len(own) + 1
	}
	return count
}

// Sort orders nodes nearest first.
func (p Places) Sort(nodes []*ast.Node) {
	slices.SortFunc(nodes, func(a, b *ast.Node) int {
		fa, fb := ast.GetSourceFileOfNode(a), ast.GetSourceFileOfNode(b)
		if byShared := p.shared(fb) - p.shared(fa); byShared != 0 {
			return byShared
		}
		if byName := strings.Compare(fa.FileName(), fb.FileName()); byName != 0 {
			return byName
		}
		return a.Pos() - b.Pos()
	})
}

// Nearest lists files other than the linted one, nearest first.
func (p Places) Nearest(files map[*ast.SourceFile]bool) []string {
	others := []*ast.SourceFile{}
	for file := range files {
		if file != p.ctx.SourceFile {
			others = append(others, file)
		}
	}
	slices.SortFunc(others, func(a, b *ast.SourceFile) int {
		if byShared := p.shared(b) - p.shared(a); byShared != 0 {
			return byShared
		}
		return strings.Compare(a.FileName(), b.FileName())
	})
	result := []string{}
	for _, file := range others {
		result = append(result, p.Relative(file.FileName()))
	}
	return result
}

// Location renders a node as path:line.
func (p Places) Location(n *ast.Node) string {
	file := ast.GetSourceFileOfNode(n)
	line := scanner.GetECMALineOfPosition(file, scanner.SkipTrivia(file.Text(), n.Pos())) + 1
	return fmt.Sprintf("%s:%d", p.Relative(file.FileName()), line)
}

// Listed joins items, showing at most shown of them.
func Listed(items []string, shown int) string {
	if len(items) <= shown {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(items[:shown], ", "), len(items)-shown)
}

func ProjectFile(file *ast.SourceFile) bool {
	return !file.IsDeclarationFile && !strings.Contains(file.FileName(), "/node_modules/")
}

func ProjectSymbol(symbol *ast.Symbol) bool {
	if symbol == nil || len(symbol.Declarations) == 0 {
		return false
	}
	for _, declaration := range symbol.Declarations {
		if !ProjectFile(ast.GetSourceFileOfNode(declaration)) {
			return false
		}
	}
	return true
}
