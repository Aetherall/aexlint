package require_interface_implementations

import (
	"path/filepath"
	"testing"

	"github.com/typescript-eslint/tsgolint/internal/rule"
	"github.com/typescript-eslint/tsgolint/internal/rule_tester"
)

func single(line, column, endColumn int) rule_tester.InvalidTestCaseError {
	return rule_tester.InvalidTestCaseError{MessageId: "singleImplementation", Line: line, Column: column, EndColumn: endColumn}
}

func unimplemented(line, column, endColumn int) rule_tester.InvalidTestCaseError {
	return rule_tester.InvalidTestCaseError{MessageId: "unimplementedInterface", Line: line, Column: column, EndColumn: endColumn}
}

func TestRule(t *testing.T) {
	root, err := filepath.Abs("../../fixtures")
	if err != nil {
		t.Fatal(err)
	}
	shape := `export interface Shape { area(): number }
export class Circle implements Shape { area() { return 1; } }`
	rule_tester.RunRuleTester(root, "tsconfig.json", t, &Rule,
		[]rule_tester.ValidTestCase{
			// A test double declared with implements is a second implementation.
			{Code: `export interface Clock { now(): number }
export class SystemClock implements Clock { now() { return Date.now(); } }`, Files: map[string]string{
				"clock.test.ts": `import type { Clock } from "./file.js"; export class FixedClock implements Clock { now() { return 0; } }`,
			}},
			// Classes implementing an extending interface implement the extended one.
			{Code: `export interface Animal { name: string }`, Files: map[string]string{
				"pets.ts": `import type { Animal } from "./file.js";
export interface Pet extends Animal { owner: string }
export class Dog implements Pet { name = ""; owner = ""; }
export class Cat implements Pet { name = ""; owner = ""; }`,
			}},
			// A subclass implements its base class's interfaces.
			{Code: `export interface Shape { area(): number }
export class Base implements Shape { area() { return 0; } }
export class Square extends Base {}`},
			// Type aliases, augmentations of library interfaces, and interfaces merged with a class are not checked.
			{Code: `export type Props = { label: string };
declare global { interface Array<T> { last(): T | undefined } }
export interface Point { z: number }
export class Point { x = 0 }`},
			// Import aliases, type-only re-exports, namespace imports, type arguments, and class expressions.
			{Code: `export interface Repo<T> { get(): T }`, Files: map[string]string{
				"a.ts": `import { type Repo as Store } from "./file.js"; export class A implements Store<string> { get() { return ""; } }`,
				"b.ts": `export type { Repo as Store } from "./file.js";`,
				"c.ts": `import * as stores from "./b.js"; export const C = class implements stores.Store<number> { get() { return 1; } };`,
			}},
			// Circular inheritance terminates; both classes implement the interface.
			{Code: `export interface I {}
export class A extends B implements I {}
export class B extends A {}`},
			// An empty options object is accepted.
			{Code: `export interface I {} export class A implements I {} export class B implements I {}`, Options: rule_tester.OptionsFromJSON[any](`{}`)},
			// Cross-file: the dependency's interface extends Shape, so Cube implements Shape too.
			{Code: shape, Files: map[string]string{
				"solid.ts": `import type { Shape } from "./file.js";
export interface Solid extends Shape { volume(): number }
export class Cube implements Solid { area() { return 1; } volume() { return 1; } }`,
			}},
		},
		[]rule_tester.InvalidTestCase{
			// One implementing class in another file.
			{Code: `export interface Repository { find(): string }`, Files: map[string]string{
				"a.ts": `import type { Repository } from "./file.js"; export class PostgresRepository implements Repository { find() { return ""; } }`,
			}, Errors: []rule_tester.InvalidTestCaseError{single(1, 18, 28)}},
			// A data shape no class implements.
			{Code: `interface ButtonProps { label: string }`, Errors: []rule_tester.InvalidTestCaseError{unimplemented(1, 11, 22)}},
			// One class reaching an interface through several paths counts once.
			{Code: `export interface Shape {}
export interface Solid extends Shape {}
export class Cube implements Shape, Solid {}`, Errors: []rule_tester.InvalidTestCaseError{single(1, 18, 23), single(2, 18, 23)}},
			// Structural compatibility, typed object literals, and factories are not implementations.
			{Code: `export interface Named { name: string }
export class A implements Named { name = "" }
export class B { name = "" }
export const c: Named = { name: "" };
export const make = (): Named => new B();`, Errors: []rule_tester.InvalidTestCaseError{single(1, 18, 23)}},
			// A mixin call in extends contributes nothing.
			{Code: `export interface Named { name: string }
export class A implements Named { name = "" }
const mix = <T extends new (...args: any[]) => object>(base: T) => class extends base {};
export class B extends mix(A) {}`, Errors: []rule_tester.InvalidTestCaseError{single(1, 18, 23)}},
			// Merged declarations are reported on each.
			{Code: `interface Box { a: number }
interface Box { b: number }`, Errors: []rule_tester.InvalidTestCaseError{unimplemented(1, 11, 14), unimplemented(2, 11, 14)}},
			// Interfaces nested in namespaces and functions.
			{Code: `export namespace N { export interface Inner {} }
export function f() { interface Local {} return 1; }`, Errors: []rule_tester.InvalidTestCaseError{unimplemented(1, 39, 44), unimplemented(2, 33, 38)}},
			// An anonymous class expression as the only implementation.
			{Code: `export interface Repo {}
export const R = class implements Repo {};`, Errors: []rule_tester.InvalidTestCaseError{single(1, 18, 22)}},
			// TSX.
			{Code: `export interface Props { label: string }
export const Button = (props: Props) => <button>{props.label}</button>;`, Tsx: true, Errors: []rule_tester.InvalidTestCaseError{unimplemented(1, 18, 23)}},
			// Cross-file: the dependency's interface no longer extends Shape.
			{Code: shape, Files: map[string]string{
				"solid.ts": `export interface Solid { volume(): number }
export class Cube implements Solid { area() { return 1; } volume() { return 1; } }`,
			}, Errors: []rule_tester.InvalidTestCaseError{single(1, 18, 23)}},
		},
	)
}

func TestOptions(t *testing.T) {
	for _, options := range []any{[]any{}, true, map[string]any{"min": float64(3)}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Run accepted invalid options %v", options)
				}
			}()
			Rule.Run(rule.RuleContext{}, options)
		}()
	}
}
