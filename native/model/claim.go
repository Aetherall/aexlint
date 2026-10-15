package model

import (
	"slices"

	"github.com/microsoft/typescript-go/shim/ast"
	"github.com/microsoft/typescript-go/shim/checker"
)

// Claim is what a test establishes about a value: it is one of Accepted, among the values of Domain. A claim about
// literal values ranges over its Vocabulary; a claim about presence ranges over Present and the missing values
// undefined, null and other falsy values.
type Claim struct {
	Domain     []string
	Accepted   []string
	Vocabulary *Vocabulary
}

const (
	Present          = "present"
	MissingUndefined = "undefined"
	MissingNull      = "null"
	MissingFalsy     = "falsy"
)

var presence = []string{MissingFalsy, MissingNull, Present, MissingUndefined}

func missing(a Absence) []string {
	keys := []string{}
	for _, k := range []struct {
		bit Absence
		key string
	}{{AbsentFalsy, MissingFalsy}, {AbsentNull, MissingNull}, {AbsentUndefined, MissingUndefined}} {
		if a&k.bit != 0 {
			keys = append(keys, k.key)
		}
	}
	return keys
}

func minus(a, b []string) []string {
	return slices.DeleteFunc(slices.Clone(a), func(k string) bool { return slices.Contains(b, k) })
}

func within(a, b []string) []string {
	return slices.DeleteFunc(slices.Clone(a), func(k string) bool { return !slices.Contains(b, k) })
}

func presenceClaim(p Presence) Claim {
	removed := missing(p.Removes)
	if p.Missing {
		return Claim{Domain: presence, Accepted: removed}
	}
	return Claim{Domain: presence, Accepted: minus(presence, removed)}
}

// Truthiness is what a truthiness test establishes: the value is present and not falsy.
func Truthiness() Claim { return Claim{Domain: presence, Accepted: []string{Present}} }

// Presence reports whether c is about presence rather than literal values.
func (c Claim) Presence() bool { return c.Vocabulary == nil }

// Not is what holds where c does not.
func (c Claim) Not() Claim {
	return Claim{Domain: c.Domain, Accepted: minus(c.Domain, c.Accepted), Vocabulary: c.Vocabulary}
}

// Under restricts c to the values in held, such as the values a type can hold.
func (c Claim) Under(held []string) Claim {
	return Claim{Domain: within(c.Domain, held), Accepted: within(c.Accepted, held), Vocabulary: c.Vocabulary}
}

// Or is what holds where c or o holds.
func (c Claim) Or(o Claim) Claim {
	return Claim{Domain: c.Domain, Accepted: slices.DeleteFunc(slices.Clone(c.Domain), func(k string) bool {
		return !slices.Contains(c.Accepted, k) && !slices.Contains(o.Accepted, k)
	}), Vocabulary: c.Vocabulary}
}

// And is what holds where c and o hold.
func (c Claim) And(o Claim) Claim {
	return Claim{Domain: c.Domain, Accepted: within(c.Accepted, o.Accepted), Vocabulary: c.Vocabulary}
}

// Arity is the size of the smaller side of the split c makes: 1 for a test of a single value or of all values but one.
func (c Claim) Arity() int { return min(len(c.Accepted), len(c.Domain)-len(c.Accepted)) }

// Rejected is the values c rules out.
func (c Claim) Rejected() []string { return minus(c.Domain, c.Accepted) }

// Splits reports whether c separates its domain: it accepts some values and rules out others.
func (c Claim) Splits() bool { return len(c.Accepted) > 0 && len(c.Accepted) < len(c.Domain) }

// Same reports whether two claims accept the same values among the same domain.
func (c Claim) Same(o Claim) bool {
	return c.comparable(o) && sameSet(c.Domain, o.Domain) && sameSet(c.Accepted, o.Accepted)
}

func sameSet(a, b []string) bool {
	return len(a) == len(b) && len(minus(a, b)) == 0
}

func (c Claim) comparable(o Claim) bool {
	return c.Presence() == o.Presence() && (c.Presence() || slices.Equal(c.Vocabulary.Values, o.Vocabulary.Values))
}

// Implies reports whether every value c accepts, o accepts too.
func (c Claim) Implies(o Claim) bool {
	return c.comparable(o) && len(minus(c.Accepted, o.Accepted)) == 0
}

// Held returns the presence values n's type can hold at its location, or false when its type is never.
func (m *Analyzer) Held(n *ast.Node) ([]string, bool) {
	t := m.ctx.TypeChecker.GetTypeAtLocation(n)
	if t == nil || t.Flags()&checker.TypeFlagsNever != 0 {
		return nil, false
	}
	return m.HeldBy(t), true
}

// HeldBy returns the presence values t can hold.
func (m *Analyzer) HeldBy(t *checker.Type) []string {
	held := missing(m.holds(t))
	if m.present(t) {
		held = append(held, Present)
	}
	return held
}
