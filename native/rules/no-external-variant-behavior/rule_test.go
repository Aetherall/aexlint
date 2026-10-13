package no_external_variant_behavior

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

const figure = `export class Figure {
  constructor(public kind: "line" | "circle", public start: number, public center: number) {}
  static transform(value: Figure) {
    switch (value.kind) {
      case "line": return { value: value.start };
      case "circle": return { value: value.center };
    }
  }
  transform() {
    switch (this.kind) {
      case "line": return { value: this.start };
      case "circle": return { value: this.center };
    }
  }
}
export type FigureAlias = Figure;`

const tokenType = `export type Token = { category: "word"; text: string } | { category: "count"; amount: number };
export type TokenAlias = Token;`

const tokenOwner = `export const Token = {
  transform(token: Token) {
    switch (token.category) {
      case "word": return { value: token.text };
      case "count": return { value: token.amount };
    }
  }
};`

func external(code, snippet string) rule_tester.InvalidTestCaseError {
	return externalAt(code, snippet, strings.Index(code, snippet))
}

func externalAt(code, snippet string, offset int) rule_tester.InvalidTestCaseError {
	before := code[:offset]
	line := strings.Count(before, "\n") + 1
	column := offset - strings.LastIndex(before, "\n")
	return rule_tester.InvalidTestCaseError{MessageId: "externalVariantBehavior", Line: line, Column: column, EndColumn: column + len(snippet)}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"figure.ts": figure}
	switched := `import { Figure } from "./figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    case "line": return { value: value.start + 1 };
    case "circle": return { value: value.center + 1 };
  }
}`
	merged := `import type { TokenAlias } from "./token.js";
export function transform(value: TokenAlias) {
  switch (value.category) {
    case "word": return { value: value.text };
    case "count": return { value: value.amount };
  }
}`
	aliased := strings.ReplaceAll(strings.ReplaceAll(switched, "{ Figure }", "{ Figure as Drawing }"), "value: Figure", "value: Drawing")
	generic := strings.ReplaceAll(switched, "transform(value: Figure)", "transform<T extends Figure>(value: T)")
	asserted := strings.ReplaceAll(switched, "value.kind", "(value as Figure).kind")
	embedded := `import { Figure } from "./figure.js";
export function notify(value: Figure) {
  if (value.kind === "line") return { source: { value: value.start } };
  if (value.kind === "circle") return { source: { value: value.center } };
}`
	ternary := `import { Figure } from "./figure.js";
export const notify = (value: Figure) => value.kind === "line"
  ? { source: value.start }
  : value.kind === "circle" ? { source: value.center } : undefined;`
	unrelated := `import { Figure as Drawing } from "./figure.js";
export class Figure {
  static transform(value: Drawing) {
    switch (value.kind) {
      case "line": return { value: value.start };
      case "circle": return { value: value.center };
    }
  }
}`
	constructor := `import { Figure } from "./figure.js";
class Result { constructor(public source: { value: number }) {} }
export function transform(value: Figure) {
  switch (value.kind) {
    case "line": return new Result({ value: value.start });
    case "circle": return new Result({ value: value.center });
  }
}`
	array := `import { Figure } from "./figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    case "line": return [value.start];
    case "circle": return [value.center];
  }
}`
	alternative := `import { Figure } from "./figure.js";
export function notify(value: Figure) {
  if (value.kind === "line") {
    return value.transform();
  } else {
    return { source: value.center };
  }
}`
	single := `import { Figure } from "./figure.js";
export const notify = (value: Figure) => value.kind === "line" ? { source: value.start } : { source: value.center };`
	negated := `import { Figure } from "./figure.js";
export const notify = (value: Figure) => !(value.kind === "line" || value.kind === "line") ? { source: value.center } : value.transform();`
	defaulted := `import { Figure } from "./figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    case "line": return value.transform();
    default: return [value.center];
  }
}`
	declared := `import { Figure } from "./figure.js";
export function notify(value: Figure) {
  if (value.kind === "line") {
    const result: { source: number } = { source: value.start };
    return result;
  } else {
    return value.transform();
  }
}`
	assigned := `import { Figure } from "./figure.js";
export function notify(value: Figure, target: { source?: { value: number } }) {
  switch (value.kind) {
    case "line":
      target.source = { value: value.start };
      break;
    case "circle":
      target.source = { value: value.center };
      break;
  }
}`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: figure},
			{Code: `import { Figure } from "./figure.js";
export function notify(value: Figure) {
  let total = 0;
  switch (value.kind) {
    case "line": total += value.start; break;
    case "circle": total = value.center; break;
  }
  return total;
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export function notify(value: Figure) {
  if (value.kind === "line") return { source: value.start };
  return value.transform();
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export const notify = (value: Figure) => value.kind === "line" ? { enabled: true } : undefined;`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export const notify = (value: Figure, ready: boolean) => value.kind === "line" && ready ? value.transform() : { source: value.center };`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export function notify(value: Figure, ready: boolean) {
  if (value.kind === "line" && ready) return value.transform();
  else return { source: value.center };
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    default: return { source: value.center };
  }
}`, Files: files},
			{Code: tokenType + "\n" + tokenOwner},
			{Code: `import { Figure } from "./figure.js";
export function transform(value: Figure) {
  switch (value.kind) {
    case "line": return Figure.transform(value);
    case "circle": return value.transform();
  }
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export function flags(value: Figure) {
  switch (value.kind) {
    case "line": return { enabled: true };
    case "circle": return { enabled: false };
  }
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export function inspect(value: Figure) {
  switch (value.kind) {
    case "line": return ((value: { start: number }) => ({ source: value.start }))({ start: 0 });
    case "circle": return ((value: { center: number }) => ({ source: value.center }))({ center: 0 });
  }
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
declare function log(message: { source: number }): void;
export function inspect(value: Figure) {
  switch (value.kind) {
    case "line": log({ source: value.start }); return 1;
    case "circle": log({ source: value.center }); return 2;
  }
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
declare function log(message: { source: number }): void;
export function inspect(value: Figure) {
  switch (value.kind) {
    case "line": return log({ source: value.start });
    case "circle": return log({ source: value.center });
  }
}`, Files: files},
			{Code: `import { Figure } from "./figure.js";
export function inspect(value: Figure) {
  switch (value.kind) {
    case "line": return () => ({ source: value.start });
    case "circle": return () => ({ source: value.center });
  }
}`, Files: files},
			{Code: `type Figure = { kind: "line"; start: number } | { kind: "circle"; center: number };
export function transform(value: Figure) {
  switch (value.kind) {
    case "line": return { value: value.start };
    case "circle": return { value: value.center };
  }
}`},
			{Code: switched, Files: map[string]string{"figure.ts": `export declare class Figure {
  kind: "line" | "circle";
  start: number;
  center: number;
}`}},
			{Code: switched, Files: map[string]string{"figure.ts": `export class Figure {
  kind = "line";
  start = 0;
  center = 0;
}`}},
			{Code: `export function transform(value: any) {
  switch (value.kind) {
    case "line": return { source: value.start };
    case "circle": return { source: value.center };
  }
}`},
			{Code: `export function transform(value: unknown) { return value; }`},
			{Code: `export function transform(value: never) { return value; }`},
			{Code: switched, Files: map[string]string{}},
			{Code: `export const value = 1;`, Options: rule_tester.OptionsFromJSON[any](`{}`)},
		},
		[]rule_tester.InvalidTestCase{
			{Code: switched, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(switched, "value.kind")}},
			{Code: aliased, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(aliased, "value.kind")}},
			{Code: generic, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(generic, "value.kind")}},
			{Code: asserted, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(asserted, "(value as Figure).kind")}},
			{Code: merged, Files: map[string]string{"token.ts": tokenType + "\n" + tokenOwner}, Errors: []rule_tester.InvalidTestCaseError{external(merged, "value.category")}},
			{Code: merged, Files: map[string]string{"token.ts": tokenType}, Errors: []rule_tester.InvalidTestCaseError{external(merged, "value.category")}},
			{Code: embedded, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(embedded, "value.kind")}},
			{Code: ternary, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(ternary, "value.kind")}},
			{Code: unrelated, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(unrelated, "value.kind")}},
			{Code: constructor, Files: files, Errors: []rule_tester.InvalidTestCaseError{conversion(constructor, "value.kind")}},
			{Code: array, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(array, "value.kind")}},
			{Code: alternative, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(alternative, "value.kind")}},
			{Code: single, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(single, "value.kind")}},
			{Code: negated, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(negated, "value.kind")}},
			{Code: defaulted, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(defaulted, "value.kind")}},
			{Code: declared, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(declared, "value.kind")}},
			{Code: assigned, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(assigned, "value.kind")}},
		},
	)
}

