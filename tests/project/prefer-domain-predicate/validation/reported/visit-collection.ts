function visit(node) { if (['ArrayExpression', 'JSXElement'].includes(node.type)) processChildren(node); }
