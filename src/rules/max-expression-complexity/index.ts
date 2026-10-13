import { defineRule, type ESTree } from "@oxlint/plugins";
import {
  computedProperty,
  isLogicalDecision,
  isSignedExpression,
  unwrap,
} from "../../expression.ts";

function isBoundary(node: ESTree.Node): boolean {
  return [
    "FunctionDeclaration",
    "FunctionExpression",
    "ArrowFunctionExpression",
    "ClassDeclaration",
    "ClassExpression",
    "ObjectExpression",
  ].includes(node.type);
}

function isIndependentContainer(node: ESTree.Node): boolean {
  return ["ArrayExpression", "JSXElement", "JSXFragment", "JSXOpeningElement"].includes(node.type);
}

function canHaveFluentReceiver(
  node: ESTree.Node,
): node is ESTree.CallExpression | ESTree.TaggedTemplateExpression {
  return node.type === "CallExpression" || node.type === "TaggedTemplateExpression";
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
  CallExpression: () => 1,
  NewExpression: () => 1,
  TaggedTemplateExpression: () => 1,
  ImportExpression: () => 1,
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
    const childrenOf = (node: ESTree.Node): ESTree.Node[] => {
      const fields = node as unknown as Record<string, ESTree.Node | ESTree.Node[] | null>;
      const children: ESTree.Node[] = [];
      for (const key of context.sourceCode.visitorKeys[node.type] ?? []) {
        const child = fields[key];
        if (Array.isArray(child)) {
          for (const element of child) if (element != null) children.push(element);
        } else if (child != null) children.push(child);
      }
      return children;
    };
    const sum = (nodes: ESTree.Node[]) => nodes.reduce((total, node) => total + scoreOf(node), 0);
    return {
      "*:exit"(node) {
        for (const region of handlerFor(measuredRegions, node)?.(node) ?? []) {
          if (region != null) regions.add(region);
        }
        if (isBoundary(node)) {
          scores.set(node, 0);
          return;
        }
        if (canHaveFluentReceiver(node)) {
          const callee = unwrap(node.type === "CallExpression" ? node.callee : node.tag);
          if (callee.type === "MemberExpression") {
            const inputs: ESTree.Node[] =
              node.type === "CallExpression" ? [...node.arguments] : [node.quasi];
            inputs.push(...computedProperty(callee));
            const stageScore = 1 + sum(inputs);
            scores.set(node, Math.max(scoreOf(callee.object), stageScore));
            return;
          }
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
            const expression = unwrap(child);
            const sameGroup =
              expression.type === "LogicalExpression" && expression.operator === node.operator;
            const childScore = scoreOf(child);
            return sameGroup ? childScore - 1 : childScore;
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
          let ancestor = node.parent;
          while (ancestor !== null && !isBoundary(ancestor) && !excessive.has(ancestor)) {
            ancestor = ancestor.parent;
          }
          if (ancestor !== null && excessive.has(ancestor)) continue;
          context.report({ node, messageId: "tooComplex", data: { score: scoreOf(node), max } });
        }
      },
    };
  },
});
