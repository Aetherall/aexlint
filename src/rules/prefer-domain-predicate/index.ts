import { defineRule, type ESTree, type Scope } from "@oxlint/plugins";
import {
  isFunctionNode,
  isLogicalDecision,
  isNegation,
  isSignedExpression,
  unwrap,
} from "../../expression.ts";
import { isValidationBody } from "./validation.ts";

type Facts = Map<string, Set<string>>;

function isSetIdentifier(node: ESTree.Node): boolean {
  return node.type === "Identifier" && node.name === "Set";
}

function isMembershipMethod(method: string | undefined): boolean {
  return method === "includes" || method === "has";
}

function literal(node: ESTree.Node): string | undefined {
  const value = unwrap(node);
  if (value.type === "TemplateLiteral" && value.expressions.length === 0) {
    return `string:${value.quasis[0]?.value.cooked}`;
  }
  if (isSignedExpression(value)) {
    const argument = unwrap(value.argument);
    if (argument.type !== "Literal") return;
    if (typeof argument.value === "number") {
      const signed = value.operator === "-" ? -argument.value : argument.value;
      return `number:${signed}`;
    }
    if ("bigint" in argument && value.operator === "-") return `bigint:${-BigInt(argument.bigint)}`;
    return;
  }
  if (value.type !== "Literal") return;
  if ("bigint" in value) return `bigint:${BigInt(value.bigint)}`;
  if (typeof value.value === "string") return `string:${value.value}`;
  if (typeof value.value === "number") return `number:${value.value}`;
}

function memberName(node: ESTree.MemberExpression): string | undefined {
  if (!node.computed && node.property.type === "Identifier") return node.property.name;
  if (node.property.type !== "Literal") return;
  if (typeof node.property.value === "string") return node.property.value;
  if (typeof node.property.value === "number") return String(node.property.value);
}

function pathOf(node: ESTree.Node): string[] | undefined {
  const value = unwrap(node);
  if (value.type === "Identifier") return [value.name];
  if (value.type === "ThisExpression") return ["this"];
  if (value.type !== "MemberExpression") return;
  const name = memberName(value);
  if (name === undefined) return;
  const owner = pathOf(value.object);
  if (owner) return [...owner, name];
}

function factsFor(path: string[], values: string[]): Facts {
  const owner = path.length === 1 ? path : path.slice(0, -1);
  const subject = JSON.stringify(owner);
  const facts = values.map((value) => JSON.stringify([path, value]));
  return new Map([[subject, new Set(facts)]]);
}

function merge(left: Facts, right: Facts): Facts {
  const result = new Map(left);
  for (const [subject, facts] of right) {
    const previous = result.get(subject) ?? [];
    result.set(subject, new Set([...previous, ...facts]));
  }
  return result;
}

function compositionParent(node: ESTree.Node): ESTree.Node | undefined {
  const parent = node.parent;
  if (!parent) return;
  if (unwrap(parent) === node) return parent;
  if (isNegation(parent) || isLogicalDecision(parent)) return parent;
}

function hasName(node: ESTree.Node): boolean {
  if (node.type === "FunctionDeclaration") return node.id !== null;
  if (node.type === "FunctionExpression" && node.id !== null) return true;
  let parent = node.parent;
  while (parent && unwrap(parent) === node) {
    node = parent;
    parent = node.parent;
  }
  if (!parent) return false;
  switch (parent.type) {
    case "VariableDeclarator":
      return parent.id.type === "Identifier";
    case "Property":
    case "MethodDefinition":
    case "PropertyDefinition":
    case "AccessorProperty":
      return !parent.computed || literal(parent.key) !== undefined;
    case "AssignmentExpression":
      return parent.operator === "=" && pathOf(parent.left) !== undefined;
    default:
      return false;
  }
}

function isBooleanExit(node: ESTree.Statement): boolean {
  const statement = node.type === "BlockStatement" ? node.body[0] : node;
  if (node.type === "BlockStatement" && node.body.length !== 1) return false;
  if (statement?.type !== "ReturnStatement" || !statement.argument) return false;
  const value = unwrap(statement.argument);
  return value.type === "Literal" && typeof value.value === "boolean";
}

function isPredicatePrelude(node: ESTree.Statement): boolean {
  if (node.type === "VariableDeclaration") return true;
  if (node.type !== "IfStatement" || node.alternate) return false;
  return isBooleanExit(node.consequent);
}

function compositionRoot(node: ESTree.Node): ESTree.Node {
  let result = node;
  let parent = compositionParent(result);
  while (parent) {
    result = parent;
    parent = compositionParent(result);
  }
  return result;
}

function isPredicateResult(node: ESTree.Node): boolean {
  const result = compositionRoot(node);
  const container = result.parent;
  if (container?.type === "ArrowFunctionExpression") return hasName(container);
  if (container?.type !== "ReturnStatement") return false;
  const body = container.parent;
  if (body.type !== "BlockStatement" || body.body.at(-1) !== container) return false;
  if (!body.body.slice(0, -1).every(isPredicatePrelude)) return false;
  const fn = body.parent;
  if (!isFunctionNode(fn)) return false;
  return hasName(fn);
}

