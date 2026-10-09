package no_repeated_value_group

import (
	"runtime"
	"strconv"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// Index lists the roots of a program's project source files by file, by the literal values they mention and by the outcome text they lead to.
// Roots mentioning no literal token are listed in other. It is built once per program from syntax alone and never mutated afterwards.
type Index struct {
	once      sync.Once
	byFile    map[*ast.SourceFile][]*ast.Node
	byKey     map[string][]*ast.Node
	byOutcome map[string][]*ast.Node
	other     []*ast.Node
}

var (
	indexesMu sync.Mutex
	indexes   = map[weak.Pointer[compiler.Program]]*Index{}
)

// IndexFor returns the program's shared index. Entries are keyed weakly and removed when the program is collected.
func IndexFor(program *compiler.Program) *Index {
	key := weak.Make(program)
	indexesMu.Lock()
	index, ok := indexes[key]
	if !ok {
		index = &Index{}
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

func tokenKey(n *ast.Node) (string, bool) {
	switch n.Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return "s:" + n.Text(), true
	case ast.KindNumericLiteral:
		if value, err := strconv.ParseFloat(n.Text(), 64); err == nil {
			return NumberKey(value), true
		}
	case ast.KindPrefixUnaryExpression:
		if unary := n.AsPrefixUnaryExpression(); unary.Operator == ast.KindMinusToken && unary.Operand.Kind == ast.KindNumericLiteral {
			if value, err := strconv.ParseFloat(unary.Operand.Text(), 64); err == nil {
				return NumberKey(-value), true
			}
		}
	}
	return "", false
}

func keys(root *ast.Node) []string {
	scope := []*ast.Node{root}
	if root.Kind == ast.KindCaseClause {
		scope = nil
		clauses := root.Parent.AsCaseBlock().Clauses.Nodes
		for _, clause := range clauses[indexOf(clauses, root):] {
			if clause.Kind == ast.KindDefaultClause {
				break
			}
			scope = append(scope, clause.Expression())
			if len(clause.Statements()) > 0 {
				break
			}
		}
	}
	found := map[string]bool{}
	result := []string{}
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if key, ok := tokenKey(n); ok {
			if !found[key] {
				found[key] = true
				result = append(result, key)
			}
			return false
		}
		n.ForEachChild(walk)
		return false
	}
	for _, n := range scope {
		walk(n)
	}
	return result
}

func (index *Index) build(program *compiler.Program) {
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
				roots := []*ast.Node{}
				var walk func(*ast.Node) bool
				walk = func(n *ast.Node) bool {
					if Root(n) {
						roots = append(roots, n)
					}
					n.ForEachChild(walk)
					return false
				}
				walk(files[i].AsNode())
				partial[i] = roots
			}
		})
	}
	group.Wait()
	index.byFile = map[*ast.SourceFile][]*ast.Node{}
	index.byKey = map[string][]*ast.Node{}
	index.byOutcome = map[string][]*ast.Node{}
	for i, file := range files {
		index.byFile[file] = partial[i]
		for _, root := range partial[i] {
			found := keys(root)
			if len(found) == 0 {
				index.other = append(index.other, root)
			}
			for _, key := range found {
				index.byKey[key] = append(index.byKey[key], root)
			}
			if outcome := Outcome(root); outcome != "" {
				index.byOutcome[outcome] = append(index.byOutcome[outcome], root)
			}
		}
	}
}

// File returns the roots of file. Callers must not modify the result.
func (index *Index) File(file *ast.SourceFile) []*ast.Node { return index.byFile[file] }

// Keyed returns the roots mentioning the literal value key. Callers must not modify the result.
func (index *Index) Keyed(key string) []*ast.Node { return index.byKey[key] }

// Outcome returns the roots leading to the outcome text. Callers must not modify the result.
func (index *Index) Outcome(outcome string) []*ast.Node { return index.byOutcome[outcome] }

// Other returns the roots mentioning no literal token. Callers must not modify the result.
func (index *Index) Other() []*ast.Node { return index.other }
