import type { ESTree } from "@oxlint/plugins";
import { unwrap } from "../../expression.ts";

const readOnlyUnaryOperators = ["!", "typeof", "+", "-", "~", "void"];

function isReadOnlyUnary(node: ESTree.UnaryExpression): boolean {
  return readOnlyUnaryOperators.includes(node.operator);
}

function guardExit(node: ESTree.Statement): "throw" | "return" | undefined {
  const statement = node.type === "BlockStatement" ? node.body[0] : node;
  if (node.type === "BlockStatement" && node.body.length !== 1) return;
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
        if (!isRead(value.object)) return false;
        return !value.computed || isRead(value.property);
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
      case "CallExpression": {
        if (!isMembership(value)) return false;
        const callee = unwrap(value.callee);
        if (callee.type !== "MemberExpression" || !isRead(callee.object)) return false;
        return value.arguments.every(isRead);
      }
      default:
        return false;
    }
  };
  const isLocal = (declaration: ESTree.VariableDeclarator): boolean => {
    if (declaration.id.type !== "Identifier" || !declaration.init) return false;
    return isRead(declaration.init);
  };
  let rejectsInvalidInput = false;
  for (const statement of body.body) {
    switch (statement.type) {
      case "VariableDeclaration":
        if (statement.kind !== "const" || !statement.declarations.every(isLocal)) return false;
        break;
      case "IfStatement": {
        if (statement.alternate || !isRead(statement.test)) return false;
        const exit = guardExit(statement.consequent);
        if (!exit) return false;
        if (exit === "throw") rejectsInvalidInput = true;
        break;
      }
      case "ReturnStatement":
        if (statement.argument !== null || statement !== body.body.at(-1)) return false;
        break;
      default:
        return false;
    }
  }
  return rejectsInvalidInput;
}