function validationBodyOf(node: ESTree.Node): ESTree.BlockStatement | undefined {
  const result = compositionRoot(node);
  let statement = result.parent;
  if (!statement) return;
  if (statement.type === "VariableDeclarator" && statement.init === result) {
    statement = statement.parent;
  } else if (statement.type !== "IfStatement" || statement.test !== result) return;
  const body = statement.parent;
  if (body?.type !== "BlockStatement") return;
  const fn = body.parent;
  if (!isFunctionNode(fn)) return;
  if (hasName(fn)) return body;
}

const comparisons: Record<string, string> = {
  "===": "eq",
  "==": "eq",
  "!==": "ne",
  "!=": "ne",
  "<": "lt",
  "<=": "le",
  ">": "gt",
  ">=": "ge",
};
const reversed: Record<string, string> = {
  eq: "eq",
  ne: "ne",
  lt: "gt",
  le: "ge",
  gt: "lt",
  ge: "le",
};

function comparisonFacts(node: ESTree.BinaryExpression | ESTree.PrivateInExpression): Facts {
  const operation = comparisons[node.operator];
  if (!operation) return new Map();
  let path = pathOf(node.left);
  let value = literal(node.right);
  let operator = operation;
  if (!path || value === undefined) {
    path = pathOf(node.right);
    value = literal(node.left);
    operator = reversed[operation]!;
  }
  if (!path || value === undefined) return new Map();
  return factsFor(path, [`${operator}:${value}`]);
}

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Experimental: identify classifications reconstructed from multiple representation checks of one subject.",
      requiresTypeChecking: false,
    },
    schema: [],
    messages: {
      preferPredicate:
        "This expression reconstructs a classification of {{subjects}} without naming it. Name the result or category, or use a named predicate or validator.",
    },
  },
  create(context) {
    const facts = new WeakMap<ESTree.Node, Facts>();
    const candidates = new Map<ESTree.Node, Set<string>>();
    const definitionOf = (node: ESTree.Node, name: string) => {
      let scope: Scope | null = context.sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(name);
        if (variable) return variable.defs[0];
        scope = scope.upper;
      }
    };
    const categoryEvidence = (node: ESTree.Node): string[] | undefined => {
      const value = unwrap(node);
      const name = pathOf(value);
      if (name) return [`category:${JSON.stringify(name)}`];
      if (value.type === "ArrayExpression") {
        const values: string[] = [];
        for (const element of value.elements) {
          if (!element) return;
          const key = literal(element);
          if (key === undefined) return;
          values.push(`eq:${key}`);
        }
        return values;
      }
      if (value.type !== "NewExpression") return;
      if (!isSetIdentifier(value.callee)) return;
      if (definitionOf(value.callee, "Set")) return;
      if (value.arguments.length !== 1) return;
      return categoryEvidence(value.arguments[0]!);
    };
    const membershipFacts = (node: ESTree.CallExpression): Facts => {
      const callee = unwrap(node.callee);
      if (callee.type !== "MemberExpression" || node.arguments.length !== 1) return new Map();
      const method = memberName(callee);
      if (!isMembershipMethod(method)) return new Map();
      const argument = node.arguments[0]!;
      const alternatives = categoryEvidence(callee.object);
      const path = pathOf(argument);
      if (alternatives && path) return factsFor(path, alternatives);
      const owner = pathOf(callee.object);
      const value = literal(argument);
      if (!owner || value === undefined) return new Map();
      return factsFor(owner, [`member:${value}`]);
    };
    const validationBodies = new WeakMap<ESTree.BlockStatement, boolean>();
    const isValidationCheck = (node: ESTree.Node): boolean => {
      const body = validationBodyOf(node);
      if (!body) return false;
      const cached = validationBodies.get(body);
      if (cached !== undefined) return cached;
      const valid = isValidationBody(body, (call) => membershipFacts(call).size > 0);
      validationBodies.set(body, valid);
      return valid;
    };
    const factsOf = (node: ESTree.Node): Facts => {
      let value = unwrap(node);
      while (isNegation(value)) value = unwrap(value.argument);
      return facts.get(value) ?? new Map();
    };
    return {
      "*:exit"(node) {
        let evidence: Facts;
        switch (node.type) {
          case "BinaryExpression":
            evidence = comparisonFacts(node);
            break;
          case "CallExpression":
            evidence = membershipFacts(node);
            break;
          case "LogicalExpression":
            if (node.operator === "??") return;
            evidence = merge(factsOf(node.left), factsOf(node.right));
            break;
          default:
            return;
        }
        facts.set(node, evidence);
        if (hasName(compositionRoot(node))) return;
        if (isPredicateResult(node) || isValidationCheck(node)) return;
        const subjects = new Set<string>();
        for (const [subject, checks] of evidence) if (checks.size >= 2) subjects.add(subject);
        if (subjects.size) candidates.set(node, subjects);
      },
      "Program:exit"() {
        for (const [node, subjects] of candidates) {
          const uncovered = new Set(subjects);
          let ancestor = compositionParent(node);
          while (ancestor) {
            for (const subject of candidates.get(ancestor) ?? []) uncovered.delete(subject);
            ancestor = compositionParent(ancestor);
          }
          if (!uncovered.size) continue;
          const names = [...uncovered].map((subject) => {
            const [root, ...members] = JSON.parse(subject) as string[];
            return root + members.map((member) => `[${JSON.stringify(member)}]`).join("");
          });
          context.report({
            node,
            messageId: "preferPredicate",
            data: { subjects: names.join(", ") },
          });
        }
      },
    };
  },
});
