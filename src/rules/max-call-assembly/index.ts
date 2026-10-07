import { defineRule, type ESTree, type Reference } from "@oxlint/plugins";
import { isFunctionNode, unwrap } from "../../expression.ts";

type Invocation = ESTree.CallExpression | ESTree.NewExpression;
type Piece = { text: string; start: number; end: number };
type Assembly = { collapsed: [number, number][]; names: Set<string> };

function isInvocation(node: ESTree.Node): node is Invocation {
  return node.type === "CallExpression" || node.type === "NewExpression";
}

function isInsideFunction(node: ESTree.Node): boolean {
  for (let ancestor = node.parent; ancestor; ancestor = ancestor.parent) {
    if (isFunctionNode(ancestor) || ancestor.type === "ClassBody") return true;
  }
  return false;
}

function isHook(node: Invocation): boolean {
  const callee = unwrap(node.callee);
  const name = callee.type === "MemberExpression" && !callee.computed ? callee.property : callee;
  return name.type === "Identifier" && /^use(?:[A-Z0-9]|$)/.test(name.name);
}

function withoutLiteralText(token: ESTree.Token): string {
  const text = token.value;
  if (token.type === "JSXText") return "";
  if (token.type === "String") return text.charAt(0).repeat(2);
  if (token.type !== "Template") return text;
  return text.charAt(0) + (text.endsWith("${") ? "${" : "`");
}

function widthOf(list: Piece[]): number {
  let width = 0;
  list.forEach((piece, index) => {
    const previous = list[index - 1];
    width += piece.text.length;
    if (!previous || previous.end === piece.start) return;
    if (/^[([]$/.test(previous.text) || /^[)\]]$/.test(piece.text)) return;
    width += 1;
  });
  return width;
}

const typeKeys = new Set([
  "typeAnnotation",
  "typeArguments",
  "typeParameters",
  "returnType",
  "superTypeArguments",
  "implements",
]);

function isLimit(value: unknown): value is number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
}

const limit = { type: "integer", minimum: 0, maximum: Number.MAX_SAFE_INTEGER };

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Limit calls whose arguments are both too wide for one line and draw on many names from the surrounding code.",
      requiresTypeChecking: false,
    },
    schema: {
      type: "array",
      minItems: 1,
      maxItems: 1,
      items: [
        {
          type: "object",
          properties: { maxWidth: limit, maxReferences: limit },
          required: ["maxWidth", "maxReferences"],
          additionalProperties: false,
        },
      ],
    },
    messages: {
      tooAssembled:
        "These arguments are {{width}} characters wide on one line and draw on {{count}} outside names ({{names}}); limits are {{maxWidth}} and {{maxReferences}}. So much is assembled at this call that it cannot be read at a glance.",
    },
  },
  create(context) {
    const option = context.options[0] as { maxWidth?: unknown; maxReferences?: unknown };
    const { maxWidth, maxReferences } = option;
    if (!isLimit(maxWidth) || !isLimit(maxReferences)) {
      throw new Error(
        "max-call-assembly requires non-negative safe integer maxWidth and maxReferences.",
      );
    }
    const { sourceCode } = context;
    const covered = new WeakSet<ESTree.Node>();
    const references = new Map<ESTree.Node, Reference>();
    for (const scope of sourceCode.scopeManager.scopes) {
      for (const reference of scope.references) {
        references.set(reference.identifier as ESTree.Node, reference);
      }
    }

    const assemblyOf = (node: Invocation, start: number, end: number): Assembly => {
      const collapsed: [number, number][] = [];
      const names = new Set<string>();
      const isOutside = (reference: Reference) =>
        !reference.resolved ||
        !reference.resolved.defs.every((def) => def.name.start >= start && def.name.end <= end);
      const visit = (current: ESTree.Node) => {
        if (current.type === "ClassBody") {
          collapsed.push([current.start, current.end]);
          return;
        }
        const reference = references.get(current);
        if (reference && isOutside(reference)) names.add(reference.identifier.name);
        if (isFunctionNode(current) && current.body) {
          collapsed.push([current.body.start, current.body.end]);
        }
        const fields = current as unknown as Record<string, ESTree.Node | ESTree.Node[] | null>;
        for (const key of sourceCode.visitorKeys[current.type] ?? []) {
          if (isFunctionNode(current) && key === "body") continue;
          if (typeKeys.has(key)) continue;
          const child = fields[key];
          if (Array.isArray(child)) {
            for (const element of child) if (element != null) visit(element);
          } else if (child != null) visit(child);
        }
      };
      for (const argument of node.arguments) visit(argument);
      return { collapsed, names };
    };

    const pieces = (
      node: Invocation,
      open: ESTree.Token,
      close: ESTree.Token,
      assembly: Assembly,
    ) => {
      const result: Piece[] = [];
      for (const token of sourceCode.getTokens(node)) {
        if (token.start < open.start || token.end > close.end) continue;
        const range = assembly.collapsed.find(
          ([start, end]) => token.start >= start && token.end <= end,
        );
        if (!range) {
          const text = withoutLiteralText(token);
          result.push({ text, start: token.start, end: token.end });
        } else if (result.at(-1)?.start !== range[0]) {
          result.push({ text: "{}", start: range[0], end: range[1] });
        }
      }
      return result.filter(
        (piece, index) => piece.text !== "," || !/^[)\]}]$/.test(result[index + 1]?.text ?? ""),
      );
    };

    const coverEnclosing = (node: ESTree.Node) => {
      let child = node;
      let parent = node.parent;
      while (parent != null) {
        if (parent.type === "ClassBody") return;
        if (isFunctionNode(parent) && parent.body === child) return;
        if (isInvocation(parent) && parent.arguments.some((argument) => argument === child)) {
          covered.add(parent);
          return;
        }
        child = parent;
        parent = parent.parent;
      }
    };

    const check = (node: Invocation) => {
      if (node.arguments.length === 0 || !isInsideFunction(node) || isHook(node)) return;
      const prefix = node.typeArguments ?? node.callee;
      const open = sourceCode.getTokenAfter(prefix, { filter: (token) => token.value === "(" });
      const close = sourceCode.getLastToken(node);
      if (open === null || close === null) return;
      const assembly = assemblyOf(node, open.start, close.end);
      if (assembly.names.size <= maxReferences) return;
      const width = widthOf(pieces(node, open, close, assembly));
      if (width <= maxWidth) return;
      coverEnclosing(node);
      if (covered.has(node)) return;
      context.report({
        loc: { start: open.loc.start, end: close.loc.end },
        messageId: "tooAssembled",
        data: {
          width,
          count: assembly.names.size,
          names: [...assembly.names].join(", "),
          maxWidth,
          maxReferences,
        },
      });
    };

    return {
      "CallExpression:exit": check,
      "NewExpression:exit": check,
    };
  },
});
