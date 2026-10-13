package no_repeated_conversion

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const drop = `export type Insert = { kind: "insert"; target: "cell" | "row"; template: string };
export type Reorder = { kind: "reorder"; key: string };
export type Drop = Reorder | Insert;`

const copy = `import type { Insert } from "./drop.js";
export function toElement(item: Insert) {
  return item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };
}`

func repeated(code, snippet string, occurrence int) rule_tester.InvalidTestCaseError {
	offset := -1
	for range occurrence {
		offset += 1 + strings.Index(code[offset+1:], snippet)
	}
	before := code[:offset]
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: "repeatedConversion", Line: strings.Count(before, "\n") + 1, Column: column, EndColumn: column + len(snippet)}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"drop.ts": drop, "copy.ts": copy}
	renamed := `import type { Insert } from "./drop.js";
export const element = (dragged: Insert) =>
  dragged.target === "cell" ? { template: dragged.template, type: "cell" } : { type: "row" };`
	narrowed := `import type { Drop } from "./drop.js";
export function element(item: Drop) {
  if (item.kind !== "insert") return undefined;
  return item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };
}`
	sameFile := `type Value = { format: "boolean" | "text" | "date"; value: string };
export function first(value: Value) {
  switch (value.format) {
    case "boolean": return { format: "boolean", value: value.value };
    case "text": return { format: "text", value: value.value };
    case "date": return { format: "date", value: value.value };
  }
}
export function second(value: Value) {
  switch (value.format) {
    case "date": return { format: "date", value: value.value };
    case "text": return { format: "text", value: value.value };
    case "boolean": return { format: "boolean", value: value.value };
  }
}`
	extra := `type Value = { format: "boolean" | "text" | "date"; value: string };
export function first(value: Value) {
  switch (value.format) {
    case "boolean": return { format: "boolean", value: value.value };
    case "text": return { format: "text", value: value.value };
  }
}
export function second(value: Value) {
  switch (value.format) {
    case "boolean": return { format: "boolean", value: value.value };
    case "text": return { format: "text", value: value.value };
    case "date": return { format: "date", value: value.value };
  }
}`
	action := `type Reminder = { type: "crew" | "personal"; color: string; crewId: string };
declare function createForCrew(input: { color: string; crewId: string }): void;
declare function createPersonal(input: { color: string }): void;
export function paste(reminder: Reminder) {
  if (reminder.type === "crew") createForCrew({ color: reminder.color, crewId: reminder.crewId });
  else createPersonal({ color: reminder.color });
}
export function assign(details: Reminder) {
  if (details.type === "crew") createForCrew({ color: details.color, crewId: details.crewId });
  else createPersonal({ color: details.color });
}`
	labelled := `import type { Insert } from "./drop.js";
export const element = (item: Insert) =>
  item.target === "cell" ? { type: "cell", label: ` + "`item ${item.template}`" + ` } : { type: "row" };`
	labelledCopy := `import type { Insert } from "./drop.js";
export const element = (dragged: Insert) =>
  dragged.target === "cell" ? { type: "cell", label: ` + "`item ${dragged.template}`" + ` } : { type: "row" };`
	shadowed := `type Group = { kind: "list" | "one"; items: { id: string }[] };
export function a(item: Group) {
  return item.kind === "list" ? { ids: item.items.map((item) => item.id) } : { kind: "one" };
}
export function b(group: Group) {
  return group.kind === "list" ? { ids: group.items.map((item) => item.id) } : { kind: "one" };
}`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: `import type { Insert } from "./drop.js";
export const element = (item: Insert) =>
  item.target === "cell" ? { type: "cell", template: item.template, extra: true } : { type: "row" };`, Files: files},
			{Code: `import type { Insert } from "./drop.js";
export const element = (item: Insert) =>
  item.target === "row" ? { type: "cell", template: item.template } : { type: "row" };`, Files: files},
			{Code: `type Other = { target: "cell" | "row"; template: string };
export const element = (item: Other) =>
  item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };`, Files: files},
			{Code: copy, Files: map[string]string{"drop.ts": drop}},
			{Code: `type Value = { format: "boolean" | "text"; value: string };
declare function send(value: unknown): void;
export function first(value: Value) {
  if (value.format === "boolean") return value.value;
  return "";
}
export function second(value: Value) {
  if (value.format === "boolean") return value.value;
  return "";
}`},
			{Code: `export const element = (item: any) => item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };`, Files: map[string]string{"copy.ts": `export const element = (item: any) => item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };`}},
			{Code: `export const element = (item: Missing) => item.target === "cell" ? { type: "cell", template: item.template } : { type: "row" };`, Files: files},
			{Code: `export const element = (item: unknown) => item;`},
			{Code: `export const element = (item: never) => item;`},
			{Code: renamed, Files: map[string]string{"drop.ts": strings.Replace(drop, `target: "cell" | "row"`, "target: string", 1), "copy.ts": copy}},
			{Code: extra},
			{Code: `export const value = 1;`, Options: rule_tester.OptionsFromJSON[any](`{}`)},
		},
		[]rule_tester.InvalidTestCase{
			{Code: renamed, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(renamed, "dragged.target", 1)}},
			{Code: narrowed, Files: files, Errors: []rule_tester.InvalidTestCaseError{repeated(narrowed, "item.target", 1)}},
			{Code: sameFile, Errors: []rule_tester.InvalidTestCaseError{repeated(sameFile, "value.format", 1), repeated(sameFile, "value.format", 2)}},
			{Code: action, Errors: []rule_tester.InvalidTestCaseError{repeated(action, "reminder.type", 1), repeated(action, "details.type", 1)}},
			{Code: labelled, Files: map[string]string{"drop.ts": drop, "copy.ts": labelledCopy}, Errors: []rule_tester.InvalidTestCaseError{repeated(labelled, "item.target", 1)}},
			{Code: shadowed, Errors: []rule_tester.InvalidTestCaseError{repeated(shadowed, "item.kind", 1), repeated(shadowed, "group.kind", 1)}},
		},
	)
}
