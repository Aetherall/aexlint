import { defineRule, type ESTree } from "@oxlint/plugins";
import {
  childNodes,
  closestAncestor,
  continuesGroup,
  invocationStage,
  isComputationBoundary,
  isLogicalDecision,
  isSignedExpression,
  unwrap,
} from "../../src/expression.ts";

function isIndependentContainer(node: ESTree.Node): boolean {
  return ["ArrayExpression", "JSXElement", "JSXFragment", "JSXOpeningElement"].includes(node.type);
}

function isAddition(node: ESTree.Node): node is ESTree.BinaryExpression & { operator: "+" } {
  return node.type === "BinaryExpression" && node.operator === "+";
}

function isPlainOrDefaultingAssignment(node: ESTree.AssignmentExpression): boolean {
  return node.operator === "=" || node.operator === "??=";
}

function isTransparentUnary(node: ESTree.UnaryExpression): boolean {
  return node.operator === "!" || node.operator === "typeof";
}

type NodeOfType<T extends ESTree.Node["type"], N = ESTree.Node> = N extends { type: infer K }
  ? T extends K
    ? N
    : never
  : never;

type ByNodeType<Result> = {
  [T in ESTree.Node["type"]]?: (node: NodeOfType<T>) => Result;
};

function handlerFor<Result>(
  table: ByNodeType<Result>,
  node: ESTree.Node,
): ((node: ESTree.Node) => Result) | undefined {
  return table[node.type] as ((node: ESTree.Node) => Result) | undefined;
}

function computedKey(node: { computed: boolean; key: ESTree.Node }): ESTree.Node[] {
  return node.computed ? [node.key] : [];
}

function unaryCost(node: ESTree.UnaryExpression): number {
  if (isTransparentUnary(node)) return 0;
  if (!isSignedExpression(node)) return 1;
  if (node.argument.type !== "Literal") return 1;
  return typeof node.argument.value === "number" || "bigint" in node.argument ? 0 : 1;
}

const operationCosts: ByNodeType<number> = {
  BinaryExpression: () => 1,
  ConditionalExpression: () => 1,
  UpdateExpression: () => 1,
  YieldExpression: () => 1,
  LogicalExpression: (node) => (node.operator === "??" ? 0 : 1),
  AssignmentExpression: (node) => (isPlainOrDefaultingAssignment(node) ? 0 : 1),
  UnaryExpression: unaryCost,
};

function ownCost(node: ESTree.Node): number {
  return handlerFor(operationCosts, node)?.(node) ?? 0;
}

type Region = ESTree.Node | null | undefined;

