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

export function invokedExpression(
  node: ESTree.CallExpression | ESTree.TaggedTemplateExpression,
): ESTree.Node {
  return unwrap(node.type === "CallExpression" ? node.callee : node.tag);
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
