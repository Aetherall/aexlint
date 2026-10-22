import { defineRule, type ESTree } from "@oxlint/plugins";
import { childNodes, isFunctionNode, isLoop, unwrap } from "../../src/expression.ts";

type Node = ESTree.Node;
type End = "first" | "last";
type Keys = Readonly<Record<string, readonly string[]>>;

/** One pass of a loop or of a callback an array method calls per element, with the variable holding its index. */
interface Pass {
  index: string;
  body: Node;
  owner: Node;
}

const passCallbacks = ["forEach", "map", "flatMap"];
const cutters = ["slice", "substring", "substr"];
const flipped: Record<string, string> = { ">": "<", "<": ">" };
const skips: Record<End, string[]> = { first: [">", "!==", "!="], last: ["<", "!==", "!="] };

function isIdentifier(node: Node | null | undefined): node is ESTree.IdentifierReference {
  return node?.type === "Identifier";
}

function isNumber(node: Node | null | undefined, value: number): boolean {
  return node?.type === "Literal" && node.value === value;
}

function isMethodCall(
  node: Node,
  names: string[],
): node is ESTree.CallExpression & { callee: ESTree.StaticMemberExpression } {
  if (node.type !== "CallExpression" || node.callee.type !== "MemberExpression") return false;
  return !node.callee.computed && names.includes(node.callee.property.name);
}

function counterPass(loop: ESTree.ForStatement): Pass | null {
  const init = loop.init;
  const id = init?.type === "VariableDeclaration" ? init.declarations[0]?.id : null;
  return id?.type === "Identifier" ? { index: id.name, body: loop.body, owner: loop } : null;
}

function entriesPass(loop: ESTree.ForOfStatement): Pass | null {
  const left = loop.left.type === "VariableDeclaration" ? loop.left.declarations[0]?.id : null;
  if (left?.type !== "ArrayPattern" || !isMethodCall(loop.right, ["entries"])) return null;
  const index = left.elements[0];
  return index?.type === "Identifier" ? { index: index.name, body: loop.body, owner: loop } : null;
}

function callbackPass(call: ESTree.CallExpression): Pass | null {
  if (!isMethodCall(call, passCallbacks)) return null;
  const callback = call.arguments[0];
  if (!callback || !isFunctionNode(callback) || !callback.body) return null;
  const index = callback.params[1];
  return index?.type === "Identifier"
    ? { index: index.name, body: callback.body, owner: callback }
    : null;
}

function isSubtraction(node: Node): node is ESTree.BinaryExpression & { operator: "-" } {
  return node.type === "BinaryExpression" && node.operator === "-";
}

function isLengthRead(node: Node): node is ESTree.StaticMemberExpression {
  return node.type === "MemberExpression" && !node.computed && node.property.name === "length";
}

function isLastIndex(node: Node): boolean {
  return isSubtraction(node) && isLengthRead(node.left) && isNumber(node.right, 1);
}

function isComparison(node: Node): node is ESTree.BinaryExpression {
  return node.type === "BinaryExpression";
}

/** The end of the loop a test leaves out, when it holds on every pass but the first or every pass but the last. */
function skippedEnd(test: Node, index: string): End | null {
  test = unwrap(test);
  if (!isComparison(test)) return null;
  const leftIndex = isIdentifier(test.left) && test.left.name === index;
  const rightIndex = isIdentifier(test.right) && test.right.name === index;
  if (leftIndex === rightIndex) return null;
  const other = leftIndex ? test.right : test.left;
  const operator = leftIndex ? test.operator : (flipped[test.operator] ?? test.operator);
  if (isNumber(other, 0) && skips.first.includes(operator)) return "first";
  return isLastIndex(other) && skips.last.includes(operator) ? "last" : null;
}

function isAppend(node: Node | null): node is ESTree.AssignmentExpression & { operator: "+=" } {
  return node?.type === "AssignmentExpression" && node.operator === "+=";
}

/** Where an expression emits: the variable it appends to, or the object whose method it calls. */
function sinkOfEmission(expression: Node, text: (node: Node) => string): string | null {
  if (isAppend(expression)) return isIdentifier(expression.left) ? expression.left.name : null;
  if (expression.type !== "CallExpression") return null;
  const callee = expression.callee;
  return callee.type === "MemberExpression" ? text(callee.object) : null;
}

/** Where a statement that only emits does so. */
function sinkOf(statement: Node, text: (node: Node) => string): string | null {
  if (statement.type === "BlockStatement" && statement.body.length === 1) {
    return sinkOf(statement.body[0]!, text);
  }
  return statement.type === "ExpressionStatement"
    ? sinkOfEmission(statement.expression, text)
    : null;
}