const measuredRegions: ByNodeType<Region[]> = {
  VariableDeclarator: (node) => [node.init],
  ExpressionStatement: (node) => [node.expression],
  ReturnStatement: (node) => [node.argument],
  ThrowStatement: (node) => [node.argument],
  ArrowFunctionExpression: (node) => (node.body.type === "BlockStatement" ? [] : [node.body]),
  IfStatement: (node) => [node.test],
  WhileStatement: (node) => [node.test],
  DoWhileStatement: (node) => [node.test],
  ForStatement: (node) => [
    node.init?.type === "VariableDeclaration" ? null : node.init,
    node.test,
    node.update,
  ],
  ForInStatement: (node) => [node.right],
  ForOfStatement: (node) => [node.right],
  SwitchStatement: (node) => [node.discriminant],
  SwitchCase: (node) => [node.test],
  AssignmentPattern: (node) => [node.right],
  PropertyDefinition: (node) => [node.value, ...computedKey(node)],
  AccessorProperty: (node) => [node.value, ...computedKey(node)],
  MethodDefinition: (node) => computedKey(node),
  ClassDeclaration: (node) => [node.superClass],
  ClassExpression: (node) => [node.superClass],
  Decorator: (node) => [node.expression],
  ExportDefaultDeclaration: (node) => [node.declaration],
  TSExportAssignment: (node) => [node.expression],
  TSEnumMember: (node) => [node.initializer],
  WithStatement: (node) => [node.object],
  ObjectExpression: (node) =>
    node.properties.map((property) =>
      property.type === "Property" && !property.computed ? property.value : property,
    ),
};

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Limit computation packed into one expression while preserving fluent and function boundaries.",
      requiresTypeChecking: false,
    },
    schema: {
      type: "array",
      minItems: 1,
      maxItems: 1,
      items: [
        {
          type: "object",
          properties: { max: { type: "integer", minimum: 1, maximum: Number.MAX_SAFE_INTEGER } },
          required: ["max"],
          additionalProperties: false,
        },
      ],
    },
    messages: {
      tooComplex:
        "Expression workload is {{score}}; maximum is {{max}}. Separate the computation into meaningful steps.",
    },
  },
  create(context) {
    const option = context.options[0] as { max?: unknown } | undefined;
    const max = option?.max;
    if (typeof max !== "number" || !Number.isSafeInteger(max) || max < 1) {
      throw new Error("max-expression-complexity requires a positive safe integer max option.");
    }
    const scores = new WeakMap<ESTree.Node, number>();
    const stringValues = new WeakSet<ESTree.Node>();
    const regions = new Set<ESTree.Node>();
    const scoreOf = (node: ESTree.Node) => scores.get(node) ?? 0;
    const childrenOf = (node: ESTree.Node) => childNodes(context.sourceCode.visitorKeys, node);
    const sum = (nodes: ESTree.Node[]) => nodes.reduce((total, node) => total + scoreOf(node), 0);
    return {
      "*:exit"(node) {
        for (const region of handlerFor(measuredRegions, node)?.(node) ?? []) {
          if (region != null) regions.add(region);
        }
        if (isComputationBoundary(node)) {
          scores.set(node, 0);
          return;
        }
        const stage = invocationStage(context.sourceCode.visitorKeys, node);
        if (stage) {
          const receiverScore = stage.receiver ? scoreOf(stage.receiver) : 0;
          scores.set(node, Math.max(receiverScore, 1 + sum(stage.inputs)));
          return;
        }
        if (isIndependentContainer(node)) {
          let score = 0;
          for (const child of childrenOf(node)) score = Math.max(score, scoreOf(child));
          scores.set(node, score);
          return;
        }
        if (node.type === "ConditionalExpression") {
          const branchScore = Math.max(scoreOf(node.consequent), scoreOf(node.alternate));
          const selectionScore = 1 + scoreOf(node.test) + branchScore;
          scores.set(node, selectionScore);
          return;
        }
        if (isLogicalDecision(node)) {
          const term = (child: ESTree.Node) => {
            const childScore = scoreOf(child);
            return continuesGroup(child, node) ? childScore - 1 : childScore;
          };
          const groupScore = 1 + term(node.left) + term(node.right);
          scores.set(node, groupScore);
          return;
        }
        const expression = unwrap(node);
        const stringLiteral = node.type === "Literal" && typeof node.value === "string";
        if (stringLiteral || node.type === "TemplateLiteral" || stringValues.has(expression)) {
          stringValues.add(node);
        }
        const addition = isAddition(node);
        const concatenation =
          addition && (stringValues.has(node.left) || stringValues.has(node.right));
        if (concatenation) stringValues.add(node);
        const operationCost = concatenation ? 0 : ownCost(node);
        const childScore = sum(childrenOf(node));
        scores.set(node, operationCost + childScore);
      },
      "Program:exit"() {
        const excessive = new Set([...regions].filter((node) => scoreOf(node) > max));
        for (const node of excessive) {
          const stop = (ancestor: ESTree.Node) =>
            isComputationBoundary(ancestor) || excessive.has(ancestor);
          const outer = closestAncestor(node, stop);
          if (outer !== null && excessive.has(outer)) continue;
          context.report({ node, messageId: "tooComplex", data: { score: scoreOf(node), max } });
        }
      },
    };
  },
});
