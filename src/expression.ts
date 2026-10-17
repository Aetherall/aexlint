import type { ESTree } from "@oxlint/plugins";

const wrapperTypes = [
  "ParenthesizedExpression",
  "ChainExpression",
  "TSAsExpression",
  "TSTypeAssertion",
  "TSSatisfiesExpression",
  "TSNonNullExpression",
  "TSInstantiationExpression",
] as const;

type ExpressionWrapper = Extract<ESTree.Node, { type: (typeof wrapperTypes)[number] }>;

function isExpressionWrapper(node: ESTree.Node): node is ExpressionWrapper {
  return wrapperTypes.some((type) => node.type === type);
}

export function isFunctionNode(
  node: ESTree.Node,
): node is ESTree.Function | ESTree.ArrowFunctionExpression {
  return ["FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression"].includes(
    node.type,
  );
}

export type Invocation = ESTree.CallExpression | ESTree.NewExpression;

export function isInvocation(node: ESTree.Node): node is Invocation {
  return node.type === "CallExpression" || node.type === "NewExpression";
}

export function isClass(node: ESTree.Node): node is ESTree.Class {
  return node.type === "ClassDeclaration" || node.type === "ClassExpression";
}

export function isComputationBoundary(node: ESTree.Node): boolean {
  return isFunctionNode(node) || isClass(node) || node.type === "ObjectExpression";
}

export interface InvocationStage {
  receiver: ESTree.Node | null;
  inputs: ESTree.Node[];
}

function fluentStage(invoked: ESTree.Node, inputs: ESTree.Node[]): InvocationStage {
  const callee = unwrap(invoked);
  if (callee.type !== "MemberExpression") return { receiver: null, inputs: [callee, ...inputs] };
  return { receiver: callee.object, inputs: [...inputs, ...computedProperty(callee)] };
}

export function invocationStage(
  visitorKeys: Readonly<Record<string, readonly string[]>>,
  node: ESTree.Node,
): InvocationStage | undefined {
  switch (node.type) {
    case "CallExpression":
      return fluentStage(node.callee, [...node.arguments]);
    case "TaggedTemplateExpression":
      return fluentStage(node.tag, [node.quasi]);
    case "NewExpression":
      return { receiver: null, inputs: [node.callee, ...node.arguments] };
    case "ImportExpression":
      return { receiver: null, inputs: childNodes(visitorKeys, node) };
    default:
      return undefined;
  }
}

export function continuesGroup(child: ESTree.Node, group: ESTree.LogicalExpression): boolean {
  const expression = unwrap(child);
  return expression.type === "LogicalExpression" && expression.operator === group.operator;
}

export function closestAncestor(
  node: ESTree.Node,
  stop: (ancestor: ESTree.Node) => boolean,
): ESTree.Node | null {
  let ancestor = node.parent;
  while (ancestor !== null && !stop(ancestor)) ancestor = ancestor.parent;
  return ancestor;
}

export function isLogicalDecision(
  node: ESTree.Node,
): node is ESTree.LogicalExpression & { operator: "&&" | "||" } {
  return node.type === "LogicalExpression" && node.operator !== "??";
}

export function isNegation(node: ESTree.Node): node is ESTree.UnaryExpression & { operator: "!" } {
  return node.type === "UnaryExpression" && node.operator === "!";
}

export function isSignedExpression(
  node: ESTree.Node,
): node is ESTree.UnaryExpression & { operator: "+" | "-" } {
  return node.type === "UnaryExpression" && ["+", "-"].includes(node.operator);
}

export function computedProperty(node: ESTree.MemberExpression): ESTree.Node[] {
  return node.computed ? [node.property] : [];
}

export function unwrap(node: ESTree.Node): ESTree.Node {
  while (isExpressionWrapper(node)) node = node.expression;
  return node;
}

export function childNodes(
  visitorKeys: Readonly<Record<string, readonly string[]>>,
  node: ESTree.Node,
  keep: (key: string) => boolean = () => true,
): ESTree.Node[] {
  const fields = node as unknown as Record<string, ESTree.Node | (ESTree.Node | null)[] | null>;
  const children: ESTree.Node[] = [];
  for (const key of (visitorKeys[node.type] ?? []).filter(keep)) {
    const value = fields[key];
    for (const child of Array.isArray(value) ? value : [value]) if (child) children.push(child);
  }
  return children;
}
