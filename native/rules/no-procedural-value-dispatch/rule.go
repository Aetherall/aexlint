package no_procedural_value_dispatch

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/typescript-eslint/tsgolint/internal/aexlint/rules/no-repeated-value-group"
	"github.com/typescript-eslint/tsgolint/internal/rule"
)

type arm struct {
	vocabulary *no_repeated_value_group.Vocabulary
	accepted   []string
	body       []*ast.Node
}

type measure struct {
	ctx        rule.RuleContext
	classifier *no_repeated_value_group.Classifier
}

// control reports whether nodes contain a decision, loop or try statement, including inside nested arrow functions and function expressions.
func control(nodes []*ast.Node) bool {
	found := false
	var walk func(*ast.Node) bool
	walk = func(n *ast.Node) bool {
		if found {
			return true
		}
		if ast.IsFunctionLike(n) && n.Kind != ast.KindArrowFunction && n.Kind != ast.KindFunctionExpression {
			return false
		}
		switch n.Kind {
		case ast.KindIfStatement, ast.KindSwitchStatement, ast.KindConditionalExpression, ast.KindTryStatement,
			ast.KindForStatement, ast.KindForOfStatement, ast.KindForInStatement, ast.KindWhileStatement, ast.KindDoStatement:
			found = true
			return true
		}
		n.ForEachChild(walk)
		return found
	}
	for _, n := range nodes {
		if n != nil {
			walk(n)
		}
	}
	return found
}

func (m *measure) report(subject *ast.Node, arms []arm) {
	procedural := []string{}
	for _, a := range arms {
		if control(a.body) {
			procedural = append(procedural, a.vocabulary.Display(a.accepted))
		}
	}
	if len(procedural) == 0 {
		return
	}
	m.ctx.ReportNode(subject, rule.RuleMessage{
		Id:          "proceduralBranch",
		Description: fmt.Sprintf("Branches on `%s` contain their own control flow.", no_repeated_value_group.Text(subject, 40)),
		Help: fmt.Sprintf("Branches with nested decisions, loops or error handling: %s. Each is a procedure for its values, written inline instead of standing on its own.",
			strings.Join(procedural, "; ")),
	})
}

// matching returns the leaf of condition testing the same subject as first.
func (m *measure) matching(condition *ast.Node, first *no_repeated_value_group.Leaf) *no_repeated_value_group.Leaf {
	for _, leaf := range m.classifier.Leaves(condition) {
		if first == nil || no_repeated_value_group.SameSubject(leaf, first) {
			return leaf
		}
	}
	return nil
}

// chain collects the arms of an if/else-if chain testing first's subject, returning the trailing else, or the first statement not testing it, as rest.
func (m *measure) chain(n *ast.Node, first *no_repeated_value_group.Leaf) ([]arm, *ast.Node) {
	arms := []arm{}
	for {
		statement := n.AsIfStatement()
		leaf := m.matching(statement.Expression, first)
		if leaf == nil {
			return arms, n
		}
		arms = append(arms, arm{vocabulary: leaf.Vocabulary, accepted: leaf.Accepted, body: []*ast.Node{statement.ThenStatement}})
		if statement.ElseStatement == nil || statement.ElseStatement.Kind != ast.KindIfStatement {
			return arms, statement.ElseStatement
		}
		n = statement.ElseStatement
	}
}

func (m *measure) statements(list []*ast.Node) {
	for i := 0; i < len(list); i++ {
		if list[i].Kind != ast.KindIfStatement {
			continue
		}
		first := m.matching(list[i].AsIfStatement().Expression, nil)
		if first == nil {
			continue
		}
		arms, rest := m.chain(list[i], first)
		end := i
		if rest == nil {
			for end+1 < len(list) && list[end+1].Kind == ast.KindIfStatement {
				more, tail := m.chain(list[end+1], first)
				if len(more) == 0 || tail != nil {
					break
				}
				arms = append(arms, more...)
				end++
			}
		}
		if len(arms) >= 2 || rest != nil {
			m.report(first.Subject, arms)
		}
		i = end
	}
}

func (m *measure) switchStatement(n *ast.Node) {
	subject := n.Expression()
	v := m.classifier.Vocabulary(subject)
	if v == nil {
		return
	}
	arms := []arm{}
	for _, clause := range n.AsSwitchStatement().CaseBlock.AsCaseBlock().Clauses.Nodes {
		if clause.Kind == ast.KindDefaultClause || !no_repeated_value_group.Root(clause) {
			continue
		}
		if s := m.classifier.Classify(clause); s != nil {
			clauses := clause.Parent.AsCaseBlock().Clauses.Nodes
			for _, c := range clauses[indexOf(clauses, clause):] {
				if statements := c.Statements(); len(statements) > 0 {
					arms = append(arms, arm{vocabulary: s.Leaves[0].Vocabulary, accepted: s.Leaves[0].Accepted, body: statements})
					break
				}
			}
		}
	}
	if len(arms) > 0 {
		m.report(no_repeated_value_group.Unwrap(subject), arms)
	}
}

func indexOf(nodes []*ast.Node, n *ast.Node) int {
	for i, m := range nodes {
		if m == n {
			return i
		}
	}
	return -1
}

func (m *measure) ternary(n *ast.Node) {
	if p := n.Parent; p.Kind == ast.KindConditionalExpression && p.AsConditionalExpression().WhenFalse == n {
		return
	}
	first := m.matching(n.AsConditionalExpression().Condition, nil)
	if first == nil {
		return
	}
	arms := []arm{}
	for current := n; current.Kind == ast.KindConditionalExpression; current = current.AsConditionalExpression().WhenFalse {
		leaf := m.matching(current.AsConditionalExpression().Condition, first)
		if leaf == nil {
			break
		}
		arms = append(arms, arm{vocabulary: leaf.Vocabulary, accepted: leaf.Accepted, body: []*ast.Node{current.AsConditionalExpression().WhenTrue}})
	}
	if len(arms) >= 2 {
		m.report(first.Subject, arms)
	}
}

func (m *measure) visit(n *ast.Node) bool {
	switch n.Kind {
	case ast.KindSourceFile, ast.KindBlock, ast.KindModuleBlock, ast.KindCaseClause, ast.KindDefaultClause:
		m.statements(n.Statements())
	case ast.KindSwitchStatement:
		m.switchStatement(n)
	case ast.KindConditionalExpression:
		m.ternary(n)
	}
	n.ForEachChild(m.visit)
	return false
}

var Rule = rule.Rule{
	Name: "aexlint/no-procedural-value-dispatch",
	Run: func(ctx rule.RuleContext, options any) rule.RuleListeners {
		no_repeated_value_group.ParseOptions("no-procedural-value-dispatch", options)
		m := &measure{ctx: ctx, classifier: no_repeated_value_group.NewClassifier(ctx.TypeChecker)}
		m.visit(ctx.SourceFile.AsNode())
		return rule.RuleListeners{}
	},
}