func TestModuleOwnership(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	operation := `export function transform(value: Token) {
  switch (value.category) {
    case "word": return { value: value.text };
    case "count": return { value: value.amount };
  }
}`
	consumer := `import type { Token } from "./token.js";` + "\n" + operation
	barrel := strings.ReplaceAll(consumer, "./token.js", "./barrel.js")
	alias := strings.ReplaceAll(consumer, "value: Token", "value: Local") + "\ntype Local = Token;"
	generic := strings.ReplaceAll(consumer, "transform(value: Token)", "transform<T extends Token>(value: T)")
	inferred := `import { create } from "./token.js";
export function transform() {
  const value = create();
  switch (value.category) {
    case "word": return { value: value.text };
    case "count": return { value: value.amount };
  }
}`
	property := `import type { Token } from "./token.js";
export function transform(holder: { value: Token }) {
  switch (holder.value.category) {
    case "word": return { value: holder.value.text };
    case "count": return { value: holder.value.amount };
  }
}`
	namespaced := "export namespace Tokens {\n" + tokenType + "\n" + operation + "\n}"
	namespaceConsumer := `import { Tokens } from "./token.js";` + "\n" + strings.ReplaceAll(operation, "value: Token", "value: Tokens.Token")
	siblingNamespace := namespaced + "\nexport namespace Consumer {\n" + strings.ReplaceAll(operation, "value: Token", "value: Tokens.Token") + "\n}"
	classOutside := figure + `
export function outside(value: Figure) {
  switch (value.kind) {
    case "line": return { value: value.start };
    case "circle": return { value: value.center };
  }
}`
	runtimeOutside := tokenType + "\n" + tokenOwner + "\n" + operation
	namespaceRuntime := tokenType + "\nexport namespace Token {\n" + operation + "\n}"
	namespaceOutside := namespaceRuntime + "\n" + strings.ReplaceAll(operation, "transform(", "outside(")
	aliasToClass := strings.ReplaceAll(classOutside, "outside(value: Figure)", "outside(value: FigureAlias)")
	interfaceType := `export interface Token { category: "word" | "count"; text: string; amount: number }`
	files := map[string]string{"token.ts": tokenType}
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: tokenType + "\n" + operation},
			{Code: tokenType + "\ntype Local = Token;\n" + strings.ReplaceAll(operation, "value: Token", "value: Local")},
			{Code: namespaced},
			{Code: namespaced + "\nexport namespace Tokens {\n" + strings.ReplaceAll(operation, "transform(", "second(") + "\n}"},
			{Code: namespaceConsumer, Files: map[string]string{"token.ts": "export declare namespace Tokens {\n" + tokenType + "\n}"}},
			{Code: consumer, Files: map[string]string{"token.d.ts": tokenType}},
			{Code: interfaceType + "\n" + operation},
			{Code: namespaceRuntime},
			{Code: consumer, Files: map[string]string{"token.ts": strings.ReplaceAll(tokenType, "export type", "export declare type")}},
			{Code: consumer, Files: map[string]string{"token.ts": `export type Token = Missing;`}},
			{Code: consumer, Files: map[string]string{"token.ts": `export type Token = Token;`}},
			{Code: "interface Token { category: string; text: string; amount: number }\n" + operation},
		},
		[]rule_tester.InvalidTestCase{
			{Code: consumer, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(consumer, "value.category")}},
			{Code: consumer, Files: map[string]string{"token.ts": interfaceType}, Errors: []rule_tester.InvalidTestCaseError{external(consumer, "value.category")}},
			{Code: barrel, Files: map[string]string{"token.ts": tokenType, "barrel.ts": `export type { Token } from "./token.js";`}, Errors: []rule_tester.InvalidTestCaseError{external(barrel, "value.category")}},
			{Code: alias, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(alias, "value.category")}},
			{Code: generic, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(generic, "value.category")}},
			{Code: inferred, Files: map[string]string{"token.ts": tokenType + `
export function create(): Token { return { category: "word", text: "hello" }; }`}, Errors: []rule_tester.InvalidTestCaseError{external(inferred, "value.category")}},
			{Code: property, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(property, "holder.value.category")}},
			{Code: namespaceConsumer, Files: map[string]string{"token.ts": namespaced}, Errors: []rule_tester.InvalidTestCaseError{external(namespaceConsumer, "value.category")}},
			{Code: siblingNamespace, Errors: []rule_tester.InvalidTestCaseError{externalAt(siblingNamespace, "value.category", strings.LastIndex(siblingNamespace, "value.category"))}},
			{Code: classOutside, Errors: []rule_tester.InvalidTestCaseError{externalAt(classOutside, "value.kind", strings.LastIndex(classOutside, "value.kind"))}},
			{Code: runtimeOutside, Errors: []rule_tester.InvalidTestCaseError{external(runtimeOutside, "value.category")}},
			{Code: namespaceOutside, Errors: []rule_tester.InvalidTestCaseError{externalAt(namespaceOutside, "value.category", strings.LastIndex(namespaceOutside, "value.category"))}},
			{Code: aliasToClass, Errors: []rule_tester.InvalidTestCaseError{externalAt(aliasToClass, "value.kind", strings.LastIndex(aliasToClass, "value.kind"))}},
		},
	)
}

