package no_repeated_value_group

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/core"
	"github.com/microsoft/typescript-go/shim/scanner"
	"github.com/microsoft/typescript-go/shim/tspath"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

// ParseOptions accepts omitted options, null and {}, and rejects anything else.
func ParseOptions(name string, options any) {
	if options == nil {
		return
	}
	if object, ok := options.(map[string]any); ok && len(object) == 0 {
		return
	}
	panic(fmt.Sprintf("aexlint/%s accepts no options", name))
}

// Places describes sites relative to the linted file: same file first, then by shared leading directories, then by path and position.
type Places struct {
	ctx rule.RuleContext
}

func NewPlaces(ctx rule.RuleContext) Places { return Places{ctx: ctx} }

func (p Places) relative(fileName string) string {
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

// Location renders a node as path:line.
func (p Places) Location(n *ast.Node) string {
	file := ast.GetSourceFileOfNode(n)
	line := scanner.GetECMALineOfPosition(file, scanner.SkipTrivia(file.Text(), n.Pos())) + 1
	return fmt.Sprintf("%s:%d", p.relative(file.FileName()), line)
}

// Listed joins items, showing at most shown of them.
func Listed(items []string, shown int) string {
	if len(items) <= shown {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(items[:shown], ", "), len(items)-shown)
}

type measure struct {
	ctx        rule.RuleContext
	classifier *Classifier
	index      *Index
	places     Places
}

func complement(s *Site) []string {
	result := []string{}
	for _, value := range s.Leaves[0].Vocabulary.Values {
		if !slices.Contains(s.Leaves[0].Accepted, value) {
			result = append(result, value)
		}
	}
	return result
}

// matches returns the other unnamed sites and the named sites spelling out s's group.
func (m *measure) matches(s *Site) ([]*ast.Node, []*Site) {
	seen := map[*ast.Node]bool{s.Root: true}
	others, named := []*ast.Node{}, []*Site{}
	visit := func(root *ast.Node) {
		if seen[root] {
			return
		}
		seen[root] = true
		if other := m.classifier.Classify(root); other != nil && other.Group == s.Group {
			if other.Named != "" {
				named = append(named, other)
			} else {
				others = append(others, root)
			}
		}
	}
	for _, values := range [][]string{s.Leaves[0].Accepted, complement(s)} {
		for _, root := range m.index.Keyed(values[0]) {
			visit(root)
		}
	}
	for _, root := range m.index.Other() {
		visit(root)
	}
	return others, named
}

func (m *measure) report(s *Site, others []*ast.Node, named []*Site) {
	group := s.Leaves[0].Vocabulary.Display(s.Leaves[0].Accepted)
	var description string
	switch s.Root.Kind {
	case ast.KindArrayLiteralExpression:
		description = fmt.Sprintf("This list spells out the value group %s, which is also written elsewhere.", group)
	case ast.KindCaseClause:
		description = fmt.Sprintf("These cases group the values %s of `%s`, a group also written elsewhere.", group, Text(s.Leaves[0].Subject, 40))
	default:
		description = fmt.Sprintf("`%s` is tested against the value group %s, which is also written elsewhere.", Text(s.Leaves[0].Subject, 40), group)
	}
	m.places.Sort(others)
	locations := []string{}
	for _, n := range others {
		locations = append(locations, m.places.Location(n))
	}
	help := fmt.Sprintf("Written in %d places. Others, nearest first: %s.", len(others)+1, Listed(locations, 3))
	if len(named) > 0 {
		names := []string{}
		for _, n := range named {
			names = append(names, fmt.Sprintf("%s (%s)", n.Named, m.places.Location(n.Root)))
		}
		slices.Sort(names)
		help += " Already named by " + Listed(slices.Compact(names), 3) + "."
	}
	help += " When the group changes, every copy has to be found and changed the same way."
	m.ctx.ReportRange(core.NewTextRange(s.Pos, s.End), rule.RuleMessage{Id: "repeatedValueGroup", Description: description, Help: help})
}

func (m *measure) run() {
	for _, root := range m.index.File(m.ctx.SourceFile) {
		s := m.classifier.Classify(root)
		if s == nil || s.Group == "" || s.Named != "" {
			continue
		}
		if others, named := m.matches(s); len(others) > 0 {
			m.report(s, others, named)
		}
	}
}

var Rule = rule.Rule{
	Name: "aexlint/no-repeated-value-group",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		ParseOptions("no-repeated-value-group", options)
		(&measure{ctx: ctx, classifier: NewClassifier(ctx.TypeChecker), index: IndexFor(ctx.Program), places: NewPlaces(ctx)}).run()
		return rule.RuleListeners{}
	},
}
