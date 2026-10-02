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

export function unwrap(node: ESTree.Node): ESTree.Node {
  while (isExpressionWrapper(node)) node = node.expression;
  return node;
}
