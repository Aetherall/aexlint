import { defineRule, type ESTree } from "@oxlint/plugins";
import {
  childNodes,
  closestAncestor,
  invocationStage,
  isFunctionNode,
} from "../../src/expression.ts";

function isBoundary(node: ESTree.Node): boolean {
  return isFunctionNode(node) || node.type === "ClassBody";
}

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description: "Limit buried input-producing invocations without penalizing fluent sequences.",
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
        "Input-production depth is {{depth}}; maximum is {{max}}. Make the prerequisites explicit with meaningful intermediate results.",
    },
  },
  create(context) {
    const option = context.options[0] as { max?: unknown } | undefined;
    const max = option?.max;
    if (typeof max !== "number" || !Number.isSafeInteger(max) || max < 1) {
      throw new Error("max-expression-depth requires a positive safe integer max option.");
    }
    const depths = new WeakMap<ESTree.Node, number>();
    const candidates = new Map<ESTree.Node, number>();
    const depthOf = (node: ESTree.Node) => depths.get(node) ?? 0;
    const greatestDepth = (nodes: ESTree.Node[]) => {
      let depth = 0;
      for (const node of nodes) depth = Math.max(depth, depthOf(node));
      return depth;
    };
    const childrenOf = (node: ESTree.Node) => childNodes(context.sourceCode.visitorKeys, node);
    return {
      "*:exit"(node) {
        if (isBoundary(node)) {
          depths.set(node, 0);
          return;
        }
        const stage = invocationStage(context.sourceCode.visitorKeys, node);
        if (!stage) {
          depths.set(node, greatestDepth(childrenOf(node)));
          return;
        }
        const receiverDepth = stage.receiver ? depthOf(stage.receiver) : 0;
        const depth = 1 + greatestDepth(stage.inputs);
        depths.set(node, Math.max(depth, receiverDepth));
        if (depth > max) candidates.set(node, depth);
      },
      "Program:exit"() {
        for (const [node, depth] of candidates) {
          const stop = (ancestor: ESTree.Node) => isBoundary(ancestor) || candidates.has(ancestor);
          const outer = closestAncestor(node, stop);
          if (outer !== null && candidates.has(outer)) continue;
          context.report({ node, messageId: "tooDeep", data: { depth, max } });
        }
      },
    };
  },
});
