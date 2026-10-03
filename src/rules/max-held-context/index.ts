import { defineRule, type Context, type ESTree, type LineColumn } from "@oxlint/plugins";
import { isFunctionNode, unwrap } from "../../expression.ts";

type Loop =
  | ESTree.ForStatement
  | ESTree.ForInStatement
  | ESTree.ForOfStatement
  | ESTree.WhileStatement
  | ESTree.DoWhileStatement;

const exits = ["ReturnStatement", "ThrowStatement", "ContinueStatement", "BreakStatement"];

function isClass(node: ESTree.Node): boolean {
  return node.type === "ClassDeclaration" || node.type === "ClassExpression";
}

function isLoop(node: ESTree.Node): node is Loop {
  return [
    "ForStatement",
    "ForInStatement",
    "ForOfStatement",
    "WhileStatement",
    "DoWhileStatement",
  ].includes(node.type);
}

/** A function passed directly into a call or construction: its body is part of the operation that receives it. */
function isArgumentCallback(node: ESTree.Node): boolean {
  let value: ESTree.Node = node;
  while (value.parent !== null && unwrap(value.parent) !== value.parent) value = value.parent;
  const parent = value.parent;
  if (parent?.type !== "CallExpression" && parent?.type !== "NewExpression") return false;
  return (parent.arguments as ESTree.Node[]).includes(value);
}

function isExit(statement: ESTree.Node | null | undefined): boolean {
  if (!statement) return false;
  if (exits.includes(statement.type)) return true;
  return (
    statement.type === "BlockStatement" &&
    statement.body.length === 1 &&
    exits.includes(statement.body[0]!.type)
  );
}

function isGuard(node: ESTree.IfStatement): boolean {
  return node.alternate === null && isExit(node.consequent);
}

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Limit the enclosing branches, loops, cases, handlers and callbacks held in mind where a decision is made.",
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
        "This decision is made inside {{open}} open contexts (held-context depth {{depth}}); maximum is {{max}}. A reader must keep the enclosing branches, loops, cases, handlers or callbacks in mind to follow it.",
    },
  },
  create(context) {
    const option = context.options[0] as { max?: unknown } | undefined;
    const max = option?.max;
    if (typeof max !== "number" || !Number.isSafeInteger(max) || max < 1) {
      throw new Error("max-held-context requires a positive safe integer max option.");
    }
    const measure = new HeldContextMeasure(context, max);
    return {
      Program(node) {
        measure.unit(node);
      },
    };
  },
});

/** One open context; once a decision in it is reported, the rest of it is not reported again. */
interface Region {
  reported: boolean;
}

/** Walks each measured unit, tracking the contexts open at every decision. */
class HeldContextMeasure {
  private readonly context: Context;
  private readonly max: number;

  constructor(context: Context, max: number) {
    this.context = context;
    this.max = max;
  }

  unit(node: ESTree.Node): void {
    this.walk(node, 0, { reported: false });
  }

  private static inside(region: Region): Region {
    return { reported: region.reported };
  }

  private children(node: ESTree.Node, depth: number, region: Region): void {
    const fields = node as unknown as Record<string, ESTree.Node | (ESTree.Node | null)[] | null>;
    for (const key of this.context.sourceCode.visitorKeys[node.type] ?? []) {
      const child = fields[key];
      if (Array.isArray(child)) {
        for (const element of child) if (element) this.walk(element, depth, region);
      } else if (child && typeof child.type === "string") this.walk(child, depth, region);
    }
  }

  private opened(node: ESTree.Node, depth: number, region: Region): void {
    this.walk(node, depth + 1, HeldContextMeasure.inside(region));
  }

  private walk(node: ESTree.Node, depth: number, region: Region): void {
    if (isFunctionNode(node)) {
      if (isArgumentCallback(node)) this.opened(node.body!, depth, region);
      else if (node.body) this.unit(node.body);
      return;
    }
    if (isClass(node)) {
      this.children(node, 0, { reported: false });
      return;
    }
    switch (node.type) {
      case "IfStatement":
        return this.ifStatement(node, depth, region);
      case "SwitchStatement":
        this.decide(node, depth, region);
        this.walk(node.discriminant, depth, region);
        for (const branch of node.cases) {
          if (branch.test) this.walk(branch.test, depth, region);
          const inside = HeldContextMeasure.inside(region);
          for (const statement of branch.consequent) this.walk(statement, depth + 1, inside);
        }
        return;
      case "TryStatement":
        this.walk(node.block, depth, region);
        if (node.handler) {
          this.decide(node.handler, depth, region);
          if (node.handler.param) this.walk(node.handler.param, depth, region);
          this.opened(node.handler.body, depth, region);
        }
        if (node.finalizer) this.walk(node.finalizer, depth, region);
        return;
      case "ConditionalExpression":
        this.decide(node, depth, region);
        this.walk(node.test, depth, region);
        this.opened(node.consequent, depth, region);
        this.opened(node.alternate, depth, region);
        return;
    }
    if (isLoop(node)) return this.loop(node, depth, region);
    this.children(node, depth, region);
  }

  private ifStatement(node: ESTree.IfStatement, depth: number, region: Region): void {
    this.decide(node, depth, region);
    this.walk(node.test, depth, region);
    if (isGuard(node)) {
      this.walk(node.consequent, depth, region);
      return;
    }
    this.opened(node.consequent, depth, region);
    if (node.alternate?.type === "IfStatement") this.walk(node.alternate, depth, region);
    else if (node.alternate) this.opened(node.alternate, depth, region);
  }

  private loop(node: Loop, depth: number, region: Region): void {
    this.decide(node, depth, region);
    const fields = node as unknown as Record<string, ESTree.Node | null | undefined>;
    for (const key of ["init", "left", "right", "test", "update"]) {
      if (fields[key]) this.walk(fields[key], depth, region);
    }
    this.opened(node.body, depth, region);
  }

  /** Reports the first decision over the limit in its context; the rest of that context is then not reported. */
  private decide(node: ESTree.Node, open: number, region: Region): void {
    const depth = open + 1;
    if (region.reported || depth <= this.max) return;
    this.context.report({
      loc: this.head(node),
      messageId: "tooDeep",
      data: { depth, open, max: this.max },
    });
    region.reported = true;
  }

  /** From the decision's keyword to the parenthesis closing its condition; a ternary or `do … while` reports its condition. */
  private head(node: ESTree.Node): { start: LineColumn; end: LineColumn } {
    const source = this.context.sourceCode;
    const at = (offset: number) => source.getLocFromIndex(offset);
    if (node.type === "ConditionalExpression" || node.type === "DoWhileStatement") {
      return { start: at(node.test.range[0]), end: at(node.test.range[1]) };
    }
    if (node.type === "CatchClause") {
      const end = node.param
        ? source.text.indexOf(")", node.param.range[1]) + 1
        : node.range[0] + "catch".length;
      return { start: at(node.range[0]), end: at(end) };
    }
    const fields = node as unknown as Record<string, ESTree.Node | null | undefined>;
    const last = ["test", "update", "right", "discriminant"]
      .map((key) => fields[key])
      .filter((part): part is ESTree.Node => Boolean(part))
      .reduce<number>((end, part) => Math.max(end, part.range[1]), node.range[0]);
    return { start: at(node.range[0]), end: at(source.text.indexOf(")", last) + 1) };
  }

  toString(): string {
    return `held-context measure (max ${this.max})`;
  }
}
