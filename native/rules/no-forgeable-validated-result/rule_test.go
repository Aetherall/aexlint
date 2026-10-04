package no_forgeable_validated_result

import (
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

func forgeable(line, column, endColumn int) rule_tester.InvalidTestCaseError {
	return rule_tester.InvalidTestCaseError{MessageId: "forgeableValidatedResult", Line: line, Column: column, EndColumn: endColumn}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	bind := `import { assertTool, type Tool } from "./policy.js";
export function bind(tool: Tool): Tool { assertTool(tool); return { ...tool }; }`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// A class with private state can only be produced by its constructor.
			{Code: `export class Payload {
  readonly #model: string;
  constructor(input: unknown) {
    if (typeof input !== "string") throw new Error("model");
    this.#model = input;
  }
}
export function parsePayload(input: unknown): Payload { return new Payload(input); }
export class Guarded {
  private readonly model: string;
  constructor(input: string) { if (!input) throw new Error("model"); this.model = input; }
}`},
			// A class owns its instances, even with public fields: a validating constructor, a validating static factory,
			// and a check made by constructing a class are all accepted.
			{Code: `export class Settings {
  model: string;
  constructor(input: unknown) {
    if (typeof input !== "string") throw new Error("model");
    this.model = input;
  }
}
export class Name {
  constructor(readonly value: string) {}
  static parse(input: string): Name { if (!input) throw new Error("empty"); return new Name(input); }
}
class Check { constructor(input: string) { if (!input) throw new Error("x"); } }
type Payload = { model: string };
export function make(input: string): Payload { new Check(input); return { model: input }; }`},
			// A unique symbol key brands the type.
			{Code: `declare const brand: unique symbol;
export type Checked = { readonly [brand]: true; model: string };
export function check(input: { model: string }): Checked {
  if (!input.model) throw new Error("model");
  return input as Checked;
}`},
			// Throws that do not depend on input, inside callbacks, inside try blocks, or after an early-return guard.
			{Code: `type Payload = { model: string };
let initialized = false;
export function a(input: string): Payload { if (!initialized) throw new Error("init"); return { model: input }; }
export function b(input: string[]): Payload { input.forEach((x) => { if (!x) throw new Error("x"); }); return { model: "" }; }
export function c(input: string): Payload { try { if (!input) throw new Error("x"); } catch { return { model: "" }; } return { model: input }; }
export function d(input: string): Payload { if (input) return { model: input }; throw new Error("empty"); }`},
			// Rejection only inside a declaration-file function is not followed.
			{Code: `import { parse } from "./schema.js";
type Payload = { model: string };
export function load(input: unknown): Payload { parse(input); return { model: "" }; }`, Files: map[string]string{
				"schema.d.ts": `export declare function parse(input: unknown): void;`,
			}},
			// A result type declared in a declaration file is not owned by the project.
			{Code: `import type { External } from "./external.js";
export function check(input: External): External { if (!input.model) throw new Error("model"); return { ...input }; }`, Files: map[string]string{
				"external.d.ts": `export interface External { model: string }`,
			}},
			// Primitives, arrays, functions, type parameters, type guards, assertion functions, and unknown are not checked.
			{Code: `export function a(input: string): string { if (!input) throw new Error("x"); return input; }
export function b(input: string[]): string[] { if (!input.length) throw new Error("x"); return input; }
export function c(input: string): () => string { if (!input) throw new Error("x"); return () => input; }
export function d<T extends object>(input: T): T { if (!input) throw new Error("x"); return input; }
export function e(input: unknown): input is { model: string } { if (input === null) throw new Error("x"); return true; }
export function f(input: unknown): asserts input is { model: string } { if (!input) throw new Error("x"); }
export function g(input: unknown): unknown { if (!input) throw new Error("x"); return input; }`},
			// Cross-file: the dependency logs instead of throwing.
			{Code: bind, Files: map[string]string{
				"policy.ts": `export type Tool = { name: string; run(): void };
export function assertTool(tool: Tool): void { if (!tool.name) console.warn("unnamed tool"); }`,
			}},
			// An inline library type, such as Record without a project alias, is not owned by the project.
			{Code: `export function settings(input: unknown): Record<string, unknown> { if (!input) throw new Error("x"); return {}; }`},
			// A presence test establishes only what the non-nullable result type already carries.
			{Code: `type Payload = { model: string };
const cache = new Map<string, Payload>();
export function get(key: string): Payload { const found = cache.get(key); if (!found) throw new Error("missing"); return found; }
const table: Record<string, Payload> = {};
export function lookup(key: string): Payload { const found = table[key]; if (!found) throw new Error("missing"); return { ...found }; }
export function copy(key: string): Payload { const found = cache.get(key); if (found === undefined || null === found) throw new Error("missing"); return { ...found }; }`},
			// A check made by an owner's constructor is carried by the owner's type, not by the caller's result.
			{Code: `class Owned { readonly #id: string; constructor(id: string) { if (!id) throw new Error("id"); this.#id = id; } }
type View = { owner: Owned };
export function view(id: string): View { return { owner: new Owned(id) }; }`},
			// A result made only of operations holds no state for the check to establish.
			{Code: `export function create(options: { max: number }): { enter(): void; exit: () => void } {
  if (options.max < 1) throw new Error("max");
  return { enter() {}, exit: () => {} };
}`},
			// Preconditions: the checks read data the result is not built from. A lookup by a primitive key produces new data,
			// so the loaded record is its own root, and the key flowing into the result does not connect them.
			{Code: `type Template = { id: string; isDefault: boolean };
type Actor = { roles: string[] };
const templates = new Map<string, Template>();
const existing: Template[] = [];
function ensureAdmin(actor: Actor): void { if (!actor.roles.includes("admin")) throw new Error("forbidden"); }
export function rename(command: { id: string; actor: Actor }): { id: string } {
  ensureAdmin(command.actor);
  const template = templates.get(command.id)!;
  if (template.isDefault) throw new Error("default");
  return { id: template.id };
}
export function create(command: { id: string; name: string }): { id: string; name: string } {
  const duplicates = existing.filter((template) => template.id === command.id);
  if (duplicates.length > 0) throw new Error("duplicate");
  return { id: command.id, name: command.name };
}`},
			// A presence test exempts the throw under it from enclosing conditions, and data obtained without input (the
			// current time) does not connect a check to the result.
			{Code: `class Scope { constructor(readonly id: string) {} }
type Site = { id: string; open: boolean };
const sites = new Map<string, Site>();
function capability(at: Date, profile: { id: string }): "ok" | "late" { return at.getTime() > 0 && profile.id ? "ok" : "late"; }
export function list(query: { scope: Scope | string }): { site: Scope | string } {
  if (query.scope instanceof Scope) {
    const site = sites.get(query.scope.id);
    if (!site) throw new Error("missing");
  }
  return { site: query.scope };
}
export function respond(command: { profile: { id: string }; note: string }): { at: Date; note: string } {
  const now = new Date();
  if (capability(now, command.profile) === "late") throw new Error("late");
  return { at: now, note: command.note };
}`},
			// A method call checks an unknown part of its receiver, not a specific field; a helper's check maps onto the
			// argument's checked fields; storing into one field of a local does not reach its other fields.
			{Code: `class Actor { constructor(readonly id: string, readonly roles: string[]) {} }
type Entity = { id: string; teamId: string; name: string; updatedAt: string };
function ensureMember(actor: Actor): void { if (!actor.roles.includes("member")) throw new Error("forbidden"); }
function load(id: string): Entity { return { id, teamId: "", name: "", updatedAt: "" }; }
export function folder(command: { folder: { ids(): string[]; siteId: string } }): { siteId: string } {
  if (!command.folder.ids().includes("x")) throw new Error("missing");
  return { siteId: command.folder.siteId };
}
export function member(actor: Actor): { actorId: string } { ensureMember(actor); return { actorId: actor.id }; }
export function edit(command: { id: string; name: string; teamId: string }): { updatedAt: string } {
  const entity = load(command.id);
  entity.name = command.name;
  if (entity.teamId !== command.teamId) throw new Error("other team");
  return { updatedAt: entity.updatedAt };
}`},
			// The presence part of a combined condition checks no data: only the site's workspace is checked, and the result
			// does not read it.
			{Code: `type Site = { id: string; teamId: string; name: string };
const sites = new Map<string, Site>();
export function get(query: { siteId: string; teamId: string }): { id: string; name: string } {
  const site = sites.get(query.siteId);
  if (!site || site.teamId !== query.teamId) throw new Error("not found");
  return { id: site.id, name: site.name };
}`},
			// Type tests establish what the narrowed type carries: instanceof, in, typeof, and type guards.
			{Code: `class Draft { constructor(readonly id: string) {} }
class Published { constructor(readonly id: string, readonly url: string) {} }
type Serialized = { id: string; url: string };
export function serialize(model: Draft | Published): Serialized {
  if (model instanceof Draft) throw new Error("drafts are not serialized");
  return { id: model.id, url: model.url };
}
export function keyed(input: { id: string } | { key: string }): { id: string } {
  if (!("id" in input)) throw new Error("no id");
  return { id: input.id };
}
export function typed(input: { value: unknown }): { value: number } {
  if (typeof input.value !== "number") throw new Error("not a number");
  return { value: input.value };
}
export function listed(input: { values: unknown }): { values: unknown[] } {
  if (!Array.isArray(input.values)) throw new Error("not a list");
  return { values: input.values };
}`},
			// A check whose subject the result does not contain: the result is a constant chosen by the check.
			{Code: `type Summary = { summary: string };
export function tier(input: string): Summary { if (input !== "a") throw new Error("tier"); return { summary: "fixed" }; }`},
			// Recursion terminates and counts as not rejecting.
			{Code: `type Node = { next?: Node };
export function walk(input: Node): Node { return input.next ? walk(input.next) : { ...input }; }`},
			// An empty options object is accepted.
			{Code: `export const value = 1;`, Options: rule_tester.OptionsFromJSON[any](`{}`)},
		},
		[]rule_tester.InvalidTestCase{
			// The documented example: an interface, a derived local, and a throw on input.
			{Code: `export interface Payload {
  readonly model: string;
  readonly tier?: string;
}
export function applySettings(before: Payload, input: unknown): Payload {
  if (!input || typeof input !== "object") throw new Error("payload must be an object");
  const after = input as Payload;
  if (after.model !== before.model) throw new Error("cannot change model");
  return { ...after };
}`, Errors: []rule_tester.InvalidTestCaseError{forgeable(5, 17, 30)}},
			// Equivalent declarations: object alias, Record alias, intersection, and inline return type.
			{Code: `type A = { model: string };
type R = Record<string, unknown>;
type Base = { id: string };
type I = Base & { model: string };
export function a(input: string): A { if (!input) throw new Error("x"); return { model: input }; }
export function r(input: unknown): R { if (!input) throw new Error("x"); return { ...(input as object) }; }
export function i(input: string): I { if (!input) throw new Error("x"); return { id: "", model: input }; }
export function n(input: string): { model: string } { if (!input) throw new Error("x"); return { model: input }; }`, Errors: []rule_tester.InvalidTestCaseError{
				forgeable(5, 17, 18), forgeable(6, 17, 18), forgeable(7, 17, 18), forgeable(8, 17, 18),
			}},
			// Async results, switch discriminants, destructured parameters, for-of bindings, and variable-assigned arrows.
			{Code: `type Summary = { summary: string };
export async function load(input: { text: string }): Promise<Summary> {
  const text = input.text.trim();
  if (!text) throw new Error("empty");
  return { summary: text };
}
export function tier(input: string): Summary {
  switch (input) {
    case "a": return { summary: input };
    default: throw new Error("tier");
  }
}
export function named({ name }: { name: string }): Summary { if (!name) throw new Error("x"); return { summary: name }; }
export function each(input: string[]): Summary { for (const item of input) { if (!item) throw new Error("x"); } return { summary: input.join() }; }
export const arrow = (input: string): Summary => { if (!input) throw new Error("x"); return { summary: input }; };`, Errors: []rule_tester.InvalidTestCaseError{
				forgeable(2, 23, 27), forgeable(7, 17, 21), forgeable(13, 17, 22), forgeable(14, 17, 21), forgeable(15, 14, 19),
			}},
			// Rejection through a method on this, and a private method.
			{Code: `type Tool = { name: string }; type Payload = Record<string, unknown>;
export class Policy {
  bind(contract: Tool, tool: Tool): Tool {
    this.assertCompatible(contract, tool);
    return { ...tool };
  }
  private settings(input: Record<string, unknown>): Payload {
    if (Object.keys(input).length > 3) throw new Error("x");
    return { ...input };
  }
  assertCompatible(a: Tool, b: Tool): void {
    if (a.name !== b.name) throw new Error("mismatch");
  }
}`, Errors: []rule_tester.InvalidTestCaseError{forgeable(3, 3, 7), forgeable(7, 11, 19)}},
			// Cross-file: the dependency throws on its parameter.
			{Code: bind, Files: map[string]string{
				"policy.ts": `export type Tool = { name: string; run(): void };
export function assertTool(tool: Tool): void { if (!tool.name) throw new Error("unnamed tool"); }`,
			}, Errors: []rule_tester.InvalidTestCaseError{forgeable(2, 17, 21)}},
			// The checked data reaches the result through an object argument, a method call on it, and a mutation of a local.
			{Code: `type Order = { items: number[] };
function sum(values: number[]): number { return values.reduce((a, b) => a + b, 0); }
export function order(input: { items: number[] }): Order {
  const total = sum(input.items);
  if (total > 10) throw new Error("too large");
  return { items: input.items };
}
export function collect(input: string[]): { values: string[] } {
  const out: { values: string[] } = { values: [] };
  for (const value of input) {
    if (!value.trim()) throw new Error("blank");
    out.values.push(value);
  }
  return out;
}`, Errors: []rule_tester.InvalidTestCaseError{forgeable(3, 17, 22), forgeable(8, 17, 24)}},
			// Destructuring: object patterns keep property chains, array patterns with holes hold the whole source.
			{Code: `type Pair = { first: string; second: string };
export function pair(input: { left: string; rest: string[] }): Pair {
  const { left } = input;
  const [, second] = input.rest;
  if (!left || !second) throw new Error("x");
  return { first: left, second };
}`, Errors: []rule_tester.InvalidTestCaseError{forgeable(2, 17, 21)}},
			// A method call on the returned value, and a helper checking the returned field, are checks of the result.
			{Code: `type Named = { name: string };
function ensureNamed(target: { name: string }): void { if (!target.name) throw new Error("unnamed"); }
export function valid(input: { isValid(): boolean; name: string }): Named { if (!input.isValid()) throw new Error("x"); return { ...input }; }
export function named(input: { name: string; id: string }): Named { ensureNamed(input); return { name: input.name }; }`,
				Errors: []rule_tester.InvalidTestCaseError{forgeable(3, 17, 22), forgeable(4, 17, 22)}},
			// A response obtained through a function-typed parameter is its own root; checking it and returning its data
			// is a check of the result.
			{Code: `type Usage = { tokens: number };
type Message = { stopReason: string; content: { type: string; text: string }[]; usage: Usage };
type Completion = (context: string, options: { max: number }) => Promise<Message>;
export async function generate(input: { context: string; max: number }, complete: Completion): Promise<{ summary: string; usage: Usage }> {
  const response = await complete(input.context, { max: input.max });
  if (response.stopReason !== "stop") throw new Error("failed");
  const summary = response.content.filter((b) => b.type === "text").map((b) => b.text).join("\n");
  if (!summary.trim()) throw new Error("empty");
  return { summary, usage: response.usage };
}`, Errors: []rule_tester.InvalidTestCaseError{forgeable(4, 23, 31)}},
			// A type test combined with a content test still checks content.
			{Code: `export function ranged(input: { value: unknown }): { value: number } {
  if (typeof input.value !== "number" || input.value < 0) throw new Error("negative");
  return { value: input.value };
}`, Errors: []rule_tester.InvalidTestCaseError{forgeable(1, 17, 23)}},
			// A presence test combined with a content test still checks content.
			{Code: `type Payload = { model: string };
const cache = new Map<string, Payload>();
export function get(key: string): Payload { const found = cache.get(key); if (!found || !found.model) throw new Error("x"); return found; }`, Errors: []rule_tester.InvalidTestCaseError{forgeable(3, 17, 20)}},
		},
	)
}
