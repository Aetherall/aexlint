import { defineRule, type ESTree } from "@oxlint/plugins";
import {
  branchInitializer,
  childNodes,
  closestAncestor,
  continuesGroup,
  isComputationBoundary,
  isLogicalDecision,
  singleReadConstants,
} from "../../src/expression.ts";

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
    const constants = singleReadConstants(context.sourceCode.scopeManager);
    const deferred = new Map<ESTree.Node, ESTree.Node>();
    return {
      "*:exit"(node) {
        if (isComputationBoundary(node)) {
          depths.set(node, 0);
          return;
        }
        const initializer = branchInitializer(constants, node);
        if (initializer) {
          deferred.set(node, initializer);
          depths.set(node, depthOf(initializer));
          return;
        }
        const children = childNodes(context.sourceCode.visitorKeys, node);
        let depth = children.reduce((deepest, child) => Math.max(deepest, depthOf(child)), 0);
        if (isLogicalDecision(node)) {
          const branchDepth = (child: ESTree.Node) => {
            const childDepth = depthOf(child);
            return continuesGroup(deferred.get(child) ?? child, node) ? childDepth - 1 : childDepth;
          };
          depth = 1 + Math.max(branchDepth(node.left), branchDepth(node.right));
          if (depth > max) candidates.set(node, depth);
        }
        if (node.type === "ConditionalExpression") {
          depth += 1;
          if (depth > max) candidates.set(node, depth);
        }
        depths.set(node, depth);
      },
      "Program:exit"() {
        for (const [node, depth] of candidates) {
          const stop = (ancestor: ESTree.Node) =>
            isComputationBoundary(ancestor) || candidates.has(ancestor);
          const outer = closestAncestor(node, stop);
          if (outer !== null && candidates.has(outer)) continue;
          context.report({ node, messageId: "tooDeep", data: { depth, max } });
        }
      },
    };
  },
});