function isPropertyName(node: Node, parent: Node): boolean {
  if (parent.type === "Property") return parent.key === node;
  return parent.type === "MemberExpression" && parent.property === node && !parent.computed;
}

function isReference(node: Node): node is ESTree.IdentifierReference {
  const parent = node.parent;
  if (node.type !== "Identifier" || parent === null) return false;
  return !isPropertyName(node, parent);
}

function walk(keys: Keys, node: Node, visit: (node: Node) => boolean): void {
  if (!visit(node)) return;
  for (const child of childNodes(keys, node)) walk(keys, child, visit);
}

/** The names a pass declares: its index, and the variables and parameters inside it. */
function declaredIn(keys: Keys, pass: Pass): Set<string> {
  const names = new Set([pass.index]);
  const collect = (pattern: Node) =>
    walk(keys, pattern, (node) => {
      if (isReference(node)) names.add(node.name);
      return true;
    });
  walk(keys, pass.owner, (node) => {
    if (node.type === "VariableDeclarator") collect(node.id);
    if (isFunctionNode(node)) node.params.forEach(collect);
    return true;
  });
  return names;
}

/** Whether a statement only emits what no pass changes: it reads nothing the pass declares. */
function constant(keys: Keys, statement: Node, pass: Pass): boolean {
  const declared = declaredIn(keys, pass);
  let found = false;
  walk(keys, statement, (node) => {
    found ||= isReference(node) && declared.has(node.name);
    return !found;
  });
  return !found;
}

/** Whether a statement can end its pass early: a `continue` of the loop, or a `return` from the callback. */
function leaves(keys: Keys, statement: Node, pass: Pass): boolean {
  const ending = isFunctionNode(pass.owner) ? "ReturnStatement" : "ContinueStatement";
  let found = false;
  const nested = (node: Node) => {
    const inner = isLoop(node) && node !== statement;
    return inner || isFunctionNode(node);
  };
  walk(keys, statement, (node) => {
    found ||= node.type === ending;
    return !found && !nested(node);
  });
  return found;
}

/** Whether a statement emits into a sink: appends to it, or calls a method on it, outside the functions it declares. */
function emitsTo(keys: Keys, statement: Node, sink: string, text: (node: Node) => string): boolean {
  let found = false;
  walk(keys, statement, (node) => {
    found ||= sinkOfEmission(node, text) === sink;
    return !found && !isFunctionNode(node);
  });
  return found;
}

function statementsOf(body: Node): Node[] {
  return body.type === "BlockStatement" ? body.body : [body];
}

function isNegation(node: Node): node is ESTree.UnaryExpression & { operator: "-" } {
  return node.type === "UnaryExpression" && node.operator === "-";
}

function positive(node: Node | null): number {
  const value = node?.type === "Literal" ? node.value : null;
  return typeof value === "number" && value > 0 ? value : 0;
}

/** The count an end position takes off: `N` in `-N` or in `name.length - N`. */
function fromEnd(node: Node, name: string): Node | null {
  if (isNegation(node)) return node.argument;
  if (!isSubtraction(node) || !isLengthRead(node.left)) return null;
  const sized = node.left.object;
  return isIdentifier(sized) && sized.name === name ? node.right : null;
}

function cutLength(call: ESTree.CallExpression, name: string): { end: End; length: number } | null {
  const [from, to, ...rest] = call.arguments;
  if (rest.length > 0 || !from) return null;
  if (!to) {
    const length = positive(from);
    return length > 0 ? { end: "first", length } : null;
  }
  if (!isNumber(from, 0)) return null;
  const length = positive(fromEnd(to, name));
  return length > 0 ? { end: "last", length } : null;
}

function literal(node: Node): string | null {
  node = unwrap(node);
  if (node.type === "Literal" && typeof node.value === "string") return node.value;
  if (node.type !== "TemplateLiteral" || node.expressions.length > 0) return null;
  return node.quasis[0]?.value.cooked ?? null;
}

/** The literal text every value of an expression starts or ends with. */
function edge(node: Node, end: End): string {
  node = unwrap(node);
  const whole = literal(node);
  if (whole !== null) return whole;
  if (node.type === "TemplateLiteral") {
    const quasi = end === "last" ? node.quasis.at(-1) : node.quasis[0];
    return quasi?.value.cooked ?? "";
  }
  if (!isConcatenation(node)) return "";
  const [outer, inner] = end === "last" ? [node.right, node.left] : [node.left, node.right];
  const text = edge(outer, end);
  if (literal(outer) === null) return text;
  return end === "last" ? edge(inner, end) + text : text + edge(inner, end);
}

