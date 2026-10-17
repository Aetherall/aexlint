const kinds = ['Array', 'Object'] as const; const alias = kinds; if (alias.includes(node.type)) use(node);
