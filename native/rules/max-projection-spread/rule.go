package max_projection_spread

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/utils"
)

const maxSafeInteger = 1<<53 - 1

const defaultMax = 2

func parseOptions(options any) int {
	if options == nil {
		return defaultMax
	}
	object, ok := options.(map[string]any)
	if !ok {
		panic(`aexlint/max-projection-spread accepts an optional options object: { "max"?: <positive integer> }`)
	}
	for key := range object {
		if key != "max" {
			panic(fmt.Sprintf("aexlint/max-projection-spread: unknown option %q; only \"max\" is accepted", key))
		}
	}
	raw, present := object["max"]
	if !present {
		return defaultMax
	}
	value, ok := raw.(float64)
	if !ok || value < 1 || value > maxSafeInteger || value != float64(int64(value)) {
		panic(`aexlint/max-projection-spread: "max" must be a positive safe integer`)
	}
	return int(value)
}

// derivation is a property of an object literal that derives its value from a member chain of an owned value.
type derivation struct {
	property *ast.Node
	key      string
	root     *ast.Symbol
	owner    *ast.Symbol
	// identity is equal for derivations that restate the same derivation: same key, owner, member declarations, and
	// literal arguments.
	identity string
}

type measure struct {
	ctx         rule.RuleContext
	derivations map[*ast.Node]*derivation
	projections map[*ast.Node]bool
}

func (m *measure) owned(symbol *ast.Symbol) bool {
	if symbol == nil || len(symbol.Declarations) == 0 {
		return false
	}
	for _, declaration := range symbol.Declarations {
		file := ast.GetSourceFileOfNode(declaration)
		if file.IsDeclarationFile || m.ctx.Program.IsSourceFileFromExternalLibrary(file) {
			return false
		}
	}
	return true
}

// ownerOf returns the owner of a chain's root: the alias, otherwise the symbol, of its non-nullable declared type,
// when the project declares it.
func (m *measure) ownerOf(root *ast.Node) *ast.Symbol {
	t := m.ctx.TypeChecker.GetTypeAtLocation(root)
	if t == nil {
		return nil
	}
	t = m.ctx.TypeChecker.GetNonNullableType(t)
	if t.Flags()&checker.TypeFlagsObject == 0 {
		return nil
	}
	symbol := t.Symbol()
	if alias := t.Alias(); alias != nil {
		symbol = alias.Symbol()
	}
	if !m.owned(symbol) {
		return nil
	}
	return symbol
}

// derive returns the derivation a property assignment makes, or nil.
func (m *measure) derive(property *ast.Node) *derivation {
	if d, ok := m.derivations[property]; ok {
		return d
	}
	m.derivations[property] = nil
	key, ok := KeyOf(property)
	if !ok {
		return nil
	}
	chain, ok := ChainOf(property.Initializer())
	if !ok {
		return nil
	}
	root := m.ctx.TypeChecker.GetSymbolAtLocation(chain.Root)
	owner := m.ownerOf(chain.Root)
	if root == nil || owner == nil {
		return nil
	}
	parts := []string{key, fmt.Sprintf("%p", owner.Declarations[0])}
	for _, step := range chain.Steps {
		if step.Kind != ast.KindPropertyAccessExpression {
			continue
		}
		member := m.ctx.TypeChecker.GetSymbolAtLocation(step.Name())
		if member == nil || len(member.Declarations) == 0 {
			return nil
		}
		parts = append(parts, fmt.Sprintf("%p", member.Declarations[0]))
	}
	parts = append(parts, Fingerprint(chain))
	d := &derivation{property: property, key: key, root: root, owner: owner, identity: strings.Join(parts, "|")}
	m.derivations[property] = d
	return d
}

// projected reports whether a derivation belongs to a projection: its object literal has another derivation from the
// same variable.
func (m *measure) projected(d *derivation) bool {
	literal := d.property.Parent
	if done, ok := m.projections[d.property]; ok {
		return done
	}
	count := 0
	for _, property := range literal.AsObjectLiteralExpression().Properties.Nodes {
		if other := m.derive(property); other != nil && other.root == d.root && other.owner == d.owner {
			count++
		}
	}
	m.projections[d.property] = count >= 2
	return count >= 2
}

// relative returns fileName relative to the process's working directory, the configuration directory under aexlint-typed, falling back to the program's directory.
func (m *measure) relative(fileName string) string {
	directory, err := os.Getwd()
	if err != nil {
		directory = m.ctx.Program.GetCurrentDirectory()
	}
	directory = tspath.NormalizePath(directory)
	return tspath.GetRelativePathFromDirectory(directory, fileName, tspath.ComparePathsOptions{
		UseCaseSensitiveFileNames: m.ctx.Program.UseCaseSensitiveFileNames(),
		CurrentDirectory:          directory,
	})
}

