package no_repeated_value_policy

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const account = `export type Plan = "team" | "personal" | "free";
export interface Account { id: string; plan: Plan }`

func policy(code, snippet string) rule_tester.InvalidTestCaseError {
	offset := strings.Index(code, snippet)
	before := code[:offset]
	line := strings.Count(before, "\n") + 1
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: "repeatedValuePolicy", Line: line, Column: column, EndColumn: column + len(snippet)}
}

func files(extra ...string) map[string]string {
	result := map[string]string{"account.ts": account}
	for i := 0; i+1 < len(extra); i += 2 {
		result[extra[i]] = extra[i+1]
	}
	return result
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	rows := `import type { Account } from "./account.js";
export function rows(accounts: Account[]) {
  return accounts.map(({ id, plan }) => ({ id, allowed: plan === "free" ? ["basic"] : undefined }));
}`
	summary := `import type { Account } from "./account.js";
export function summary(accounts: Account[], fallback: string[]) {
  return accounts.map((account) => ({ allowed: account.plan === "free" ? ["basic"] : fallback }));
}`
	blocks := `import type { Account } from "./account.js";
declare function track(event: string, id: string): void;
export function select(account: Account) {
  if (account.plan !== "free") {
    track("select", account.id);
  }
}`
	switched := `import type { Account } from "./account.js";
declare function track(event: string, id: string): void;
export function open(account: Account) {
  switch (account.plan) {
    case "personal":
      track("select", account.id);
      break;
    default:
      break;
  }
}`
	user := `import type { Account } from "./account.js";
declare function track(event: string, id: string): void;
export function visit(account: Account) { if (account.plan === "personal") { track("select", account.id); } }`
	badge := `import type { Account } from "./account.js";
export const Badge = (props: { r: Account }) => <div>{props.r.plan === "free" && <span>free</span>}</div>;`
	closed := `import type { Kind } from "./kind.js";
export function f(k: Kind) { return k === "a" ? ["first"] : []; }`
	other := `import type { Kind } from "./kind.js";
export function g(k: Kind) { return k === "a" ? ["first"] : ["second"]; }`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// The same test with a different outcome, and the same outcome for a different test.
			{Code: rows, Files: files("other.ts", `import type { Account } from "./account.js";
export function other(accounts: Account[]) {
  return accounts.map(({ plan }) => ({ allowed: plan === "free" ? ["trial"] : undefined, named: plan === "personal" ? ["basic"] : undefined }));
}`)},
			// Trivial outcomes do not establish a shared policy.
			{Code: `import type { Account } from "./account.js";
export function f(r: Account) {
  if (r.plan === "free") return null;
  if (r.plan === "personal") { return; }
  switch (r.plan) { case "team": break; }
  return r.plan === "personal" ? true : false;
}`, Files: files("other.ts", `import type { Account } from "./account.js";
export function g(r: Account) {
  if (r.plan === "free") return null;
  if (r.plan === "personal") { return; }
  switch (r.plan) { case "team": break; }
  return r.plan === "personal" ? true : false;
}`)},
			// Repetition within one file is not spread across files.
			{Code: rows + "\n" + strings.ReplaceAll(rows[strings.Index(rows, "export"):], "rows", "more"), Files: files()},
			// Broad types are not vocabularies.
			{Code: `export function f(plan: string) { return plan === "free" ? ["basic"] : undefined; }`,
				Files: files("other.ts", `export function g(plan: string) { return plan === "free" ? ["basic"] : undefined; }`)},
			// Cross-file: widening only the dependency's type removes the report.
			{Code: closed, Files: map[string]string{"kind.ts": `export type Kind = string;`, "a.ts": other}},
		},
		[]rule_tester.InvalidTestCase{
			// The same ternary outcome in another file, through a destructured and a direct property.
			{Code: rows, Files: files("other.ts", summary), Errors: []rule_tester.InvalidTestCaseError{policy(rows, `plan === "free"`)}},
			// An if block and a case arm with the same outcome; the complement reads as "anything but".
			{Code: blocks, Files: files("a.ts", strings.ReplaceAll(blocks, "select(", "pick(")), Errors: []rule_tester.InvalidTestCaseError{policy(blocks, `account.plan !== "free"`)}},
			{Code: switched, Files: files("a.ts", user), Errors: []rule_tester.InvalidTestCaseError{policy(switched, `case "personal"`)}},
			// A JSX guard.
			{Code: badge, Tsx: true, Files: files("other.tsx", `import type { Account } from "./account.js";
export const Row = ({ r }: { r: Account }) => <p>{r.plan === "free" && <span>free</span>}</p>;`),
				Errors: []rule_tester.InvalidTestCaseError{policy(badge, `props.r.plan === "free"`)}},
			// Cross-file: the dependency's closed type makes the tests a shared policy.
			{Code: closed, Files: map[string]string{"kind.ts": `export type Kind = "a" | "b";`, "a.ts": strings.ReplaceAll(other, `["second"]`, `[]`)},
				Errors: []rule_tester.InvalidTestCaseError{policy(closed, `k === "a"`)}},
		},
	)
}
