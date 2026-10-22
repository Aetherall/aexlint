import { defineRule, type Context, type ESTree, type LineColumn } from "@oxlint/plugins";
import {
  childNodes,
  isClass,
  isFunctionNode,
  isInvocation,
  isLoop,
  type Loop,
  unwrap,
} from "../../src/expression.ts";

const exits = ["ReturnStatement", "ThrowStatement", "ContinueStatement", "BreakStatement"];

/** A function passed directly into a call or construction: its body is part of the operation that receives it. */
function isArgumentCallback(node: ESTree.Node): boolean {
  let value: ESTree.Node = node;
  while (value.parent !== null && unwrap(value.parent) !== value.parent) value = value.parent;
  const parent = value.parent;
  if (parent === null || !isInvocation(parent)) return false;
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
    const measure = new HeldContextMeasure(context);
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

  constructor(context: Context) {
    const option = context.options[0] as { max?: unknown } | undefined;
    const max = option?.max;
    if (typeof max !== "number" || !Number.isSafeInteger(max) || max < 1) {
      throw new Error("max-held-context requires a positive safe integer max option.");
    }
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
    for (const child of childNodes(this.context.sourceCode.visitorKeys, node)) {
      this.walk(child, depth, region);
    }
  }

  private opened(node: ESTree.Node, depth: number, region: Region): void {
    this.walk(node, depth + 1, HeldContextMeasure.inside(region));
  }

  private walk(node: ESTree.Node, depth: number, region: Region): void {
    if (isFunctionNode(node)) {
      if (isArgumentCallback(node)) return this.opened(node.body!, depth, region);
      if (node.body) this.unit(node.body);
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
        return this.switchStatement(node, depth, region);
      case "TryStatement":
        return this.tryStatement(node, depth, region);
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
    if (node.alternate?.type === "IfStatement") return this.walk(node.alternate, depth, region);
    if (node.alternate) this.opened(node.alternate, depth, region);
  }

  private switchStatement(node: ESTree.SwitchStatement, depth: number, region: Region): void {
    this.decide(node, depth, region);
    this.walk(node.discriminant, depth, region);
    for (const branch of node.cases) {
      if (branch.test) this.walk(branch.test, depth, region);
      const inside = HeldContextMeasure.inside(region);
      for (const statement of branch.consequent) this.walk(statement, depth + 1, inside);
    }
  }

  private tryStatement(node: ESTree.TryStatement, depth: number, region: Region): void {
    this.walk(node.block, depth, region);
    if (node.handler) {
      this.decide(node.handler, depth, region);
      if (node.handler.param) this.walk(node.handler.param, depth, region);
      this.opened(node.handler.body, depth, region);
    }
    if (node.finalizer) this.walk(node.finalizer, depth, region);
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
    const data = { depth, open, max: this.max };
    this.context.report({ loc: this.head(node), messageId: "tooDeep", data });
    region.reported = true;
  }

  /** From the decision's keyword to the parenthesis closing its condition; a ternary or `do … while` reports its condition. */
  private head(node: ESTree.Node): { start: LineColumn; end: LineColumn } {
    const source = this.context.sourceCode;
    const at = (offset: number) => source.getLocFromIndex(offset);
    const conditionOnly = node.type === "ConditionalExpression" || node.type === "DoWhileStatement";
    if (conditionOnly) {
      return { start: at(node.test.range[0]), end: at(node.test.range[1]) };
    }
    if (node.type === "CatchClause") return this.catchHead(node);
    const fields = node as unknown as Record<string, ESTree.Node | null | undefined>;
    const last = ["test", "update", "right", "discriminant"]
      .map((key) => fields[key])
      .filter((part): part is ESTree.Node => Boolean(part))
      .reduce<number>((end, part) => Math.max(end, part.range[1]), node.range[0]);
    return { start: at(node.range[0]), end: at(source.text.indexOf(")", last) + 1) };
  }

  private catchHead(node: ESTree.CatchClause): { start: LineColumn; end: LineColumn } {
    const source = this.context.sourceCode;
    const end = node.param
      ? source.text.indexOf(")", node.param.range[1]) + 1
      : node.range[0] + "catch".length;
    return { start: source.getLocFromIndex(node.range[0]), end: source.getLocFromIndex(end) };
  }

  toString(): string {
    return `held-context measure (max ${this.max})`;
  }
}
