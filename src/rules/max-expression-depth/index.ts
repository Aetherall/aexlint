import { defineRule, type ESTree } from "@oxlint/plugins";
import { unwrap } from "../../expression.ts";

function isBoundary(node: ESTree.Node): boolean {
  return [
    "FunctionDeclaration",
    "FunctionExpression",
    "ArrowFunctionExpression",
    "ClassBody",
  ].includes(node.type);
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
    return {
      "*:exit"(node) {
        if (isBoundary(node)) {
          depths.set(node, 0);
          return;
        }
        let inputs: ESTree.Node[];
        let callee: ESTree.Node | undefined;
        let receiverDepth = 0;
        switch (node.type) {
          case "CallExpression":
            callee = unwrap(node.callee);
            inputs = [...node.arguments];
            break;
          case "TaggedTemplateExpression":
            callee = unwrap(node.tag);
            inputs = [node.quasi];
            break;
          case "NewExpression":
            inputs = [node.callee, ...node.arguments];
            break;
          case "ImportExpression":
            inputs = childrenOf(node);
            break;
          default:
            depths.set(node, greatestDepth(childrenOf(node)));
            return;
        }
        if (callee?.type === "MemberExpression") {
          receiverDepth = depthOf(callee.object);
          if (callee.computed) inputs.push(callee.property);
        } else if (callee !== undefined) inputs.push(callee);
        const depth = 1 + greatestDepth(inputs);
        depths.set(node, Math.max(depth, receiverDepth));
        if (depth > max) candidates.set(node, depth);
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