func conversion(code, snippet string) rule_tester.InvalidTestCaseError {
	e := external(code, snippet)
	e.MessageId = "externalVariantConversion"
	return e
}

func TestConversionOwnership(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	label := `export type Label = { text: number } | { size: number };`
	shape := `import { Figure } from "./figure.js";
export class Shape {
  constructor(public source: { size: number }) {}
  static from(value: Figure) {
    switch (value.kind) {
      case "line": return new Shape({ size: value.start });
      case "circle": return new Shape({ size: value.center });
    }
  }
}`
	files := map[string]string{"figure.ts": figure, "label.ts": label, "shape.ts": shape, "token.ts": tokenType}
	consumer := `import { Figure } from "./figure.js";
import type { Label } from "./label.js";
export function label(value: Figure): Label {
  switch (value.kind) {
    case "line": return { text: value.start };
    case "circle": return { size: value.center };
  }
}`
	declared := `import { Figure } from "./figure.js";
import type { Label } from "./label.js";
export function label(value: Figure) {
  if (value.kind === "line") {
    const result: Label = { text: value.start };
    return result;
  } else {
    return value.transform();
  }
}`
	constructed := `import { Figure } from "./figure.js";
import { Shape } from "./shape.js";
export function shape(value: Figure) {
  return value.kind === "line" ? new Shape({ size: value.start }) : new Shape({ size: value.center });
}`
	sameOwner := `import type { Token } from "./token.js";
export function swap(value: Token): Token {
  return value.category === "word" ? { category: "count", amount: value.text.length } : { category: "word", text: String(value.amount) };
}`
	many := `import { Figure } from "./figure.js";
import type { Label } from "./label.js";
import type { Token } from "./token.js";
import { Shape } from "./shape.js";
export function many(value: Figure, ready: boolean) {
  switch (value.kind) {
    case "line": {
      const label: Label = { text: value.start };
      return label;
    }
    case "circle": {
      if (ready) {
        const token: Token = { category: "count", amount: value.center };
        return token;
      }
      return new Shape({ size: value.center });
    }
  }
}`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			{Code: shape, Files: files},
			{Code: label + "\n" + strings.ReplaceAll(consumer, "import type { Label } from \"./label.js\";\n", ""), Files: files},
		},
		[]rule_tester.InvalidTestCase{
			{Code: consumer, Files: files, Errors: []rule_tester.InvalidTestCaseError{conversion(consumer, "value.kind")}},
			{Code: declared, Files: files, Errors: []rule_tester.InvalidTestCaseError{conversion(declared, "value.kind")}},
			{Code: constructed, Files: files, Errors: []rule_tester.InvalidTestCaseError{conversion(constructed, "value.kind")}},
			{Code: consumer, Files: map[string]string{"figure.ts": figure, "label.d.ts": label}, Errors: []rule_tester.InvalidTestCaseError{external(consumer, "value.kind")}},
			{Code: many, Files: files, Errors: []rule_tester.InvalidTestCaseError{conversion(many, "value.kind")}},
			{Code: sameOwner, Files: files, Errors: []rule_tester.InvalidTestCaseError{external(sameOwner, "value.category")}},
		},
	)
}
