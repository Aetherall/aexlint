import type { ESTree } from "@oxlint/plugins";
import { unwrap } from "../../expression.ts";

const readOnlyUnaryOperators = ["!", "typeof", "+", "-", "~", "void"];

function isReadOnlyUnary(node: ESTree.UnaryExpression): boolean {
  return readOnlyUnaryOperators.includes(node.operator);
}

export function soleStatement(node: ESTree.Statement): ESTree.Statement | undefined {
  if (node.type !== "BlockStatement") return node;
  if (node.body.length === 1) return node.body[0];
}

function guardExit(node: ESTree.Statement): "throw" | "return" | undefined {
  const statement = soleStatement(node);
  if (statement?.type === "ThrowStatement") return "throw";
  if (statement?.type === "ReturnStatement" && statement.argument === null) return "return";
}

export function isValidationBody(
  body: ESTree.BlockStatement,
  isMembership: (node: ESTree.CallExpression) => boolean,
): boolean {
  const isRead = (node: ESTree.Node): boolean => {
    const value = unwrap(node);
    switch (value.type) {
      case "Identifier":
      case "ThisExpression":
      case "Literal":
        return true;
      case "MemberExpression":
        return isRead(value.object) && (!value.computed || isRead(value.property));
      case "UnaryExpression":
        return isReadOnlyUnary(value) && isRead(value.argument);
      case "BinaryExpression":
      case "LogicalExpression":
        return isRead(value.left) && isRead(value.right);
      case "ConditionalExpression":
        return isRead(value.test) && isRead(value.consequent) && isRead(value.alternate);
      case "ArrayExpression":
        return value.elements.every((element) => element === null || isRead(element));
      case "TemplateLiteral":
        return value.expressions.every(isRead);
      case "CallExpression":
        return isMembershipRead(value);
      default:
        return false;
    }
  };
  const isMembershipRead = (call: ESTree.CallExpression): boolean => {
    if (!isMembership(call)) return false;
    const callee = unwrap(call.callee);
    if (callee.type !== "MemberExpression" || !isRead(callee.object)) return false;
    return call.arguments.every(isRead);
  };
  const isLocal = (declaration: ESTree.VariableDeclarator): boolean => {
    if (declaration.id.type !== "Identifier" || !declaration.init) return false;
    return isRead(declaration.init);
  };
  const guardOf = (statement: ESTree.IfStatement): "throw" | "return" | undefined => {
    if (statement.alternate || !isRead(statement.test)) return;
    return guardExit(statement.consequent);
  };
  const isValidationStep = (statement: ESTree.Statement): boolean => {
    switch (statement.type) {
      case "VariableDeclaration":
        return statement.kind === "const" && statement.declarations.every(isLocal);
      case "IfStatement":
        return guardOf(statement) !== undefined;
      case "ReturnStatement":
        return statement.argument === null && statement === body.body.at(-1);
      default:
        return false;
    }
  };
  if (!body.body.every(isValidationStep)) return false;
  return body.body.some(
    (statement) => statement.type === "IfStatement" && guardOf(statement) === "throw",
  );
}
