package max_projection_spread

import (
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

func spreadAt(line, column, endColumn int) rule_tester.InvalidTestCaseError {
	return rule_tester.InvalidTestCaseError{MessageId: "spreadProjection", Line: line, Column: column, EndColumn: endColumn}
}

const site = `export class Visibility { constructor(private readonly value: string) {} is(value: string) { return this.value === value; } }
export class SiteId { constructor(private readonly value: string) {} serialize() { return this.value; } }
export class Site {
  constructor(readonly id: SiteId, private readonly visibility: Visibility, private readonly name: string) {}
  getVisibility() { return this.visibility; }
  getName() { return this.name; }
}
export class Building {
  constructor(readonly id: SiteId, private readonly visibility: Visibility) {}
  getVisibility() { return this.visibility; }
}`

const projection = `import type { Site } from "./site.js";
export function project(site: Site) {
  return {
    id: site.id.serialize(),
    private: site.getVisibility().is("private"),
  };
}`

func files(extra map[string]string) map[string]string {
	result := map[string]string{"site.ts": site}
	for name, code := range extra {
		result[name] = code
	}
	return result
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	list := `import type { Site } from "./site.js";
export const list = (sites: Site[]) => sites.map((site) => ({ id: site.id.serialize(), private: site.getVisibility().is("private") }));`
	search := `import type { Site } from "./site.js";
export function search(found: Site) { return { private: found.getVisibility().is("private"), id: found.id.serialize() }; }`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// Two files restate the derivations: the default limit is 2.
			{Code: projection, Files: files(map[string]string{"list.ts": list})},
			// Cross-file: a different literal argument is a different derivation.
			{Code: projection, Files: files(map[string]string{"list.ts": list, "public.ts": `import type { Site } from "./site.js";
export function project(site: Site) { return { id: site.getName(), public: site.getVisibility().is("public") }; }`})},
			// A literal with one derivation uses the value; it is not a projection.
			{Code: projection, Files: files(map[string]string{"list.ts": list, "one.ts": `import type { Site } from "./site.js";
export function label(site: Site) { return { private: site.getVisibility().is("private"), title: "x" }; }`})},
			// Same-named members of another type are different derivations.
			{Code: projection, Files: files(map[string]string{"list.ts": list, "building.ts": `import type { Building } from "./site.js";
export function project(site: Building) { return { id: site.id.serialize(), private: site.getVisibility().is("private") }; }`})},
			// Copies, shorthand, this-rooted chains, library types, and non-literal arguments are not derivations.
			{Code: `import type { Site } from "./site.js";
export function copies(site: Site, id: string, locale: string) {
  return { id, copy: site.id, label: site.getName().padEnd(locale.length), other: String(site.getName()) };
}
export function dates(date: Date) { return { at: date.toISOString(), day: date.getDay() }; }
export class Owner { private readonly name = ""; describe() { return { upper: this.name.toUpperCase(), lower: this.name.toLowerCase() }; } }`,
				Files: files(map[string]string{
					"b.ts": `export const b = (date: Date) => ({ at: date.toISOString(), day: date.getDay() });`,
					"c.ts": `export const c = (date: Date) => ({ at: date.toISOString(), day: date.getDay() });`,
				})},
			// An empty options object is accepted.
			{Code: `export const value = 1;`, Options: rule_tester.OptionsFromJSON[any](`{}`)},
		},
		[]rule_tester.InvalidTestCase{
			// Three files restate id and private, through different variable names and property orders.
			{Code: projection, Files: files(map[string]string{"list.ts": list, "search.ts": search}),
				Errors: []rule_tester.InvalidTestCaseError{spreadAt(4, 5, 7)}},
			// The limit is configurable: four files with max 3.
			{Code: projection, Files: files(map[string]string{"list.ts": list, "search.ts": search, "more.ts": `import type { Site } from "./site.js";
export const more = (site: Site) => ({ id: site.id.serialize(), private: site.getVisibility().is("private"), name: site.getName() });`}),
				Options: rule_tester.OptionsFromJSON[any](`{"max": 3}`), Errors: []rule_tester.InvalidTestCaseError{spreadAt(4, 5, 7)}},
			// Keyword arguments, and a string argument that reads like a number, are compared by value and kind.
			{Code: `export interface Flags { get(name: string | number | boolean | null): { on(): boolean } }
export const a = (flags: Flags) => ({ x: flags.get(true).on(), y: flags.get(null).on(), z: flags.get("1").on() });`,
				Files: map[string]string{
					"b.ts": `import type { Flags } from "./file.js"; export const b = (f: Flags) => ({ x: f.get(true).on(), y: f.get(null).on(), z: f.get(1).on() });`,
					"c.ts": `import type { Flags } from "./file.js"; export const c = (f: Flags) => ({ x: f.get(true).on(), y: f.get(null).on(), z: f.get(1).on() });`,
				},
				Errors: []rule_tester.InvalidTestCaseError{spreadAt(2, 39, 40)}},
			// An interface owner, and a nullable root.
			{Code: `export interface Account { getEmail(): { normalize(): string }; getPlan(): { is(name: string): boolean } }
export function a(account: Account | undefined) { return { email: account!.getEmail().normalize(), paid: account!.getPlan().is("paid") }; }`,
				Files: map[string]string{
					"b.ts": `import type { Account } from "./file.js"; export const b = (x: Account) => ({ email: x.getEmail().normalize(), paid: x.getPlan().is("paid") });`,
					"c.ts": `import type { Account } from "./file.js"; export const c = (y: Account) => ({ paid: y.getPlan().is("paid"), email: y.getEmail().normalize() });`,
				},
				Errors: []rule_tester.InvalidTestCaseError{spreadAt(2, 60, 65)}},
		},
	)
}
