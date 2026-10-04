import { defineRule } from "@oxlint/plugins";

export default defineRule({
  meta: {
    type: "suggestion",
    docs: { description: "Disallow else if.", requiresTypeChecking: false },
    schema: [],
    messages: {
      elseIf:
        "This `else if` applies only when every earlier condition in its chain is false, so its meaning depends on the conditions above it.",
    },
  },
  create(context) {
    return {
      IfStatement(node) {
        const alternate = node.alternate;
        if (alternate?.type !== "IfStatement") return;
        const elseToken = context.sourceCode.getTokenBefore(alternate);
        const ifToken = context.sourceCode.getFirstToken(alternate);
        if (elseToken === null || ifToken === null) return;
        context.report({
          messageId: "elseIf",
          loc: { start: elseToken.loc.start, end: ifToken.loc.end },
        });
      },
    };
  },
});
