import { defineRule, type ESTree } from "@oxlint/plugins";
import { isLogicalDecision, unwrap } from "../../expression.ts";

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

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Limit nested logical groups and ternary decisions without penalizing flat chains.",
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
      tooDeep:
        "Decision depth is {{depth}}; maximum is {{max}}. Make the nested decisions explicit with meaningful predicates or branches.",
    },
  },
  create(context) {
    const option = context.options[0] as { max?: unknown } | undefined;
    const max = option?.max;
    if (typeof max !== "number" || !Number.isSafeInteger(max) || max < 1) {
      throw new Error("max-decision-depth requires a positive safe integer max option.");
    }
    const depths = new WeakMap<ESTree.Node, number>();
    const candidates = new Map<ESTree.Node, number>();
    const depthOf = (node: ESTree.Node) => depths.get(node) ?? 0;
    return {
      "*:exit"(node) {
        if (isBoundary(node)) {
          depths.set(node, 0);
          return;
        }
        let depth = 0;
        const fields = node as unknown as Record<string, ESTree.Node | ESTree.Node[] | null>;
        for (const key of context.sourceCode.visitorKeys[node.type] ?? []) {
          const child = fields[key];
          if (Array.isArray(child)) {
            for (const element of child) {
              if (element != null) depth = Math.max(depth, depthOf(element));
            }
          } else if (child != null) depth = Math.max(depth, depthOf(child));
        }
        if (isLogicalDecision(node)) {
          const branchDepth = (child: ESTree.Node) => {
            const expression = unwrap(child);
            const sameGroup =
              expression.type === "LogicalExpression" && expression.operator === node.operator;
            const childDepth = depthOf(child);
            return sameGroup ? childDepth - 1 : childDepth;
          };
          depth = 1 + Math.max(branchDepth(node.left), branchDepth(node.right));
          if (depth > max) candidates.set(node, depth);
        } else if (node.type === "ConditionalExpression") {
          const branchDepth = Math.max(depthOf(node.consequent), depthOf(node.alternate));
          depth = 1 + Math.max(depthOf(node.test), branchDepth);
          if (depth > max) candidates.set(node, depth);
        }
        depths.set(node, depth);
      },
      "Program:exit"() {
        for (const [node, depth] of candidates) {
          let ancestor = node.parent;
          while (ancestor !== null && !isBoundary(ancestor) && !candidates.has(ancestor)) {
            ancestor = ancestor.parent;
          }
          if (ancestor !== null && candidates.has(ancestor)) continue;
          context.report({ node, messageId: "tooDeep", data: { depth, max } });
        }
      },
    };
  },
});
