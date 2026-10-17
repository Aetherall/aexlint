const isContainer = ((node) => ['Array', 'Object'].includes(node.type)) satisfies Predicate;
