function visit(node) { if (node.type === 'LogicalExpression' && node.operator !== '??') count(node); }