// nearest orders files other than the linted one by how many leading directories they share with it, then by path.
func (m *measure) nearest(files map[*ast.SourceFile]bool) []string {
	own := strings.Split(tspath.GetDirectoryPath(m.ctx.SourceFile.FileName()), "/")
	shared := func(file *ast.SourceFile) int {
		directories := strings.Split(tspath.GetDirectoryPath(file.FileName()), "/")
		count := 0
		for count < len(own) && count < len(directories) && own[count] == directories[count] {
			count++
		}
		return count
	}
	others := []*ast.SourceFile{}
	for file := range files {
		if file != m.ctx.SourceFile {
			others = append(others, file)
		}
	}
	slices.SortFunc(others, func(a, b *ast.SourceFile) int {
		if byShared := shared(b) - shared(a); byShared != 0 {
			return byShared
		}
		return strings.Compare(a.FileName(), b.FileName())
	})
	result := []string{}
	for _, file := range others {
		result = append(result, m.relative(file.FileName()))
	}
	return result
}

// listed joins items, showing at most shown of them.
func listed(items []string, shown int) string {
	if len(items) <= shown {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(items[:shown], ", "), len(items)-shown)
}

// chainText returns a derivation's value as written, without its root.
func (m *measure) chainText(d *derivation) string {
	value := skipOuter(d.property.Initializer())
	text := m.ctx.SourceFile.Text()[utils.TrimNodeTextRange(m.ctx.SourceFile, value).Pos():value.End()]
	chain, _ := ChainOf(value)
	text = strings.TrimPrefix(text, chain.Root.Text())
	text = strings.TrimPrefix(text, "!")
	text = strings.TrimPrefix(text, "?")
	return strings.TrimPrefix(text, ".")
}

type spread struct {
	d     *derivation
	files map[*ast.SourceFile]bool
}

func (m *measure) run(limit int) {
	index := IndexFor(m.ctx.Program)
	literals := []*ast.Node{}
	local := map[*ast.Node][]*derivation{}
	for _, property := range index.File(m.ctx.SourceFile) {
		d := m.derive(property)
		if d == nil || !m.projected(d) {
			continue
		}
		literal := property.Parent
		if local[literal] == nil {
			literals = append(literals, literal)
		}
		local[literal] = append(local[literal], d)
	}
	counted := map[string]map[*ast.SourceFile]bool{}
	for _, literal := range literals {
		for _, d := range local[literal] {
			if counted[d.identity] != nil {
				continue
			}
			files := map[*ast.SourceFile]bool{}
			for _, candidate := range index.Keyed(d.property) {
				if other := m.derive(candidate); other != nil && other.identity == d.identity && m.projected(other) {
					files[ast.GetSourceFileOfNode(candidate)] = true
				}
			}
			counted[d.identity] = files
		}
	}
	for _, literal := range literals {
		spreads := []spread{}
		for _, d := range local[literal] {
			if files := counted[d.identity]; len(files) > limit {
				spreads = append(spreads, spread{d: d, files: files})
			}
		}
		if len(spreads) == 0 {
			continue
		}
		m.report(spreads, limit)
	}
}

func (m *measure) report(spreads []spread, limit int) {
	first := spreads[0].d
	slices.SortStableFunc(spreads, func(a, b spread) int {
		if byFiles := len(b.files) - len(a.files); byFiles != 0 {
			return byFiles
		}
		return strings.Compare(a.d.key, b.d.key)
	})
	described := []string{}
	files := map[*ast.SourceFile]bool{}
	for _, s := range spreads {
		described = append(described, fmt.Sprintf("`%s` (`%s`) in %d files", s.d.key, m.chainText(s.d), len(s.files)))
		for file := range s.files {
			files[file] = true
		}
	}
	count := "1 derivation"
	if len(spreads) > 1 {
		count = fmt.Sprintf("%d derivations", len(spreads))
	}
	m.ctx.ReportNode(first.property.Name(), rule.RuleMessage{
		Id: "spreadProjection",
		Description: fmt.Sprintf("This projection of `%s` restates %s that other files also restate (maximum %d files each): %s.",
			first.owner.Name, count, limit, listed(described, 3)),
		Help: fmt.Sprintf("Changing one of these derivations requires finding each file by hand. Other files, nearest first: %s.",
			listed(m.nearest(files), 3)),
	})
}

var Rule = rule.Rule{
	Name: "aexlint/max-projection-spread",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		limit := parseOptions(options)
		(&measure{ctx: ctx, derivations: map[*ast.Node]*derivation{}, projections: map[*ast.Node]bool{}}).run(limit)
		return rule.RuleListeners{}
	},
}
