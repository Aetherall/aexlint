const policy = { ['container'](node) { return ['Array', 'Object'].includes(node.type); } };
