import type { ESTree, ScopeManager } from "@oxlint/plugins";

const wrapperTypes = [
  "ParenthesizedExpression",
  "ChainExpression",
  "TSAsExpression",
  "TSTypeAssertion",
  "TSSatisfiesExpression",
  "TSNonNullExpression",
  "TSInstantiationExpression",
] as const;

type ExpressionWrapper = Extract<ESTree.Node, { type: (typeof wrapperTypes)[number] }>;

function isExpressionWrapper(node: ESTree.Node): node is ExpressionWrapper {
  return wrapperTypes.some((type) => node.type === type);
}

export function isFunctionNode(
  node: ESTree.Node,
): node is ESTree.Function | ESTree.ArrowFunctionExpression {
  return ["FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression"].includes(
    node.type,
  );
}

export type Invocation = ESTree.CallExpression | ESTree.NewExpression;

export function isInvocation(node: ESTree.Node): node is Invocation {
  return node.type === "CallExpression" || node.type === "NewExpression";
}

export function isClass(node: ESTree.Node): node is ESTree.Class {
  return node.type === "ClassDeclaration" || node.type === "ClassExpression";
}

export function isComputationBoundary(node: ESTree.Node): boolean {
  return isFunctionNode(node) || isClass(node) || node.type === "ObjectExpression";
}

export interface InvocationStage {
  receiver: ESTree.Node | null;
  inputs: ESTree.Node[];
}

function fluentStage(invoked: ESTree.Node, inputs: ESTree.Node[]): InvocationStage {
  const callee = unwrap(invoked);
  if (callee.type !== "MemberExpression") return { receiver: null, inputs: [callee, ...inputs] };
  return { receiver: callee.object, inputs: [...inputs, ...computedProperty(callee)] };
}

export function invocationStage(
  visitorKeys: Readonly<Record<string, readonly string[]>>,
  node: ESTree.Node,
): InvocationStage | undefined {
  switch (node.type) {
    case "CallExpression":
      return fluentStage(node.callee, [...node.arguments]);
    case "TaggedTemplateExpression":
      return fluentStage(node.tag, [node.quasi]);
    case "NewExpression":
      return { receiver: null, inputs: [node.callee, ...node.arguments] };
    case "ImportExpression":
      return { receiver: null, inputs: childNodes(visitorKeys, node) };
    default:
      return undefined;
  }
}

export function continuesGroup(child: ESTree.Node, group: ESTree.LogicalExpression): boolean {
  const expression = unwrap(child);
  return expression.type === "LogicalExpression" && expression.operator === group.operator;
}

export function closestAncestor(
  node: ESTree.Node,
  stop: (ancestor: ESTree.Node) => boolean,
): ESTree.Node | null {
  let ancestor = node.parent;
  while (ancestor !== null && !stop(ancestor)) ancestor = ancestor.parent;
  return ancestor;
}

export function isLogicalDecision(
  node: ESTree.Node,
): node is ESTree.LogicalExpression & { operator: "&&" | "||" } {
  return node.type === "LogicalExpression" && node.operator !== "??";
}

export function isNegation(node: ESTree.Node): node is ESTree.UnaryExpression & { operator: "!" } {
  return node.type === "UnaryExpression" && node.operator === "!";
}

export function isSignedExpression(
  node: ESTree.Node,
): node is ESTree.UnaryExpression & { operator: "+" | "-" } {
  return node.type === "UnaryExpression" && ["+", "-"].includes(node.operator);
}

export function computedProperty(node: ESTree.MemberExpression): ESTree.Node[] {
  return node.computed ? [node.property] : [];
}

export function unwrap(node: ESTree.Node): ESTree.Node {
  while (isExpressionWrapper(node)) node = node.expression;
  return node;
}

export function childNodes(
  visitorKeys: Readonly<Record<string, readonly string[]>>,
  node: ESTree.Node,
  keep: (key: string) => boolean = () => true,
): ESTree.Node[] {
  const fields = node as unknown as Record<string, ESTree.Node | (ESTree.Node | null)[] | null>;
  const children: ESTree.Node[] = [];
  for (const key of (visitorKeys[node.type] ?? []).filter(keep)) {
    const value = fields[key];
    for (const child of Array.isArray(value) ? value : [value]) if (child) children.push(child);
  }
  return children;
}

function isStatementList(node: ESTree.Node): boolean {
  return node.type === "BlockStatement" || node.type === "Program";
}

function isLoopBody(node: ESTree.Node, parent: ESTree.Node): boolean {
  const loops = ["ForStatement", "ForInStatement", "ForOfStatement", "WhileStatement"];
  const loop = [...loops, "DoWhileStatement"].includes(parent.type);
  return loop && (parent as ESTree.WhileStatement).body === node;
}

function statementOf(node: ESTree.Node): ESTree.Node | null {
  let child = node;
  for (let parent = node.parent; parent !== null; parent = parent.parent) {
    if (isStatementList(parent) || isLoopBody(child, parent)) return child;
    child = parent;
  }
  return null;
}

export function singleReadConstants(scopeManager: ScopeManager): Map<ESTree.Node, ESTree.Node> {
  const initializers = new Map<ESTree.Node, ESTree.Node>();
  for (const scope of scopeManager.scopes) {
    for (const variable of scope.variables) {
      const [def, ...redeclared] = variable.defs;
      const declarator = def?.node;
      const declaration = def?.parent;
      if (redeclared.length > 0 || declarator?.type !== "VariableDeclarator") continue;
      if (declarator.id.type !== "Identifier" || !declarator.init) continue;
      const constDeclaration =
        declaration?.type === "VariableDeclaration" && declaration.kind === "const";
      if (!constDeclaration) continue;
      const [read, ...rereads] = variable.references.filter((reference) => reference.isRead());
      if (!read || rereads.length > 0) continue;
      if (read.from.variableScope !== variable.scope.variableScope) continue;
      const readStatement = statementOf(read.identifier as ESTree.Node);
      if (statementOf(declarator)?.parent !== readStatement?.parent) continue;
      initializers.set(read.identifier as ESTree.Node, declarator.init);
    }
  }
  return initializers;
}

function isInsideBranch(node: ESTree.Node): boolean {
  let child = node;
  for (let parent = node.parent; parent !== null; parent = parent.parent) {
    if (isFunctionNode(parent) || isClass(parent)) return false;
    if (parent.type === "ConditionalExpression" && parent.test !== child) return true;
    if (parent.type === "LogicalExpression" && parent.right === child) return true;
    child = parent;
  }
  return false;
}

function isCondition(node: ESTree.Node): boolean {
  const expression = unwrap(node);
  if (isNegation(expression) || isLogicalDecision(expression)) return true;
  if (expression.type !== "BinaryExpression") return false;
  return ["==", "!=", "===", "!==", "<", ">", "<=", ">=", "instanceof", "in"].includes(
    expression.operator,
  );
}

export function branchInitializer(
  constants: Map<ESTree.Node, ESTree.Node>,
  read: ESTree.Node,
): ESTree.Node | undefined {
  const initializer = constants.get(read);
  if (!initializer || isCondition(initializer) || !isInsideBranch(read)) return undefined;
  return initializer;
}
