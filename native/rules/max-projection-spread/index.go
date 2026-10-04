package max_projection_spread

import (
	"runtime"
	"strings"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// DerivationIndex lists every candidate derivation in a program's project source files, by file and by syntactic
// fingerprint. It is built once per program from syntax alone and never mutated afterwards.
type DerivationIndex struct {
	once   sync.Once
	byFile map[*ast.SourceFile][]*ast.Node
	byKey  map[string][]*ast.Node
}

var (
	indexesMu sync.Mutex
	indexes   = map[weak.Pointer[compiler.Program]]*DerivationIndex{}
)

// IndexFor returns the program's shared index. Entries are keyed weakly and removed when the program is collected.
func IndexFor(program *compiler.Program) *DerivationIndex {
	key := weak.Make(program)
	indexesMu.Lock()
	index, ok := indexes[key]
	if !ok {
		index = &DerivationIndex{}
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

func skipOuter(n *ast.Node) *ast.Node {
	for {
		switch n.Kind {
		case ast.KindParenthesizedExpression, ast.KindAsExpression, ast.KindSatisfiesExpression, ast.KindNonNullExpression:
			n = n.Expression()
		default:
			return n
		}
	}
}

func literal(n *ast.Node) bool {
	switch skipOuter(n).Kind {
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral, ast.KindNumericLiteral, ast.KindTrueKeyword,
		ast.KindFalseKeyword, ast.KindNullKeyword:
		return true
	}
	return false
}

// literalText returns a literal argument's text; keywords have no Text in the AST.
func literalText(n *ast.Node) string {
	n = skipOuter(n)
	switch n.Kind {
	case ast.KindTrueKeyword:
		return "true"
	case ast.KindFalseKeyword:
		return "false"
	case ast.KindNullKeyword:
		return "null"
	case ast.KindStringLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return "s:" + n.Text()
	}
	return "n:" + n.Text()
}

// Chain is a derivation's value: its root identifier and, from the root outwards, the property accesses and calls
// applied to it.
type Chain struct {
	Root  *ast.Node
	Steps []*ast.Node
}

// ChainOf returns the chain of a derivation's value: an identifier followed by property accesses and method calls
// with literal or no arguments, including at least one call.
func ChainOf(value *ast.Node) (Chain, bool) {
	steps := []*ast.Node{}
	called := false
	n := skipOuter(value)
	for {
		switch n.Kind {
		case ast.KindCallExpression:
			for _, argument := range n.Arguments() {
				if !literal(argument) {
					return Chain{}, false
				}
			}
			if skipOuter(n.Expression()).Kind != ast.KindPropertyAccessExpression {
				return Chain{}, false
			}
			called = true
			steps = append(steps, n)
			n = skipOuter(n.Expression())
		case ast.KindPropertyAccessExpression:
			steps = append(steps, n)
			n = skipOuter(n.Expression())
		case ast.KindIdentifier:
			if !called {
				return Chain{}, false
			}
			for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
				steps[i], steps[j] = steps[j], steps[i]
			}
			return Chain{Root: n, Steps: steps}, true
		default:
			return Chain{}, false
		}
	}
}

// KeyOf returns a property assignment's static key.
func KeyOf(property *ast.Node) (string, bool) {
	if property.Kind != ast.KindPropertyAssignment {
		return "", false
	}
	name := property.Name()
	switch name.Kind {
	case ast.KindIdentifier, ast.KindStringLiteral, ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral:
		return name.Text(), true
	}
	return "", false
}

// Fingerprint describes a chain's text: member names and literal arguments, from the root outwards.
func Fingerprint(chain Chain) string {
	parts := []string{}
	for _, step := range chain.Steps {
		if step.Kind == ast.KindPropertyAccessExpression {
			parts = append(parts, step.Name().Text())
			continue
		}
		arguments := []string{}
		for _, argument := range step.Arguments() {
			arguments = append(arguments, literalText(argument))
		}
		parts = append(parts, "("+strings.Join(arguments, ",")+")")
	}
	return strings.Join(parts, ".")
}

// keyOf returns the index key of a candidate derivation: its property key and chain fingerprint.
func keyOf(property *ast.Node) (string, bool) {
	key, ok := KeyOf(property)
	if !ok {
		return "", false
	}
	chain, ok := ChainOf(property.Initializer())
	if !ok {
		return "", false
	}
	return key + "=" + Fingerprint(chain), true
}

func (index *DerivationIndex) build(program *compiler.Program) {
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
					if _, ok := keyOf(n); ok {
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
	index.byFile = map[*ast.SourceFile][]*ast.Node{}
	index.byKey = map[string][]*ast.Node{}
	for i, file := range files {
		index.byFile[file] = partial[i]
		for _, n := range partial[i] {
			key, _ := keyOf(n)
			index.byKey[key] = append(index.byKey[key], n)
		}
	}
}

// File returns the candidate derivations in file. Callers must not modify the result.
func (index *DerivationIndex) File(file *ast.SourceFile) []*ast.Node { return index.byFile[file] }

// Keyed returns the candidate derivations with the index key of property. Callers must not modify the result.
func (index *DerivationIndex) Keyed(property *ast.Node) []*ast.Node {
	key, _ := keyOf(property)
	return index.byKey[key]
}
