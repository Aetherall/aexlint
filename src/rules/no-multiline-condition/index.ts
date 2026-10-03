import { defineRule, type ESTree } from "@oxlint/plugins";

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description: "Keep the condition of if, while, do...while and for statements on one line.",
      requiresTypeChecking: false,
    },
    schema: [],
    messages: {
      multiline:
        "This {{statement}} condition spans {{lines}} lines, so the decision cannot be read at a glance before the body.",
    },
  },
  create(context) {
    const check = (test: ESTree.Expression | null, statement: string) => {
      if (test === null) return;
      const lines = test.loc.end.line - test.loc.start.line + 1;
      if (lines > 1)
        context.report({ node: test, messageId: "multiline", data: { statement, lines } });
    };
    return {
      IfStatement: (node) => check(node.test, "if"),
      WhileStatement: (node) => check(node.test, "while"),
      DoWhileStatement: (node) => check(node.test, "do...while"),
      ForStatement: (node) => check(node.test, "for"),
    };
  },
});