function isConcatenation(node: Node): node is ESTree.BinaryExpression & { operator: "+" } {
  return node.type === "BinaryExpression" && node.operator === "+";
}

function loopOf(node: Node): Node | null {
  for (let parent = node.parent; parent !== null; parent = parent.parent) {
    if (isLoop(parent)) return parent;
    if (isFunctionNode(parent)) return null;
  }
  return null;
}

function appendOf(reference: { identifier: Node }): ESTree.AssignmentExpression | null {
  const parent = reference.identifier.parent;
  return isAppend(parent) && parent.left === reference.identifier ? parent : null;
}

function lineOf(node: Node): string {
  return String(node.loc.start.line);
}

const where: Record<End, string> = {
  first: "at the start, or twice in a row",
  last: "at the end, or twice in a row",
};

export default defineRule({
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Disallow placing a separator by a loop pass's position, or cutting it off after the loop, instead of between the pieces emitted.",
      requiresTypeChecking: false,
    },
    schema: [],
    messages: {
      positionalSeparator:
        "`{{test}}` places this separator by the pass's position, not by whether a piece {{relation}}: {{consequence}} Emit it between the pieces actually emitted, by joining them or by tracking whether one was emitted.",
      trimmedSeparator:
        "`{{name}}` gets a separator with every piece in the loop on line {{line}}, and this `{{method}}` cuts the {{end}} one off by its length, not by whether a piece was emitted: {{consequence}} Join the pieces with the separator instead.",
    },
  },
  create(context) {
    const keys = context.sourceCode.visitorKeys as Keys;
    const text = (node: Node) => context.sourceCode.getText(node);

    const checkPass = (pass: Pass | null) => {
      if (!pass) return;
      const statements = statementsOf(pass.body);
      for (const [at, guard] of statements.entries()) {
        if (guard.type !== "IfStatement" || guard.alternate) continue;
        const end = skippedEnd(guard.test, pass.index);
        const sink = end && sinkOf(guard.consequent, text);
        if (!end || !sink || !constant(keys, guard.consequent, pass)) continue;
        const after = (i: number) => (end === "first") === i > at;
        const pieces = statements.filter((s, i) => after(i) && emitsTo(keys, s, sink, text));
        const piece = end === "first" ? pieces[0] : pieces.at(-1);
        if (!piece) continue;
        const until = Math.max(at, statements.indexOf(piece));
        const leaving = statements
          .slice(0, until)
          .find((s) => s !== guard && leaves(keys, s, pass));
        const consequence = leaving
          ? `a pass can end on line ${lineOf(leaving)} without its piece, which leaves the separator ${where[end]}.`
          : "it is only right while every pass emits a piece.";
        const relation = end === "first" ? "came before it" : "comes after it";
        const data = { test: text(guard.test), relation, consequence };
        context.report({ node: guard, messageId: "positionalSeparator", data });
      }
    };

    const checkCut = (call: ESTree.CallExpression) => {
      if (!isMethodCall(call, cutters) || !isIdentifier(call.callee.object)) return;
      const target = call.callee.object;
      const cut = cutLength(call, target.name);
      if (!cut) return;
      const scope = context.sourceCode.getScope(call);
      const variable = scope.references.find((r) => r.identifier === target)?.resolved;
      if (!variable) return;
      const start = call.range[0];
      const writes = variable.references.filter(
        (r) => r.isWrite() && r.identifier.range[0] < start,
      );
      const last = writes.at(-1);
      const append = last ? appendOf(last) : null;
      const loop = append ? loopOf(append) : null;
      if (!loop || loop.range[1] > start) return;
      const pieces = writes.map(appendOf).filter((a) => a !== null && loopOf(a) === loop);
      const piece = cut.end === "last" ? pieces.at(-1) : pieces[0];
      if (!piece || edge(piece.right, cut.end).length < cut.length) return;
      const declarator = variable.defs[0]?.node;
      const init = declarator?.type === "VariableDeclarator" ? declarator.init : null;
      const empty = init && literal(init) === "";
      const consequence = empty
        ? "it is only right while the cut keeps the separator's length."
        : `when no pass runs, it cuts ${cut.length} characters written before the loop.`;
      const method = call.callee.property.name;
      const data = { name: target.name, line: lineOf(loop), method, end: cut.end, consequence };
      context.report({ node: call, messageId: "trimmedSeparator", data });
    };

    return {
      ForStatement: (node) => checkPass(counterPass(node)),
      ForOfStatement: (node) => checkPass(entriesPass(node)),
      CallExpression: (node) => {
        checkPass(callbackPass(node));
        checkCut(node);
      },
    };
  },
});
