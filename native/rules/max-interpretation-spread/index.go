package max_interpretation_spread

import (
	"runtime"
	"strconv"
	"sync"
	"weak"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/compiler"
)

// BranchIndex lists every equality comparison and switch statement in a program's project source files, by file,
// and by the parsed literal values they compare against. Branches that also compare against anything else, such as
// constants, enum members, or object members, are listed in other. It is built once per program from syntax alone and never mutated afterwards.
type BranchIndex struct {
	once   sync.Once
	byFile map[*ast.SourceFile][]*ast.Node
	byKey  map[string][]*ast.Node
	other  []*ast.Node
	files  []*ast.SourceFile
	size   int
}

func NumberKey(value float64) string { return "n:" + strconv.FormatFloat(value, 'g', -1, 64) }

// keyOf returns the index key of a compared operand that is a literal token: its parsed value.
func keyOf(n *ast.Node) (string, bool) {
	for n.Kind == ast.KindParenthesizedExpression {
		n = n.Expression()
	}
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

// keysOf returns the keys a branch compares against, or false when one of its compared operands has no key.
func keysOf(n *ast.Node) ([]string, bool) {
	if n.Kind == ast.KindSwitchStatement {
		keys := []string{}
		for _, clause := range n.AsSwitchStatement().CaseBlock.AsCaseBlock().Clauses.Nodes {
			if clause.Kind == ast.KindDefaultClause {
				continue
			}
			key, ok := keyOf(clause.Expression())
			if !ok {
				return nil, false
			}
			keys = append(keys, key)
		}
		return keys, len(keys) > 0
	}
	// Either operand of an equality may be the compared unit; only a literal token identifies its value without a checker.
	binary := n.AsBinaryExpression()
	keys := []string{}
	for _, side := range []*ast.Node{binary.Left, binary.Right} {
		if key, ok := keyOf(side); ok {
			keys = append(keys, key)
		}
	}
	return keys, len(keys) > 0
}

var (
	indexesMu sync.Mutex
	indexes   = map[weak.Pointer[compiler.Program]]*BranchIndex{}
)

// IndexFor returns the program's shared index. Entries are keyed weakly and removed when the program is collected.
func IndexFor(program *compiler.Program) *BranchIndex {
	key := weak.Make(program)
	indexesMu.Lock()
	index, ok := indexes[key]
	if !ok {
		index = &BranchIndex{}
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

// equality reports whether n is an equality comparison that can compare a value with a literal: neither operand is
// null, true, false, or a typeof expression, none of which has a string or number literal type.
func equality(n *ast.Node) bool {
	if n.Kind != ast.KindBinaryExpression {
		return false
	}
	binary := n.AsBinaryExpression()
	switch binary.OperatorToken.Kind {
	case ast.KindEqualsEqualsEqualsToken, ast.KindExclamationEqualsEqualsToken, ast.KindEqualsEqualsToken, ast.KindExclamationEqualsToken:
	default:
		return false
	}
	for _, side := range []*ast.Node{binary.Left, binary.Right} {
		for side.Kind == ast.KindParenthesizedExpression {
			side = side.Expression()
		}
		switch side.Kind {
		case ast.KindNullKeyword, ast.KindTrueKeyword, ast.KindFalseKeyword, ast.KindTypeOfExpression:
			return false
		}
	}
	return true
}

func (index *BranchIndex) build(program *compiler.Program) {
	for _, file := range program.SourceFiles() {
		if !file.IsDeclarationFile && !program.IsSourceFileFromExternalLibrary(file) {
			index.files = append(index.files, file)
		}
	}
	partial := make([][]*ast.Node, len(index.files))
	workers := min(runtime.GOMAXPROCS(0), max(len(index.files), 1))
	var group sync.WaitGroup
	for worker := range workers {
		group.Go(func() {
			for i := worker; i < len(index.files); i += workers {
				nodes := []*ast.Node{}
				var walk func(*ast.Node) bool
				walk = func(n *ast.Node) bool {
					if equality(n) || n.Kind == ast.KindSwitchStatement {
						nodes = append(nodes, n)
					}
					n.ForEachChild(walk)
					return false
				}
				walk(index.files[i].AsNode())
				partial[i] = nodes
			}
		})
	}
	group.Wait()
	index.byFile = map[*ast.SourceFile][]*ast.Node{}
	index.byKey = map[string][]*ast.Node{}
	for i, file := range index.files {
		index.byFile[file] = partial[i]
		index.size += len(partial[i])
		for _, n := range partial[i] {
			keys, ok := keysOf(n)
			if !ok {
				index.other = append(index.other, n)
				continue
			}
			for _, key := range keys {
				index.byKey[key] = append(index.byKey[key], n)
			}
		}
	}
}

// File returns the branches in file. Callers must not modify the result.
func (index *BranchIndex) File(file *ast.SourceFile) []*ast.Node { return index.byFile[file] }

// Keyed returns the branches that compare against the literal value key. Callers must not modify the result.
func (index *BranchIndex) Keyed(key string) []*ast.Node { return index.byKey[key] }

// Other returns the branches that compare against an operand other than a literal token. Callers must not modify the result.
func (index *BranchIndex) Other() []*ast.Node { return index.other }

// Files returns the indexed project source files. Callers must not modify the result.
func (index *BranchIndex) Files() []*ast.SourceFile { return index.files }

// Len returns the number of indexed branches.
func (index *BranchIndex) Len() int { return index.size }
