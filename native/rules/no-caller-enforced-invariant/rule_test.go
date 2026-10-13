package no_caller_enforced_invariant

import (
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

func enforced(line, column, endColumn int) rule_tester.InvalidTestCaseError {
	return rule_tester.InvalidTestCaseError{MessageId: "checkedBeforeHandOff", Line: line, Column: column, EndColumn: endColumn}
}

const credential = `export class Credential {
  private constructor(readonly expiresAt: Date) {}
  static create(expiresAt: Date): Credential { return new Credential(expiresAt); }
}`

const checkedCredential = `export class Credential {
  private constructor(readonly expiresAt: Date) {}
  static create(expiresAt: Date): Credential {
    if (expiresAt <= new Date()) throw new Error("past");
    return new Credential(expiresAt);
  }
}`

const issue = `import { Credential } from "./credential.js";
export function issue(command: { expiresAt: string }) {
  const expiresAt = new Date(command.expiresAt);
  if (Number.isNaN(expiresAt.getTime()) || expiresAt <= new Date()) throw new Error("future");
  return Credential.create(expiresAt);
}`

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// Cross-file: the factory checks its parameter itself.
			{Code: issue, Files: map[string]string{"credential.ts": checkedCredential}},
			// The check is made after the call, on data not passed, or only narrows.
			{Code: `import { Credential } from "./credential.js";
export function after(command: { expiresAt: string }) {
  const expiresAt = new Date(command.expiresAt);
  const created = Credential.create(expiresAt);
  if (expiresAt <= new Date()) throw new Error("future");
  return created;
}
export function unrelated(command: { expiresAt: string; label: string }) {
  if (command.label.length > 20) throw new Error("label");
  return Credential.create(new Date(command.expiresAt));
}
export function narrowed(command: { expiresAt?: Date }) {
  if (!command.expiresAt) throw new Error("missing");
  return Credential.create(command.expiresAt);
}`, Files: map[string]string{"credential.ts": credential}},
			// Plain functions, library classes, and members of the same class are not hand-offs.
			{Code: `function store(value: string): string[] { return [value]; }
export function plain(input: { name: string }) {
  if (input.name.length > 10) throw new Error("long");
  const names = new Map<string, string>();
  names.set("name", input.name);
  return store(input.name);
}
export class Owner {
  private constructor(readonly name: string) {}
  static create(name: string): Owner {
    if (name.length > 10) throw new Error("long");
    return new Owner(name);
  }
}`},
			// Queries and other members that do not store the argument in the class's state take no ownership of it.
			{Code: `export class Site {
  private readonly parcels: string[] = [];
  hasParcel(name: string): boolean { return this.parcels.includes(name); }
}
export class Logger { warn(message: string): void { console.log(message); } }
export function join(site: Site, logger: Logger, input: { name: string }) {
  if (input.name.length > 20) throw new Error("long");
  logger.warn(input.name);
  return site.hasParcel(input.name);
}`},
			// Passing a value whole to an authorization check reads an unknown part of it, not the field handed off.
			{Code: `import { Credential } from "./credential.js";
type Actor = { id: string; ranks: string[] };
function can(actor: Actor, action: string): boolean { return actor.ranks.includes(action); }
export class Grant { private constructor(readonly actorId: string) {} static of(actorId: string): Grant { return new Grant(actorId); } }
export function grant(command: { actor: Actor }) {
  if (!can(command.actor, "grant")) throw new Error("forbidden");
  return Grant.of(command.actor.id);
}`, Files: map[string]string{"credential.ts": credential}},
			// Two getters on one object are different data; and a callee that builds a value type rejecting its parameter
			// checks it.
			{Code: `export class GivenName { private constructor(readonly value: string) {} static from(value: string): GivenName { if (!value.trim()) throw new Error("empty"); return new GivenName(value); } }
export class Person {
  private constructor(readonly first: GivenName, readonly last: string) {}
  static named(first: string, last: string): Person { return new Person(GivenName.from(first), last); }
}
type Source = { getFirst(): string; getLast(): string };
export function clone(source: Source) {
  const first = source.getFirst();
  const last = source.getLast();
  if (first.length > 20) throw new Error("long");
  return Person.named(first, last);
}`},
			// A subclass calling an inherited member is the owner's own code.
			{Code: `export class Store { private readonly items: string[] = []; add(item: string): void { this.items.push(item); } }
export class Names extends Store {
  addName(name: string): void { if (name.length > 10) throw new Error("long"); this.add(name); }
}
export class Shortlist extends Names { promote(name: string): void { if (!name.startsWith("a")) throw new Error("a"); super.add(name); } }
function Collection<T>(of: T) { return class { protected readonly items: T[] = [of]; protected remove(item: T): void { this.items.push(item); } }; }
export class Crops extends Collection("") { withdraw(crop: string): void { if (crop === "wheat") throw new Error("wheat"); this.remove(crop); } }`},
			// A check that cannot run before the call: another branch of the same if, a branch that returns first.
			{Code: `export class Ranger { private constructor(readonly by: string) {} static enlist(by: string): Ranger { return new Ranger(by); } reactivate(by: string): void { if (!by.startsWith("u")) throw new Error("user"); } }
export function join(existing: Ranger | undefined, input: { by: string }) {
  if (existing) {
    existing.reactivate(input.by);
  } else {
    return Ranger.enlist(input.by);
  }
}
export function create(input: { by: string; legacy: boolean }) {
  if (input.legacy) {
    if (input.by.length > 3) throw new Error("long");
    return undefined;
  }
  return Ranger.enlist(input.by);
}`},
			// Data computed from the checked data, or selected by it, is not the checked data.
			{Code: `export class Person { private constructor(readonly id: string) {} static withId(id: string): Person { return new Person(id); } }
export function parse(input: { last?: string }) {
  if (!input.last || input.last.length > 40) throw new Error("last");
  const id = input.last.split(":")[1] ?? "generated";
  return Person.withId(id);
}
export function pick(input: { format: string; a: string; b: string }) {
  if (input.format !== "a" && input.format !== "b") throw new Error("format");
  return Person.withId(input.format === "a" ? input.a : input.b);
}`},
			// An assertion helper is a narrowing test, and a factory called only to see whether it throws hands nothing over.
			{Code: `export class Path { private constructor(readonly value: string) {} static of(value: string): Path { return new Path(value); } }
function ensure(condition: unknown, message: string): asserts condition { if (!condition) throw new Error(message); }
export function locate(input: { folder?: { name: string } }) {
  ensure(input.folder, "not found");
  return Path.of(input.folder.name);
}
export function probe(input: { raw: string }) {
  if (input.raw.length > 100) throw new Error("long");
  try { Path.of(input.raw); return true; } catch { return false; }
}`},
			// An empty options object is accepted.
			{Code: `export const value = 1;`, Options: rule_tester.OptionsFromJSON[any](`{}`)},
			// The call runs only where the check cannot reject: the check needs existing, the call its absence.
			{Code: `export class Item { private constructor(readonly owner: string) {} static create(owner: string): Item { return new Item(owner); } }
export function ensure(existing: { owner: string } | undefined, owner: string) {
  if (existing && existing.owner !== owner) throw new Error("other owner");
  if (!existing) return Item.create(owner);
  return undefined;
}`},
		},
		[]rule_tester.InvalidTestCase{
			// The documented example: the factory accepts what its caller checked.
			{Code: issue, Files: map[string]string{"credential.ts": credential}, Errors: []rule_tester.InvalidTestCaseError{enforced(5, 28, 37)}},
			// An event-sourced class: a static factory returning the instance in a wrapper, and a method recording an event
			// through inherited machinery on this.
			{Code: `class Aggregate { protected readonly events: unknown[] = []; protected record(event: unknown): void { this.events.push(event); } }
export class Token extends Aggregate {
  static issue(expiresAt: Date): { token: Token; secret: string } { const token = new Token(); token.record({ expiresAt }); return { token, secret: "s" }; }
  extend(until: Date): void { this.record({ until }); }
}
export function issue(input: { expiresAt: Date }, token: Token) {
  if (input.expiresAt <= new Date()) throw new Error("past");
  token.extend(input.expiresAt);
  return Token.issue(input.expiresAt);
}`, Errors: []rule_tester.InvalidTestCaseError{enforced(8, 16, 31), enforced(9, 22, 37)}},
			// A check that also reads the actor, which the class is not given, is a precondition of the operation.
			{Code: `export class Inspection { private constructor(readonly farmId: string) {} static old(farmId: string): Inspection { return new Inspection(farmId); } }
export function countersign(command: { actor: { id: string; root: boolean }; farmId: string }) {
  if (!command.actor.root && command.actor.id !== command.farmId) throw new Error("other farm");
  return Inspection.old(command.farmId);
}`, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "checkedBeforeHandOff", Line: 4, Column: 25, EndColumn: 39}}},
			// A method checking its receiver's state (the actor's permission) reads context the class is not given, even when
			// the method's arguments are handed to it.
			{Code: `export class Actor { constructor(readonly id: string, private readonly ranks: string[]) {} ensure(crewId: string): void { if (!this.ranks.includes(crewId)) throw new Error("forbidden"); } }
export class Leaflet { private constructor(readonly crewId: string, readonly by: string) {} static send(crewId: string, by: string): Leaflet { return new Leaflet(crewId, by); } }
export function send(command: { actor: Actor; crewId: string }) {
  command.actor.ensure(command.crewId);
  return Leaflet.send(command.crewId, command.actor.id);
}`, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "checkedBeforeHandOff", Line: 5, Column: 23, EndColumn: 37}}},
			// An access check on the actor, made by a method of the site handed over, reads the actor's state, which the class
			// is not given: only the actor's identifier is.
			{Code: `export class Actor { constructor(readonly userId: string, private readonly crews: string[]) {} canAccessCrew(crewId: string): boolean { return this.crews.includes(crewId); } }
export class Site { constructor(readonly id: string, readonly crewId: string) {} ensureMayAccess(actor: Actor): void { if (!actor.canAccessCrew(this.crewId)) throw new Error("forbidden"); } }
export class Leaflet { private constructor(readonly site: Site, readonly by: string) {} static send(input: { site: Site; by: string }): Leaflet { return new Leaflet(input.site, input.by); } }
const sites = new Map<string, Site>();
export function send(command: { actor: Actor; siteId: string }) {
  const { actor, siteId } = command;
  const site = sites.get(siteId)!;
  site.ensureMayAccess(actor);
  return Leaflet.send({ site, by: actor.userId });
}`, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "checkedBeforeHandOff", Line: 9, Column: 23, EndColumn: 49}}},
			// A check reading an injected service reads data the class does not receive.
			{Code: `export class Ranger { private constructor(readonly crewId: string) {} static enlist(crewId: string): Ranger { return new Ranger(crewId); } }
export class Handler {
  constructor(private readonly registrations: { enabled(crewId: string): boolean }) {}
  run(command: { crewId: string }) {
    if (!this.registrations.enabled(command.crewId)) throw new Error("not available");
    return Ranger.enlist(command.crewId);
  }
}`, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "checkedBeforeHandOff", Line: 6, Column: 26, EndColumn: 40}}},
			// A check nested in a condition (applied only to some actors) still precedes a call after it.
			{Code: `export class Response { private constructor(readonly owner: string) {} static of(owner: string): Response { return new Response(owner); } }
export function read(query: { root: boolean; owner: string }) {
  if (!query.root) {
    if (query.owner.length > 20) throw new Error("owner");
  }
  return Response.of(query.owner);
}`, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "checkedBeforeHandOff", Line: 6, Column: 22, EndColumn: 33}}},
			// A constructor that stores a checked field of the argument, and a method changing state with a checked value.
			{Code: `export class Account {
  constructor(readonly email: string) {}
  private name = "";
  rename(name: string): void { this.name = name; }
}
export function open(input: { email: string; name: string }) {
  if (!input.email.includes("@")) throw new Error("email");
  const account = new Account(input.email);
  if (input.name.trim() === "") throw new Error("name");
  account.rename(input.name);
  return account;
}`, Errors: []rule_tester.InvalidTestCaseError{enforced(8, 31, 42), enforced(10, 18, 28)}},
			// Guards that can both hold keep the check before the call.
			{Code: `export class Item { private constructor(readonly owner: string) {} static create(owner: string): Item { return new Item(owner); } }
export function ensure(existing: { owner: string } | undefined, owner: string) {
  if (existing && existing.owner !== owner) throw new Error("other owner");
  if (existing) return Item.create(owner);
  return undefined;
}`, Errors: []rule_tester.InvalidTestCaseError{{MessageId: "checkedBeforeHandOff"}}},
		},
	)
}
