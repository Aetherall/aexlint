const policy = { [method](node) { return ['Array', 'Object'].includes(node.type); } };
